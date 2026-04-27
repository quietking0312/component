package mevent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Event is an in-process application event.
type Event struct {
	Name string    // event identifier, e.g. "user.created"
	Data any       // payload
	Time time.Time // emission time
}

// Handler processes events.
type Handler func(ctx context.Context, event Event) error

// ErrorStrategy defines how Publish handles handler errors.
type ErrorStrategy int

const (
	// StopOnError stops executing remaining handlers on first error.
	StopOnError ErrorStrategy = iota
	// ContinueOnError continues executing all handlers, collecting errors.
	ContinueOnError
)

// Middleware wraps handlers with cross-cutting concerns (logging, recovery, metrics, etc.).
type Middleware func(next Handler) Handler

// Bus is an in-process event bus for decoupling module communication.
// Zero-value Bus is NOT usable; always construct with New().
type Bus struct {
	mu          sync.RWMutex
	handlers    map[string][]Handler
	strategy    ErrorStrategy
	middlewares []Middleware
}

// Option configures a Bus.
type Option func(*Bus)

// New creates a new event bus.
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

// WithErrorStrategy sets the error handling strategy for Publish.
func WithErrorStrategy(s ErrorStrategy) Option {
	return func(b *Bus) {
		b.strategy = s
	}
}

// Use appends middleware to the bus. Middleware is applied to all handlers
// registered after the call. To apply to all handlers, call Use before Subscribe.
func (b *Bus) Use(mw ...Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middlewares = append(b.middlewares, mw...)
}

// Subscribe registers a handler for events with the given name.
// Handlers are executed in registration order.
func (b *Bus) Subscribe(name string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], b.wrap(h))
}

// wrap applies middlewares to a handler.
func (b *Bus) wrap(h Handler) Handler {
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		h = b.middlewares[i](h)
	}
	return h
}

// Publish synchronously dispatches an event to all subscribers of e.Name.
// If no subscribers exist, it returns nil immediately.
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

// PublishAsync dispatches an event asynchronously in a new goroutine.
// Errors are silently dropped; use Publish if you need error handling.
func (b *Bus) PublishAsync(ctx context.Context, e Event) {
	go func() {
		_ = b.Publish(ctx, e)
	}()
}

// HasSubscribers reports whether anyone is listening to the given event name.
func (b *Bus) HasSubscribers(name string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[name]) > 0
}

// Subscribers returns a snapshot of registered event names.
func (b *Bus) Subscribers() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	names := make([]string, 0, len(b.handlers))
	for n := range b.handlers {
		names = append(names, n)
	}
	return names
}
