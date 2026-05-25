package gorpc

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
)

// Codec 传输编解码格式
type Codec int

const (
	// CodecGob 使用 Go 内置 gob 编码（性能最好，仅 Go 客户端可用）
	CodecGob Codec = iota
	// CodecJSON 使用 JSON-RPC 编码（跨语言，性能略低）
	CodecJSON
)

// ========== Server ==========

// ServerConfig 服务端配置
type ServerConfig struct {
	// 监听地址，默认 :9001
	Address string
	// 编解码格式，默认 CodecGob
	Codec Codec
}

// Server net/rpc 服务端
type Server struct {
	cfg    *ServerConfig
	srv    *rpc.Server
	lis    net.Listener
	stopCh chan struct{}
}

// NewServer 创建服务端，cfg 为 nil 时使用默认配置。
func NewServer(cfg *ServerConfig) *Server {
	if cfg == nil {
		cfg = &ServerConfig{}
	}
	if cfg.Address == "" {
		cfg.Address = ":9001"
	}
	return &Server{
		cfg:    cfg,
		srv:    rpc.NewServer(),
		stopCh: make(chan struct{}),
	}
}

// Register 注册服务对象。
// 遵循 net/rpc 规范：方法必须满足 func (t *T) MethodName(args *Args, reply *Reply) error。
func (s *Server) Register(receiver interface{}) error {
	return s.srv.Register(receiver)
}

// RegisterName 以指定名称注册服务对象。
func (s *Server) RegisterName(name string, receiver interface{}) error {
	return s.srv.RegisterName(name, receiver)
}

// Serve 开始监听并阻塞服务。
func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		return fmt.Errorf("gorpc listen %s: %w", s.cfg.Address, err)
	}
	s.lis = lis

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return nil
			default:
				log.Printf("gorpc accept error: %v", err)
				continue
			}
		}
		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(conn net.Conn) {
	if s.cfg.Codec == CodecJSON {
		s.srv.ServeCodec(jsonrpc.NewServerCodec(conn))
	} else {
		s.srv.ServeConn(conn)
	}
}

// Stop 关闭服务端。
func (s *Server) Stop() error {
	close(s.stopCh)
	if s.lis != nil {
		return s.lis.Close()
	}
	return nil
}

// ========== Client ==========

// ClientConfig 客户端配置
type ClientConfig struct {
	// 目标地址，如 127.0.0.1:9001
	Target string
	// 编解码格式，需与服务端一致，默认 CodecGob
	Codec Codec
	// 连接超时，默认 5s
	DialTimeout time.Duration
}

// Client net/rpc 客户端
type Client struct {
	cfg    *ClientConfig
	client *rpc.Client
}

// NewClient 创建并连接客户端。
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		cfg = &ClientConfig{}
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}

	conn, err := net.DialTimeout("tcp", cfg.Target, cfg.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("gorpc dial %s: %w", cfg.Target, err)
	}

	var rc *rpc.Client
	if cfg.Codec == CodecJSON {
		rc = jsonrpc.NewClient(conn)
	} else {
		rc = rpc.NewClient(conn)
	}

	return &Client{cfg: cfg, client: rc}, nil
}

// Call 同步调用远程方法。
// serviceMethod 格式为 "ServiceName.MethodName"。
func (c *Client) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return c.client.Call(serviceMethod, args, reply)
}

// Go 异步调用远程方法，返回 *rpc.Call 可通过 Done channel 等待结果。
func (c *Client) Go(serviceMethod string, args interface{}, reply interface{}, done chan *rpc.Call) *rpc.Call {
	return c.client.Go(serviceMethod, args, reply, done)
}

// Close 关闭连接。
func (c *Client) Close() error {
	return c.client.Close()
}
