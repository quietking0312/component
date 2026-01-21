package mpubsub

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type WriteSnapshot[T any] struct {
	Key   string
	Write WriteIface[T]
}

type WriteIface[T any] interface {
	Write(T) error
	Close() error
	ID() string
}

// 订阅组

type SubGroup[T any] struct {
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

type ChannelOption[T any] func(group *SubGroup[T])

func withChannelLogger[T any](logger LoggerIface) ChannelOption[T] {
	return func(g *SubGroup[T]) {
		g.logger = logger
	}
}

func NewSubGroup[T any](id string, opts ...ChannelOption[T]) *SubGroup[T] {
	ctx, cancel := context.WithCancel(context.Background())
	ch := &SubGroup[T]{
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
func (c *SubGroup[T]) Register(writer WriteIface[T]) error {
	if _, exists := c.group.Load(writer.ID()); exists {
		c.group.Delete(writer.ID())
	}
	c.group.Store(writer.ID(), writer)
	return nil
}

func (c *SubGroup[T]) ID() string {
	return c.id
}

func (c *SubGroup[T]) Unregister(key string) error {
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

func (c *SubGroup[T]) start() {
	for i := 0; i < 10; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}
	c.wg.Add(1)
	go c.cleanupWorker()
}

func (c *SubGroup[T]) Write(m T) error {
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

func (c *SubGroup[T]) worker(id int) {
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

func (c *SubGroup[T]) distributeMessage(msg T, workerId int) {
	var subscribers []WriteSnapshot[T]
	c.group.Range(func(key, value any) bool {
		subscribers = append(subscribers, WriteSnapshot[T]{Key: key.(string), Write: value.(WriteIface[T])})
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
			go func(s WriteSnapshot[T]) {
				defer wg.Done()
				c.writeToSubscriber(s.Key, s.Write, msg)
			}(sub)
		}
	default:
		c.batchDistribute(subscribers, msg)
	}
}

func (c *SubGroup[T]) batchDistribute(
	subscribers []WriteSnapshot[T], msg T) {
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
			go func(s WriteSnapshot[T]) {
				defer wg.Done()
				c.writeToSubscriber(s.Key, s.Write, msg)
			}(sub)
		}
		wg.Wait()
	}

}

func (c *SubGroup[T]) WriteToSubscriber(key string, msg T) {
	w, ok := c.group.Load(key)
	if !ok {
		return
	}
	writer := w.(WriteIface[T])
	c.writeToSubscriber(key, writer, msg)
}

// 写入单个订阅者
func (c *SubGroup[T]) writeToSubscriber(key string, writer WriteIface[T], msg T) {
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

func (c *SubGroup[T]) cleanupWorker() {
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

func (c *SubGroup[T]) Close() error {
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
