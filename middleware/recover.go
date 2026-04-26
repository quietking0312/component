package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Logger 日志接口，用于解耦对具体日志库的依赖。
// 使用者可传入 zap.SugaredLogger、logrus、标准库 log 或自定义实现。
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

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
	return RecoverWithLogger(nil, recoveryFunc)
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
	if p.logger != nil {
		msg := string(b)
		p.logger.Error("panic recovered", "stack", msg)
	}
	return len(b), nil
}

// Recovery 返回一个标准库的 http.Handler 中间件，用于恢复 panic。
// 适用于非 Gin 框架的 HTTP 服务，发生 panic 时返回 500 错误。
// 注意：此版本不记录 panic 日志，如需日志请使用 RecoveryWithLogger。
func Recovery(next http.Handler) http.Handler {
	return RecoveryWithLogger(nil, next)
}

// RecoveryWithLogger 返回标准库的 http.Handler 中间件，支持通过 Logger 接口注入日志记录器。
// 发生 panic 时通过 logger 记录日志并返回 500 错误。
func RecoveryWithLogger(logger Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if logger != nil {
					logger.Error("panic recovered",
						"error", rec,
						"path", r.URL.Path,
						"method", r.Method,
					)
				}
				http.Error(w, fmt.Sprintf("Internal Server Error: %v", rec), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
