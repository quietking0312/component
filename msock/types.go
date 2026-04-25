package msock

import (
	"context"
	"net"
	"time"
)

// ConnType 连接类型
type ConnType string

const (
	ConnTypeTCP       ConnType = "tcp"
	ConnTypeWebSocket ConnType = "websocket"
	ConnTypeKCP       ConnType = "kcp"
)

// Message 消息接口
type Message interface {
	// RouteID 返回路由ID，用于消息路由
	RouteID() uint32
	// Data 返回消息数据
	Data() []byte
	// SetData 设置消息数据
	SetData([]byte)
}

// Conn 连接接口，统一封装 TCP/WebSocket/KCP
type Conn interface {
	// ID 返回连接唯一标识
	ID() string
	// Type 返回连接类型
	Type() ConnType
	// LocalAddr 返回本地地址
	LocalAddr() net.Addr
	// RemoteAddr 返回远程地址
	RemoteAddr() net.Addr
	// Send 发送消息
	Send(msg Message) error
	// SendBytes 发送原始字节数据
	SendBytes(data []byte) error
	// Close 关闭连接
	Close() error
	// IsClosed 检查连接是否已关闭
	IsClosed() bool
	// SetReadDeadline 设置读取超时
	SetReadDeadline(t time.Time) error
	// SetWriteDeadline 设置写入超时
	SetWriteDeadline(t time.Time) error
	// Context 返回连接的上下文
	Context() context.Context
	// SetValue 存储键值对到连接上下文
	SetValue(key, value interface{})
	// GetValue 从连接上下文获取值
	GetValue(key interface{}) (interface{}, bool)
}

// Codec 编解码器接口，用于自定义解包
type Codec interface {
	// Encode 编码消息为字节流
	Encode(msg Message) ([]byte, error)
	// Decode 解码字节流为消息
	// 返回消息和已解码的字节数
	Decode(data []byte) (Message, int, error)
	// MaxPacketSize 返回允许的最大包大小
	MaxPacketSize() int
}

// Handler 消息处理器
type Handler func(conn Conn, msg Message)

// Logger 日志接口，允许自定义日志输出
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// defaultLogger 默认日志实现（使用标准库)
type defaultLogger struct{}

func (l *defaultLogger) Debugf(format string, args ...interface{}) {}
func (l *defaultLogger) Infof(format string, args ...interface{})  {}
func (l *defaultLogger) Warnf(format string, args ...interface{})  {}
func (l *defaultLogger) Errorf(format string, args ...interface{}) {}

// nopLogger 空日志实现
type nopLogger struct{}

func (l *nopLogger) Debugf(format string, args ...interface{}) {}
func (l *nopLogger) Infof(format string, args ...interface{})  {}
func (l *nopLogger) Warnf(format string, args ...interface{})  {}
func (l *nopLogger) Errorf(format string, args ...interface{}) {}

// ServerOption 服务器配置选项
type ServerOption func(*ServerConfig)

// ServerConfig 服务器配置
type ServerConfig struct {
	// 地址
	Address string
	// 连接类型
	ConnType ConnType
	// 编解码器
	Codec Codec
	// 日志器
	Logger Logger
	// 读取缓冲区大小
	ReadBufferSize int
	// 写入缓冲区大小
	WriteBufferSize int
	// 最大连接数
	MaxConnections int
	// 读超时
	ReadTimeout time.Duration
	// 写超时
	WriteTimeout time.Duration
	// 心跳间隔
	HeartbeatInterval time.Duration
	// 心跳超时
	HeartbeatTimeout time.Duration
}

// DefaultServerConfig 返回默认服务器配置
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:           ":8080",
		ConnType:          ConnTypeTCP,
		Logger:            &defaultLogger{},
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		MaxConnections:    10000,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		HeartbeatTimeout:  90 * time.Second,
	}
}

// WithAddress 设置地址
func WithAddress(addr string) ServerOption {
	return func(c *ServerConfig) {
		c.Address = addr
	}
}

// WithConnType 设置连接类型
func WithConnType(t ConnType) ServerOption {
	return func(c *ServerConfig) {
		c.ConnType = t
	}
}

// WithCodec 设置编解码器
func WithCodec(codec Codec) ServerOption {
	return func(c *ServerConfig) {
		c.Codec = codec
	}
}

// WithLogger 设置日志器
func WithLogger(logger Logger) ServerOption {
	return func(c *ServerConfig) {
		c.Logger = logger
	}
}

// WithReadBufferSize 设置读取缓冲区大小
func WithReadBufferSize(size int) ServerOption {
	return func(c *ServerConfig) {
		c.ReadBufferSize = size
	}
}

// WithWriteBufferSize 设置写入缓冲区大小
func WithWriteBufferSize(size int) ServerOption {
	return func(c *ServerConfig) {
		c.WriteBufferSize = size
	}
}

// WithMaxConnections 设置最大连接数
func WithMaxConnections(max int) ServerOption {
	return func(c *ServerConfig) {
		c.MaxConnections = max
	}
}

// WithReadTimeout 设置读超时
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(c *ServerConfig) {
		c.ReadTimeout = timeout
	}
}

// WithWriteTimeout 设置写超时
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(c *ServerConfig) {
		c.WriteTimeout = timeout
	}
}

// WithHeartbeat 设置心跳参数
func WithHeartbeat(interval, timeout time.Duration) ServerOption {
	return func(c *ServerConfig) {
		c.HeartbeatInterval = interval
		c.HeartbeatTimeout = timeout
	}
}
