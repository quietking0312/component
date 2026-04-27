package mevent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Event 表示一个进程内应用事件。
type Event struct {
	Name string    // 事件标识，如 "user.created"
	Data any       // 负载数据
	Time time.Time // 触发时间
}

// Handler 处理事件的函数签名。
type Handler func(ctx context.Context, event Event) error

// ErrorStrategy 定义 Publish 遇到 handler 错误时的处理策略。
type ErrorStrategy int

const (
	// StopOnError 遇到第一个错误即停止执行后续 handler。
	StopOnError ErrorStrategy = iota
	// ContinueOnError 继续执行所有 handler，并收集错误。
	ContinueOnError
)

// Middleware 中间件，用于在 handler 外包裹横切关注点（日志、恢复、监控等）。
type Middleware func(next Handler) Handler

// Bus 进程内事件总线，用于解耦模块间通信。
// 零值 Bus 不可用，必须通过 New() 构造。
type Bus struct {
	mu          sync.RWMutex
	handlers    map[string][]Handler
	strategy    ErrorStrategy
	middlewares []Middleware
}

// Option Bus 的配置选项。
type Option func(*Bus)

// New 创建一个新的事件总线。
func New(opts ...Option) *Bus {
	b := &Bus{
		handlers: make(map[string][]Handler),
		strategy: StopOnError,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// WithErrorStrategy 设置 Publish 的错误处理策略。
func WithErrorStrategy(s ErrorStrategy) Option {
	return func(b *Bus) {
		b.strategy = s
	}
}

// Use 向总线追加中间件。中间件仅对后续注册的 handler 生效；
// 若要应用到所有 handler，请在 Subscribe 之前调用 Use。
func (b *Bus) Use(mw ...Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middlewares = append(b.middlewares, mw...)
}

// Subscribe 注册一个事件订阅者，按注册顺序执行。
func (b *Bus) Subscribe(name string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], b.wrap(h))
}

// wrap 将中间件应用到 handler。
func (b *Bus) wrap(h Handler) Handler {
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		h = b.middlewares[i](h)
	}
	return h
}

// Publish 同步分发事件给 e.Name 的所有订阅者。
// 若无订阅者，立即返回 nil。
func (b *Bus) Publish(ctx context.Context, e Event) error {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}

	b.mu.RLock()
	hs := make([]Handler, len(b.handlers[e.Name]))
	copy(hs, b.handlers[e.Name])
	b.mu.RUnlock()

	if len(hs) == 0 {
		return nil
	}

	var errs []error
	for _, h := range hs {
		if err := h(ctx, e); err != nil {
			if b.strategy == StopOnError {
				return err
			}
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("event %q: %d/%d handlers failed: %v", e.Name, len(errs), len(hs), errs)
	}
	return nil
}

// PublishAsync 在独立 goroutine 中异步分发事件。
// 错误会被静默丢弃；如需错误处理，请使用 Publish。
func (b *Bus) PublishAsync(ctx context.Context, e Event) {
	go func() {
		_ = b.Publish(ctx, e)
	}()
}

// HasSubscribers 报告指定事件是否存在订阅者。
func (b *Bus) HasSubscribers(name string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[name]) > 0
}

// Subscribers 返回当前已注册事件名称的快照。
func (b *Bus) Subscribers() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	names := make([]string, 0, len(b.handlers))
	for n := range b.handlers {
		names = append(names, n)
	}
	return names
}
