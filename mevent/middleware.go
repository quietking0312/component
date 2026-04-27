package mevent

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
)

// Recovery 返回一个捕获 handler panic 的中间件。
// 捕获到的 panic 会通过传入的 logger 输出；若 logger 为 nil，则使用 log.Printf。
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

// Logger 返回一个记录事件分发过程的中间件。
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

// Filter 返回一个过滤中间件，仅让满足 predicate 的事件通过。
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
