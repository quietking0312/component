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

func (l *StdLogger) Debugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

func (l *StdLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (l *StdLogger) Warnf(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

func (l *StdLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}
