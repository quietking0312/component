package mlog

import (
	"fmt"

	"go.uber.org/zap"
)

// SlogAdapter 将 mlog（zap）适配为 slog 风格的 Logger 接口。
// 可用于注入到 mcron、mhttp、middleware、mtimewheel、msock、mpubsub 等模块。
type SlogAdapter struct{}

// NewSlogAdapter 创建一个新的 slog 风格适配器。
func NewSlogAdapter() *SlogAdapter {
	return &SlogAdapter{}
}

// Debug 输出 Debug 级别日志。
func (s *SlogAdapter) Debug(msg string, args ...any) {
	if len(args) == 0 {
		Debug(msg)
		return
	}
	Debug(msg, toZapFields(args)...)
}

// Info 输出 Info 级别日志。
func (s *SlogAdapter) Info(msg string, args ...any) {
	if len(args) == 0 {
		Info(msg)
		return
	}
	Info(msg, toZapFields(args)...)
}

// Warn 输出 Warn 级别日志。
func (s *SlogAdapter) Warn(msg string, args ...any) {
	if len(args) == 0 {
		Warn(msg)
		return
	}
	Warn(msg, toZapFields(args)...)
}

// Error 输出 Error 级别日志。
func (s *SlogAdapter) Error(msg string, args ...any) {
	if len(args) == 0 {
		Error(msg)
		return
	}
	Error(msg, toZapFields(args)...)
}

// toZapFields 将 slog 风格的 key-value 参数转换为 zap.Field。
// 支持 slog 的偶数 key-value 形式，也兼容奇数长度（最后一个值作为 extra）。
func toZapFields(args []any) []zap.Field {
	var fields []zap.Field
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			fields = append(fields, zap.Any("extra", args[i]))
			break
		}
		key := fmt.Sprint(args[i])
		fields = append(fields, zap.Any(key, args[i+1]))
	}
	return fields
}
