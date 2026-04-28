//go:build !nolog
// +build !nolog

package msock

import (
	"log"
)

// StdLogger 使用标准库 log 的日志实现。
// 不再直接依赖 mlog 模块，如需使用 mlog，请在外部实现 Logger 接口并传入。
type StdLogger struct{}

// NewStdLogger 创建标准库日志记录器
func NewStdLogger() Logger {
	return &StdLogger{}
}

func (l *StdLogger) Debug(msg string, args ...any) {
	log.Printf("[DEBUG] "+msg, args...)
}

func (l *StdLogger) Info(msg string, args ...any) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *StdLogger) Warn(msg string, args ...any) {
	log.Printf("[WARN] "+msg, args...)
}

func (l *StdLogger) Error(msg string, args ...any) {
	log.Printf("[ERROR] "+msg, args...)
}
