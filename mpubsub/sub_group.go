package mpubsub

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type HandlerIface[T any] interface {
	ID() string
	Write(T) error
}

// 订阅子组
type GroupIface[T any] interface {
	GetSubList(T) []HandlerIface[T] // 根据消息 提供 具体要推送的 成员镜像
	Delete(key string)
	Set(iface HandlerIface[T])
}

type Worker[T any] struct {
	id         int
	g          *SubGroup[T]
	stopCh     chan struct{}
	lastActive time.Time
	mu         sync.RWMutex
}

func (w *Worker[T]) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done(): // 收到整个组的关闭信号
			return
		case <-w.stopCh: // 收到单独关闭信号

			return
		case job, ok := <-w.g.msgChan:
			if !ok {
				return
			}
			w.mu.Lock()
			w.lastActive = time.Now()
			w.mu.Unlock()
			w.g.distributeMessage(job, w.id)
		}
	}
}

func (w *Worker[T]) isIdle() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	idleTime := time.Since(w.lastActive)
	return idleTime > time.Minute
}

var _ SubGroupIface[any] = (*SubGroup[any])(nil)

// 订阅组
// 提供了 分片批量写入
type SubGroup[T any] struct {
	id        string
	group     GroupIface[T]
	msgChan   chan T
	closeChan chan struct{}
	closed    atomic.Bool
	wg        sync.WaitGroup
	logger    Logger
	config    SubGroupOption
	ctx       context.Context
	cancel    context.CancelFunc
	metrics   *Metrics
	workerSem chan struct{} //worker 容量
	workers   []*Worker[T]
	workersMu sync.RWMutex
}

type ChannelOption[T any] func(group *SubGroup[T])

func WithSubGroupLogger[T any](logger Logger) ChannelOption[T] {
	return func(g *SubGroup[T]) {
		g.logger = logger
	}
}

func NewSubGroup[T any](id string, group GroupIface[T], opts ...ChannelOption[T]) *SubGroup[T] {
	ctx, cancel := context.WithCancel(context.Background())
	ch := &SubGroup[T]{
		id:      id,
		msgChan: make(chan T, 100),
		logger:  _log,
		ctx:     ctx,
		cancel:  cancel,
		group:   group,
		config: SubGroupOption{
			WorkNum:    10,
			MinWorkers: 2,
			MaxWorkers: 50,
			RetryCount: 3,
			RetryDelay: 3 * time.Second,
		},
		metrics: &Metrics{},
	}
	for _, opt := range opts {
		opt(ch)
	}
	ch.workerSem = make(chan struct{}, ch.config.MaxWorkers)
	ch.start()
	go ch.autoScaleWorkers()
	return ch
}

func (c *SubGroup[T]) ID() string {
	return c.id
}

func (c *SubGroup[T]) start() {
	for i := 0; i < c.config.WorkNum; i++ {
		c.workerSem <- struct{}{}
		c.worker(i)
	}
}

func (c *SubGroup[T]) autoScaleWorkers() {
	c.wg.Add(1)
	defer c.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.scaleWorkers()
		}
	}
}

func (c *SubGroup[T]) scaleWorkers() {
	queueLen := len(c.msgChan)
	currentWorkers := int(len(c.workers))
	if queueLen > 50 && currentWorkers < c.config.MaxWorkers {
		c.scaleUp()
	} else if queueLen < 10 && currentWorkers > c.config.MinWorkers {
		c.scaleDown()
	}
}

func (c *SubGroup[T]) scaleUp() {
	select {
	case c.workerSem <- struct{}{}:
		c.workersMu.Lock()
		workerId := int(c.metrics.ActiveWorkers.Load() + 1)
		c.workersMu.Unlock()
		c.worker(workerId)
	default:
		// 扩容达到上限
	}
}

func (c *SubGroup[T]) scaleDown() {
	c.workersMu.Lock()
	defer c.workersMu.Unlock()
	if len(c.workers) <= c.config.MinWorkers {
		return
	}

	newWorkers := make([]*Worker[T], 0, len(c.workers))
	idleCont := 0
	maxIdleToRemove := len(c.workers) - c.config.MinWorkers
	for _, worker := range c.workers {
		if idleCont < maxIdleToRemove && worker.isIdle() {
			select {
			case worker.stopCh <- struct{}{}:
				idleCont++
				continue
			default:
			}
		}
		newWorkers = append(newWorkers, worker)
	}
	c.workers = newWorkers
}

func (c *SubGroup[T]) Write(m T) error {
	if c.closed.Load() {
		return fmt.Errorf("groupClosed: %v", c.id)
	}
	select {
	case c.msgChan <- m:
		return nil
	case <-time.After(20 * time.Second):
		c.logger.Error(fmt.Sprintf("writeTimeout %v", c.id))
		return fmt.Errorf("writeTimeout")

	}
}

func (c *SubGroup[T]) worker(id int) {
	c.workersMu.Lock()
	w := &Worker[T]{
		id:         id,
		g:          c,
		stopCh:     make(chan struct{}, 1),
		lastActive: time.Now(),
	}
	c.workers = append(c.workers, w)
	c.workersMu.Unlock()
	c.wg.Add(1)
	go func() {
		c.metrics.ActiveWorkers.Add(1)
		c.logger.Info(fmt.Sprintf("subGroup:%s  worker: %d create", c.id, id))
		defer func() {
			c.workersMu.Lock()
			for i, worker := range c.workers {
				if worker == w {
					c.workers = append(c.workers[:i], c.workers[i+1:]...)
				}
			}
			c.workersMu.Unlock()
			c.wg.Done()
			c.metrics.ActiveWorkers.Add(-1)
			<-c.workerSem
			c.logger.Info(fmt.Sprintf("subGroup:%s  worker: %d end", c.id, id))
		}()
		w.start(c.ctx)
	}()
}

func (c *SubGroup[T]) distributeMessage(msg T, workerId int) {
	var subscribers = c.group.GetSubList(msg)
	if len(subscribers) == 0 {
		return
	}
	switch {
	case len(subscribers) == 1:
		c.writeToSubscriber(subscribers[0], msg)
	case len(subscribers) <= 10:
		var wg sync.WaitGroup
		for _, sub := range subscribers {
			wg.Add(1)
			go func(s HandlerIface[T]) {
				defer wg.Done()
				c.writeToSubscriber(s, msg)
			}(sub)
		}
	default:
		c.batchDistribute(subscribers, msg)
	}
}

func (c *SubGroup[T]) batchDistribute(
	subscribers []HandlerIface[T], msg T) {
	batchSize := 10
	for i := 0; i < len(subscribers); i += batchSize {
		end := i + batchSize
		if end > len(subscribers) {
			end = len(subscribers)
		}
		batch := subscribers[i:end]
		sem := make(chan struct{}, 20) // 限制并发数
		var wg sync.WaitGroup
		for _, sub := range batch {
			wg.Add(1)
			sem <- struct{}{}
			go func(s HandlerIface[T]) {
				defer wg.Done()
				defer func() { <-sem }()
				c.writeToSubscriber(s, msg)
			}(sub)
		}
		wg.Wait()
	}

}

// 写入单个订阅者
func (c *SubGroup[T]) writeToSubscriber(writer HandlerIface[T], msg T) {
	var lastErr error
	for i := 0; i < c.config.RetryCount; i++ {
		if i > 0 {
			time.Sleep(c.config.RetryDelay)
		}
		if err := writer.Write(msg); err != nil {
			lastErr = err
			continue
		}
		return
	}
	c.logger.Error(fmt.Sprintf("write err:%v", lastErr))
	c.group.Delete(writer.ID())
}

func (c *SubGroup[T]) Register(iface HandlerIface[T]) {
	c.group.Set(iface)
}

func (c *SubGroup[T]) UnRegister(id string) {
	c.group.Delete(id)
}

func (c *SubGroup[T]) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(c.msgChan)
	c.cancel()
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
		c.logger.Error(fmt.Sprintf("shutdown timeout %s", c.id))
		return fmt.Errorf("shutdown timeout")
	}
}
