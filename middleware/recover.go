package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quietking0312/component/mlog"
	"go.uber.org/zap"
)

// Recover 返回 Gin 的 panic 恢复中间件
//
// 当 handler 中发生 panic 时，会调用 recoveryFunc 进行恢复处理（如返回 500 错误），
// 同时将 panic 堆栈信息记录到日志中。
//
// 使用示例：
//
//	router.Use(middleware.Recover(func(c *gin.Context, err any) {
//	    c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
//	}))
func Recover(recoveryFunc gin.RecoveryFunc) gin.HandlerFunc {
	return gin.RecoveryWithWriter(&panicLogger{}, recoveryFunc)
}

// panicLogger 实现 io.Writer 接口，用于捕获 gin.RecoveryWithWriter 输出的 panic 信息
// 内部将字节流转换为日志记录
type panicLogger struct{}

func (p *panicLogger) Write(b []byte) (n int, err error) {
	msg := string(b)
	mlog.Error("panic recovered", zap.String("stack", msg))
	return len(b), nil
}

// Recovery 返回一个标准库的 http.Handler 中间件，用于恢复 panic
//
// 适用于非 Gin 框架的 HTTP 服务，发生 panic 时记录日志并返回 500 错误
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				mlog.Error("panic recovered",
					zap.Any("error", rec),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
				)
				http.Error(w, fmt.Sprintf("Internal Server Error: %v", rec), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
