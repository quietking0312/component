package mlog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
)

var _logger *zap.Logger

const (
	FormatJSON = "json"
	FormatText = "text"
)

type LogConfig struct {
	LogPath    string // 日志路径, 空将输出控制台
	LogLevel   string // 日志等级
	Compress   bool   // 压缩日志
	MaxSize    int    // log size (M)
	MaxAge     int    // 日志保存时间 (day)
	MaxBackups int    // 日志保存文件数
	Format     string // 日志类型 text or json
}

func defaultOption() *LogConfig {
	return &LogConfig{
		LogPath:    "",
		MaxSize:    10, // 单位MB
		Compress:   true,
		MaxAge:     90,
		MaxBackups: 20,
		LogLevel:   "debug",
		Format:     FormatText,
	}
}

type Option func(cfg *LogConfig)

func getZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

func newLogWriter(logCfg *LogConfig) io.Writer {
	if logCfg.LogPath == "" || logCfg.LogPath == "-" {
		return os.Stdout
	}
	return &lumberjack.Logger{
		Filename: logCfg.LogPath,
		// 单位（MB） 默认 100
		MaxSize: logCfg.MaxSize,
		// 保留日志的天数
		MaxAge: logCfg.MaxAge,
		// 保留日志最大数量
		MaxBackups: logCfg.MaxBackups,
		// 备份文件时间是否是本地时间， 默认是UTC
		LocalTime: false,
		// 是否使用 gzip 进行文件压缩
		Compress: logCfg.Compress,
	}
}

func NewLoggerCore(logCfg *LogConfig) zapcore.Core {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		FunctionKey:    "func",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000Z0700"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	var encoder zapcore.Encoder
	if logCfg.Format == FormatJSON {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	return zapcore.NewCore(encoder,
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(newLogWriter(logCfg))),
		zap.NewAtomicLevelAt(getZapLevel(logCfg.LogLevel)))
}

func InitLog(opts ...Option) error {
	logCfg := defaultOption()
	for _, opt := range opts {
		opt(logCfg)
	}
	_logger = zap.New(NewLoggerCore(logCfg),
		//zap.AddCaller(),
		//zap.AddCallerSkip(1),
		//zap.AddStacktrace(getZapLevel(logCfg.LogLevel)),
		zap.Development())
	return nil
}

// GetLogger 从现有的 _logger 对象拷贝一个日志对象
func GetLogger(opts ...zap.Option) *zap.Logger {
	return _logger.WithOptions(opts...)
}

func Sync() error {
	return _logger.Sync()
}

func Debug(msg string, fields ...zap.Field) {
	_logger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	_logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	_logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	_logger.Error(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	_logger.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	_logger.Fatal(msg, fields...)
}
