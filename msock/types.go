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
	ConnTypeGWS       ConnType = "gws"
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

// Codec 编解码器接口，采用两阶段解码：先解 header 得到 body 长度，再读 body。
type Codec interface {
	// Encode 编码消息为字节流
	Encode(msg Message) ([]byte, error)
	// HeaderSize 返回固定 header 字节数
	HeaderSize() int
	// DecodeHeader 解析 header，返回 routeID 和 body 长度
	DecodeHeader(header []byte) (routeID uint32, bodyLen int, err error)
	// DecodeBody 将 body 字节解析为消息
	DecodeBody(routeID uint32, body []byte) (Message, error)
	// MaxPacketSize 返回允许的最大包大小（header + body）
	MaxPacketSize() int
}

// Handler 消息处理器
type Handler func(conn Conn, msg Message)

// Logger 日志接口，统一使用 slog 风格。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// defaultLogger 默认日志实现（使用标准库)
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, args ...any) {}
func (l *defaultLogger) Info(msg string, args ...any)  {}
func (l *defaultLogger) Warn(msg string, args ...any)  {}
func (l *defaultLogger) Error(msg string, args ...any) {}

// nopLogger 空日志实现
type nopLogger struct{}

func (l *nopLogger) Debug(msg string, args ...any) {}
func (l *nopLogger) Info(msg string, args ...any)  {}
func (l *nopLogger) Warn(msg string, args ...any)  {}
func (l *nopLogger) Error(msg string, args ...any) {}

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
	// 心跳间隔（服务端：检测周期；客户端：发送周期）
	HeartbeatInterval time.Duration
	// 心跳超时（超过此时间未收到心跳则断开）
	HeartbeatTimeout time.Duration
	// 心跳 Ping 路由ID
	HeartbeatPingID uint32
	// 心跳 Pong 路由ID
	HeartbeatPongID uint32
	// 心跳 Pong 内容生成函数，入参为收到的 ping 消息，返回 pong body；为 nil 时 pong body 为空
	HeartbeatPongData func(ping Message) []byte
	// KCP 配置，ConnType 为 ConnTypeKCP 时生效
	KCPConfig *KCPConfig
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
		HeartbeatPingID:   0xFFFFFFFE,
		HeartbeatPongID:   0xFFFFFFFF,
		KCPConfig:         DefaultKCPConfig(),
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

// WithHeartbeatRouteID 设置心跳包的路由ID
func WithHeartbeatRouteID(pingID, pongID uint32) ServerOption {
	return func(c *ServerConfig) {
		c.HeartbeatPingID = pingID
		c.HeartbeatPongID = pongID
	}
}

// WithHeartbeatPongData 设置 pong 内容生成函数
// fn 入参为收到的 ping 消息，返回值作为 pong 的 body
func WithHeartbeatPongData(fn func(ping Message) []byte) ServerOption {
	return func(c *ServerConfig) {
		c.HeartbeatPongData = fn
	}
}

// WithKCPConfig 设置 KCP 配置
func WithKCPConfig(cfg *KCPConfig) ServerOption {
	return func(c *ServerConfig) {
		c.KCPConfig = cfg
	}
}
