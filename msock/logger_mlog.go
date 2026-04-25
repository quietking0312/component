//go:build !nolog
// +build !nolog

package msock

import (
	"github.com/quietking0312/component/mlog"
)

// MLogLogger 使用 mlog 的日志实现
type MLogLogger struct{}

// NewMLogLogger 创建 mlog 日志器
func NewMLogLogger() Logger {
	return &MLogLogger{}
}

func (l *MLogLogger) Debugf(format string, args ...interface{}) {
	mlog.Debugf(format, args...)
}

func (l *MLogLogger) Infof(format string, args ...interface{}) {
	mlog.Infof(format, args...)
}

func (l *MLogLogger) Warnf(format string, args ...interface{}) {
	mlog.Warnf(format, args...)
}

func (l *MLogLogger) Errorf(format string, args ...interface{}) {
	mlog.Errorf(format, args...)
}
