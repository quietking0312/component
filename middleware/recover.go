package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Logger 日志接口，统一使用 slog 风格。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// stdLogLogger 默认使用标准库 log 输出。
type stdLogLogger struct{}

func (s *stdLogLogger) Debug(msg string, args ...any) { log.Println(append([]any{"[DEBUG]", msg}, args...)...) }
func (s *stdLogLogger) Info(msg string, args ...any)  { log.Println(append([]any{"[INFO]", msg}, args...)...) }
func (s *stdLogLogger) Warn(msg string, args ...any)  { log.Println(append([]any{"[WARN]", msg}, args...)...) }
func (s *stdLogLogger) Error(msg string, args ...any) { log.Println(append([]any{"[ERROR]", msg}, args...)...) }

// Recover 返回 Gin 的 panic 恢复中间件。
// 当 handler 中发生 panic 时，会调用 recoveryFunc 进行恢复处理（如返回 500 错误）。
// 注意：此版本不记录 panic 日志，如需日志请使用 RecoverWithLogger。
//
// 使用示例：
//
//	router.Use(middleware.Recover(func(c *gin.Context, err any) {
//	    c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
//	}))
func Recover(recoveryFunc gin.RecoveryFunc) gin.HandlerFunc {
	return RecoverWithLogger(&stdLogLogger{}, recoveryFunc)
}

// RecoverWithLogger 返回 Gin 的 panic 恢复中间件，支持通过 Logger 接口注入日志记录器。
// 当 handler 中发生 panic 时，会调用 recoveryFunc 进行恢复处理，
// 同时将 panic 堆栈信息通过注入的 logger 记录。
//
// 使用示例：
//
//	router.Use(middleware.RecoverWithLogger(logger, func(c *gin.Context, err any) {
//	    c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
//	}))
func RecoverWithLogger(logger Logger, recoveryFunc gin.RecoveryFunc) gin.HandlerFunc {
	return gin.RecoveryWithWriter(&panicLogger{logger: logger}, recoveryFunc)
}

// panicLogger 实现 io.Writer 接口，用于捕获 gin.RecoveryWithWriter 输出的 panic 信息。
// 内部通过注入的 Logger 接口将字节流转换为日志记录，避免直接依赖任何具体日志库。
type panicLogger struct {
	logger Logger
}

func (p *panicLogger) Write(b []byte) (n int, err error) {
	p.logger.Error("panic recovered", "stack", string(b))
	return len(b), nil
}

// Recovery 返回一个标准库的 http.Handler 中间件，用于恢复 panic。
// 适用于非 Gin 框架的 HTTP 服务，发生 panic 时返回 500 错误。
// 注意：此版本不记录 panic 日志，如需日志请使用 RecoveryWithLogger。
func Recovery(next http.Handler) http.Handler {
	return RecoveryWithLogger(&stdLogLogger{}, next)
}

// RecoveryWithLogger 返回标准库的 http.Handler 中间件，支持通过 Logger 接口注入日志记录器。
// 发生 panic 时通过 logger 记录日志并返回 500 错误。
func RecoveryWithLogger(logger Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"error", rec,
					"path", r.URL.Path,
					"method", r.Method,
				)
				http.Error(w, fmt.Sprintf("Internal Server Error: %v", rec), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
