package mevent

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
)

// Recovery returns a middleware that recovers from panics in handlers.
// Recovered panics are logged via the provided logger function; if nil,
// log.Printf is used.
func Recovery(logger func(format string, v ...any)) Middleware {
	if logger == nil {
		logger = log.Printf
	}
	return func(next Handler) Handler {
		return func(ctx context.Context, event Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger("[mevent] panic in handler for %q: %v\n%s", event.Name, r, debug.Stack())
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			return next(ctx, event)
		}
	}
}

// Logger returns a middleware that logs event dispatching.
func Logger(logger func(format string, v ...any)) Middleware {
	if logger == nil {
		logger = log.Printf
	}
	return func(next Handler) Handler {
		return func(ctx context.Context, event Event) error {
			logger("[mevent] dispatching %q", event.Name)
			err := next(ctx, event)
			if err != nil {
				logger("[mevent] handler for %q failed: %v", event.Name, err)
			}
			return err
		}
	}
}

// Filter returns a middleware that only passes events matching the predicate.
func Filter(predicate func(event Event) bool) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, event Event) error {
			if !predicate(event) {
				return nil
			}
			return next(ctx, event)
		}
	}
}
