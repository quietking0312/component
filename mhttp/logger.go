package mhttp

import "log"

// Logger 日志接口，统一使用 slog 风格。
// 业务方可传入 *slog.Logger、zap.SugaredLogger、logrus 或任意自定义实现。
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

// NopLogger 返回一个空的 Logger，用于关闭日志或单元测试。
func NopLogger() Logger { return &nopLogger{} }

type nopLogger struct{}

func (n *nopLogger) Debug(string, ...any) {}
func (n *nopLogger) Info(string, ...any)  {}
func (n *nopLogger) Warn(string, ...any)  {}
func (n *nopLogger) Error(string, ...any) {}
