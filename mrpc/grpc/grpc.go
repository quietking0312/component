package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ========== Server ==========

// ServerConfig gRPC 服务端配置
type ServerConfig struct {
	// 监听地址，默认 :9000
	Address string
	// TLS 配置，为 nil 时使用明文传输
	TLS *tls.Config
	// 单次请求最大接收字节数，默认 4MB
	MaxRecvMsgSize int
	// 单次请求最大发送字节数，默认 4MB
	MaxSendMsgSize int
	// 连接保活参数
	Keepalive *KeepaliveConfig
	// 自定义 grpc.ServerOption（会追加在内置选项之后）
	ExtraOptions []grpc.ServerOption
}

// KeepaliveConfig keepalive 参数
type KeepaliveConfig struct {
	// 客户端空闲多久后发送 ping，默认 60s
	Time time.Duration
	// ping 等待响应超时，默认 20s
	Timeout time.Duration
}

func defaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:        ":9000",
		MaxRecvMsgSize: 4 << 20,
		MaxSendMsgSize: 4 << 20,
		Keepalive: &KeepaliveConfig{
			Time:    60 * time.Second,
			Timeout: 20 * time.Second,
		},
	}
}

// Server gRPC 服务端
type Server struct {
	cfg *ServerConfig
	srv *grpc.Server
	lis net.Listener
}

// NewServer 创建 gRPC 服务端，cfg 为 nil 时使用默认配置。
func NewServer(cfg *ServerConfig) *Server {
	if cfg == nil {
		cfg = defaultServerConfig()
	}
	if cfg.Address == "" {
		cfg.Address = ":9000"
	}
	if cfg.MaxRecvMsgSize <= 0 {
		cfg.MaxRecvMsgSize = 4 << 20
	}
	if cfg.MaxSendMsgSize <= 0 {
		cfg.MaxSendMsgSize = 4 << 20
	}
	return &Server{cfg: cfg}
}

// Register 注册 gRPC 服务，在 Serve 之前调用。
//
//	s.Register(func(srv *grpc.Server) {
//	    pb.RegisterMyServiceServer(srv, &myImpl{})
//	})
func (s *Server) Register(fn func(*grpc.Server)) *Server {
	if s.srv == nil {
		s.srv = grpc.NewServer(s.buildOpts()...)
	}
	fn(s.srv)
	return s
}

// Serve 开始监听并阻塞服务。
func (s *Server) Serve() error {
	if s.srv == nil {
		s.srv = grpc.NewServer(s.buildOpts()...)
	}
	lis, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		return fmt.Errorf("grpc listen %s: %w", s.cfg.Address, err)
	}
	s.lis = lis
	return s.srv.Serve(lis)
}

// Stop 优雅关闭：先等待进行中的请求完成，超时后强制关闭。
func (s *Server) Stop(timeout time.Duration) {
	if s.srv == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		s.srv.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		s.srv.Stop()
	}
}

func (s *Server) buildOpts() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(s.cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.cfg.MaxSendMsgSize),
	}
	if s.cfg.Keepalive != nil {
		opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    s.cfg.Keepalive.Time,
			Timeout: s.cfg.Keepalive.Timeout,
		}))
	}
	if s.cfg.TLS != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.cfg.TLS)))
	}
	opts = append(opts, s.cfg.ExtraOptions...)
	return opts
}

// ========== Client ==========

// ClientConfig gRPC 客户端配置
type ClientConfig struct {
	// 目标地址，如 127.0.0.1:9000
	Target string
	// TLS 配置，为 nil 时使用明文传输
	TLS *tls.Config
	// 连接超时，默认 5s
	DialTimeout time.Duration
	// 单次请求最大接收字节数，默认 4MB
	MaxRecvMsgSize int
	// 单次请求最大发送字节数，默认 4MB
	MaxSendMsgSize int
	// 连接保活参数
	Keepalive *ClientKeepaliveConfig
	// 自定义 DialOption（追加在内置选项之后）
	ExtraOptions []grpc.DialOption
}

// ClientKeepaliveConfig 客户端 keepalive 参数
type ClientKeepaliveConfig struct {
	// 多久发一次 ping，默认 30s
	Time time.Duration
	// ping 超时，默认 10s
	Timeout time.Duration
	// 没有活跃 RPC 时也发 ping
	PermitWithoutStream bool
}

// Client gRPC 客户端连接
type Client struct {
	*grpc.ClientConn
}

// NewClient 创建并连接 gRPC 客户端，cfg 为 nil 时使用默认配置。
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		cfg = &ClientConfig{}
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.MaxRecvMsgSize <= 0 {
		cfg.MaxRecvMsgSize = 4 << 20
	}
	if cfg.MaxSendMsgSize <= 0 {
		cfg.MaxSendMsgSize = 4 << 20
	}

	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMsgSize),
		),
	}

	if cfg.TLS != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(cfg.TLS)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if cfg.Keepalive != nil {
		opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.Keepalive.Time,
			Timeout:             cfg.Keepalive.Timeout,
			PermitWithoutStream: cfg.Keepalive.PermitWithoutStream,
		}))
	}

	opts = append(opts, cfg.ExtraOptions...)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", cfg.Target, err)
	}
	return &Client{conn}, nil
}
