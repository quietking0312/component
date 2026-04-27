package msock

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

// Server 网络服务器
type Server struct {
	config      *ServerConfig
	router      *Router
	codec       Codec
	connManager *ConnManager
	logger      Logger

	listener   net.Listener
	httpServer *http.Server

	handlers struct {
		onConnect    func(Conn)
		onDisconnect func(Conn)
		onError      func(Conn, error)
	}

	closed int32
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewServer 创建服务器
func NewServer(opts ...ServerOption) (*Server, error) {
	config := DefaultServerConfig()
	for _, opt := range opts {
		opt(config)
	}

	if config.Codec == nil {
		config.Codec = NewSimpleCodec()
	}

	return &Server{
		config:      config,
		codec:       config.Codec,
		logger:      config.Logger,
		connManager: NewConnManager(config.MaxConnections),
	}, nil
}

// SetRouter 设置路由器
func (s *Server) SetRouter(router *Router) {
	s.router = router
}

// SetCodec 设置编解码器
func (s *Server) SetCodec(codec Codec) {
	s.codec = codec
}

// SetLogger 设置日志器
func (s *Server) SetLogger(logger Logger) {
	s.logger = logger
}

// OnConnect 设置连接建立回调
func (s *Server) OnConnect(fn func(Conn)) {
	s.handlers.onConnect = fn
}

// OnDisconnect 设置连接断开回调
func (s *Server) OnDisconnect(fn func(Conn)) {
	s.handlers.onDisconnect = fn
}

// OnError 设置错误回调
func (s *Server) OnError(fn func(Conn, error)) {
	s.handlers.onError = fn
}

// Run 启动服务器
func (s *Server) Run() error {
	switch s.config.ConnType {
	case ConnTypeTCP:
		return s.runTCP()
	case ConnTypeWebSocket:
		return s.runWebSocket()
	case ConnTypeKCP:
		return s.runKCP()
	case ConnTypeGWS:
		return s.runGWSServer()
	default:
		return ErrUnsupportedProtocol
	}
}

// runTCP 运行TCP服务器
func (s *Server) runTCP() error {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		s.logger.Errorf("tcp listen error: %v", err)
		return ErrListenFailed
	}
	s.listener = listener

	s.logger.Infof("tcp server listening on %s", s.config.Address)

	return s.acceptLoop()
}

// runWebSocket 运行WebSocket服务器
func (s *Server) runWebSocket() error {
	return s.runWebSocketServer()
}

// runKCP 运行KCP服务器
func (s *Server) runKCP() error {
	return s.runKCPServer()
}

// acceptLoop 接受连接循环
func (s *Server) acceptLoop() error {
	for {
		if s.isClosed() {
			return ErrServerClosed
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if s.isClosed() {
				return ErrServerClosed
			}
			s.logger.Errorf("accept error: %v", err)
			if s.handlers.onError != nil {
				s.handlers.onError(nil, err)
			}
			continue
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

// handleConn 处理新连接
func (s *Server) handleConn(netConn net.Conn) {
	defer s.wg.Done()

	// 创建连接对象
	conn := newTCPConn(netConn, s)

	// 添加到连接管理器
	if !s.connManager.Add(conn) {
		s.logger.Warnf("max connections reached, reject connection from %s", netConn.RemoteAddr())
		netConn.Close()
		return
	}
	defer s.connManager.Remove(conn.ID())

	s.logger.Infof("connection established: %s from %s", conn.ID(), conn.RemoteAddr())

	// 触发连接建立回调
	if s.handlers.onConnect != nil {
		s.handlers.onConnect(conn)
	}

	// 启动读取循环
	conn.readLoop()

	// 连接断开
	s.logger.Infof("connection closed: %s", conn.ID())

	// 触发连接断开回调
	if s.handlers.onDisconnect != nil {
		s.handlers.onDisconnect(conn)
	}
}

// handleMessage 处理消息
func (s *Server) handleMessage(conn Conn, msg Message) {
	if s.router == nil {
		s.logger.Warnf("router not set, dropping message from %s", conn.ID())
		return
	}

	// 使用工作池处理消息（可选）
	s.router.Handle(conn, msg)
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	s.logger.Infof("stopping server...")

	// 关闭监听器
	if s.listener != nil {
		s.listener.Close()
	}

	// 关闭HTTP服务器
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}

	// 关闭所有连接
	s.connManager.CloseAll()

	// 等待所有goroutine完成
	s.wg.Wait()

	s.logger.Infof("server stopped")
	return nil
}

// isClosed 检查服务器是否已关闭
func (s *Server) isClosed() bool {
	return atomic.LoadInt32(&s.closed) == 1
}

// GetConnManager 获取连接管理器
func (s *Server) GetConnManager() *ConnManager {
	return s.connManager
}

// GetConfig 获取配置
func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// SendTo 发送消息到指定连接
func (s *Server) SendTo(connID string, msg Message) error {
	conn, ok := s.connManager.Get(connID)
	if !ok {
		return ErrConnClosed
	}
	return conn.Send(msg)
}

// Broadcast 广播消息（先编码一次，再批量发送字节，避免重复编码）
func (s *Server) Broadcast(msg Message) {
	data, err := s.codec.Encode(msg)
	if err != nil {
		s.logger.Errorf("broadcast encode error: %v", err)
		return
	}
	s.connManager.BroadcastBytes(data)
}

// ConnCount 返回当前连接数
func (s *Server) ConnCount() int {
	return s.connManager.Count()
}
