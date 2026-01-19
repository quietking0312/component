package mpubsub

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type WriteSnapshot[K comparable, T any] struct {
	Key   K
	Write WriteIface[T]
}

type WriteIface[T any] interface {
	Write(T) error
	Close() error
}

// 订阅组

type SubGroup[K comparable, T any] struct {
	id        string
	group     sync.Map // K => WriteIface[T]
	msgChan   chan T
	closeChan chan struct{}
	closed    atomic.Bool
	wg        sync.WaitGroup
	logger    LoggerIface
	ctx       context.Context
	cancel    context.CancelFunc
}

type ChannelOption[K comparable, T any] func(group *SubGroup[K, T])

func withChannelLogger[K comparable, T any](logger LoggerIface) ChannelOption[K, T] {
	return func(g *SubGroup[K, T]) {
		g.logger = logger
	}
}

func NewSubChannel[K comparable, T any](id string, opts ...ChannelOption[K, T]) *SubGroup[K, T] {
	ctx, cancel := context.WithCancel(context.Background())
	ch := &SubGroup[K, T]{
		id:      id,
		msgChan: make(chan T, 100),
		logger:  _log,
		ctx:     ctx,
		cancel:  cancel,
	}
	for _, opt := range opts {
		opt(ch)
	}
	ch.start()
	return ch
}
func (c *SubGroup[K, T]) Register(key K, writer WriteIface[T]) error {
	if _, exists := c.group.Load(key); exists {
		return nil
	}
	c.group.Store(key, writer)
	return nil
}

func (c *SubGroup[K, T]) ID() string {
	return c.id
}

func (c *SubGroup[K, T]) Unregister(key K) error {
	if writer, exists := c.group.LoadAndDelete(key); exists {
		// 关闭写入器
		go func() {
			if err := writer.(WriteIface[T]).Close(); err != nil {
				c.logger.Error(fmt.Errorf("failed to close err: %v", err))
			}
		}()
		return nil
	}
	return nil
}

func (c *SubGroup[K, T]) start() {
	for i := 0; i < 10; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}
	c.wg.Add(1)
	go c.cleanupWorker()
}

func (c *SubGroup[K, T]) Write(m T) error {
	if c.closed.Load() {
		return fmt.Errorf("groupClosed: %v", c.id)
	}
	if len(c.msgChan) >= cap(c.msgChan) {
		return nil
	}
	select {
	case c.msgChan <- m:
		return nil
	case <-time.After(20 * time.Second):
		c.logger.Error(fmt.Errorf("writeTimeout %v", c.id))
		return fmt.Errorf("writeTimeout")

	}
}

func (c *SubGroup[K, T]) worker(id int) {
	defer c.wg.Done()
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.msgChan:
			if !ok {
				return
			}
			c.distributeMessage(msg, id)
		}
	}
}

func (c *SubGroup[K, T]) distributeMessage(msg T, workerId int) {
	var subscribers []WriteSnapshot[K, T]
	c.group.Range(func(key, value any) bool {
		subscribers = append(subscribers, WriteSnapshot[K, T]{Key: key.(K), Write: value.(WriteIface[T])})
		return true
	})
	if len(subscribers) == 0 {
		return
	}
	switch {
	case len(subscribers) == 1:
		c.writeToSubscriber(subscribers[0].Key, subscribers[0].Write, msg)
	case len(subscribers) <= 10:
		var wg sync.WaitGroup
		for _, sub := range subscribers {
			wg.Add(1)
			go func(s WriteSnapshot[K, T]) {
				defer wg.Done()
				c.writeToSubscriber(s.Key, s.Write, msg)
			}(sub)
		}
	default:
		c.batchDistribute(subscribers, msg)
	}
}

func (c *SubGroup[K, T]) batchDistribute(
	subscribers []WriteSnapshot[K, T], msg T) {
	batchSize := 10
	for i := 0; i < len(subscribers); i += batchSize {
		end := i + batchSize
		if end > len(subscribers) {
			end = len(subscribers)
		}
		batch := subscribers[i:end]
		var wg sync.WaitGroup
		for _, sub := range batch {
			wg.Add(1)
			go func(s WriteSnapshot[K, T]) {
				defer wg.Done()
				c.writeToSubscriber(s.Key, s.Write, msg)
			}(sub)
		}
		wg.Wait()
	}

}

// 写入单个订阅者
func (c *SubGroup[K, T]) writeToSubscriber(key K, writer WriteIface[T], msg T) {
	var lastErr error
	for i := 0; i < 10; i++ {
		if i > 1 {
			time.Sleep(10 * time.Second)
		}
		if err := writer.Write(msg); err != nil {
			lastErr = err
			continue
		}
		return
	}
	c.logger.Error(fmt.Errorf("write err:%v", lastErr))
	c.group.Delete(key)
}

func (c *SubGroup[K, T]) cleanupWorker() {
	defer c.wg.Done()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *SubGroup[K, T]) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.cancel()
	close(c.msgChan)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		c.logger.Info(fmt.Sprintf("subGroup closed %s", c.id))
		return nil
	case <-time.After(10 * time.Second):
		c.logger.Error(fmt.Errorf("shutdown timeout %s", c.id))
		return fmt.Errorf("shutdown timeout")
	}
}
