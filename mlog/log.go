package mlog

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// 全局 logger 实例
	logger *zap.Logger
	// 全局 sugar logger 实例
	sugarLogger *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	// 日志级别: debug, info, warn, error, fatal
	Level string
	// 日志格式: json, console
	Format string
	// 日志输出路径
	Output string
	// 是否输出到控制台
	Console bool
	// 日志文件最大大小(MB)
	MaxSize int
	// 保留旧文件的最大个数
	MaxBackups int
	// 保留旧文件的最大天数
	MaxAge int
	// 是否压缩旧文件
	Compress bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "console",
		Output:     "./logs/app.log",
		Console:    true,
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}
}

// Init 初始化日志
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 设置日志级别
	level := getLogLevel(cfg.Level)

	// 设置编码器
	encoder := getEncoder(cfg.Format)

	// 设置输出
	var cores []zapcore.Core

	// 文件输出
	if cfg.Output != "" {
		// 确保日志目录存在
		dir := filepath.Dir(cfg.Output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// 使用 lumberjack 实现日志切割
		writeSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Output,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})
		core := zapcore.NewCore(encoder, writeSyncer, level)
		cores = append(cores, core)
	}

	// 控制台输出
	if cfg.Console {
		consoleEncoder := getEncoder("console")
		consoleSyncer := zapcore.AddSync(os.Stdout)
		core := zapcore.NewCore(consoleEncoder, consoleSyncer, level)
		cores = append(cores, core)
	}

	// 创建 logger
	if len(cores) == 0 {
		cores = append(cores, zapcore.NewNopCore())
	}
	core := zapcore.NewTee(cores...)
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	sugarLogger = logger.Sugar()
	return nil
}

// getLogLevel 获取日志级别
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// getEncoder 获取编码器
func getEncoder(format string) zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if format == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// customTimeEncoder 自定义时间格式
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// ========== Logger 方法 ==========

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Logger().Debug(msg, fields...)
}

// Debugf 调试日志（格式化）
func Debugf(template string, args ...interface{}) {
	Sugar().Debugf(template, args...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Logger().Info(msg, fields...)
}

// Infof 信息日志（格式化）
func Infof(template string, args ...interface{}) {
	Sugar().Infof(template, args...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Logger().Warn(msg, fields...)
}

// Warnf 警告日志（格式化）
func Warnf(template string, args ...interface{}) {
	Sugar().Warnf(template, args...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Logger().Error(msg, fields...)
}

// Errorf 错误日志（格式化）
func Errorf(template string, args ...interface{}) {
	Sugar().Errorf(template, args...)
}

// Fatal 致命日志
func Fatal(msg string, fields ...zap.Field) {
	Logger().Fatal(msg, fields...)
}

// Fatalf 致命日志（格式化）
func Fatalf(template string, args ...interface{}) {
	Sugar().Fatalf(template, args...)
}

// Panic 恐慌日志
func Panic(msg string, fields ...zap.Field) {
	Logger().Panic(msg, fields...)
}

// Panicf 恐慌日志（格式化）
func Panicf(template string, args ...interface{}) {
	Sugar().Panicf(template, args...)
}

// Sync 同步日志缓冲区
func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}

// Logger 获取原始 logger 实例
func Logger() *zap.Logger {
	if logger == nil {
		return zap.NewNop()
	}
	return logger
}

// Sugar 获取 sugar logger 实例
func Sugar() *zap.SugaredLogger {
	if sugarLogger == nil {
		return zap.NewNop().Sugar()
	}
	return sugarLogger
}

// With 创建带有字段的 logger
func With(fields ...zap.Field) *zap.Logger {
	return Logger().With(fields...)
}

// WithOptions 创建带有选项的 logger
func WithOptions(opts ...zap.Option) *zap.Logger {
	return Logger().WithOptions(opts...)
}

// Named 创建带有名称的 logger
func Named(name string) *zap.Logger {
	return Logger().Named(name)
}
