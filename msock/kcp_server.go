package msock

import (
	"fmt"
	"net"

	"github.com/xtaci/kcp-go/v5"
)

// runKCPServer 运行KCP服务器（在Server.runKCP中调用）
func (s *Server) runKCPServer() error {
	cfg := s.config.KCPConfig
	if cfg == nil {
		cfg = DefaultKCPConfig()
	}

	var (
		listener net.Listener
		err      error
	)

	if cfg.EnableCrypt && cfg.CryptKey != "" {
		block, cryptErr := newKCPBlockCrypt(cfg.CryptKey)
		if cryptErr != nil {
			s.logger.Error(fmt.Sprintf("kcp create block crypt error: %v", cryptErr))
			return ErrListenFailed
		}
		listener, err = kcp.ListenWithOptions(s.config.Address, block, cfg.DataShards, cfg.ParityShards)
	} else {
		listener, err = kcp.Listen(s.config.Address)
	}
	if err != nil {
		s.logger.Error(fmt.Sprintf("kcp listen error: %v", err))
		return ErrListenFailed
	}

	if kcpListener, ok := listener.(*kcp.Listener); ok {
		kcpListener.SetReadBuffer(s.config.ReadBufferSize)
		kcpListener.SetWriteBuffer(s.config.WriteBufferSize)
	}

	s.listener = listener
	s.logger.Info(fmt.Sprintf("kcp server listening on %s", s.config.Address))

	return s.acceptKCPLoop()
}

// acceptKCPLoop 接受KCP连接循环
func (s *Server) acceptKCPLoop() error {
	cfg := s.config.KCPConfig
	if cfg == nil {
		cfg = DefaultKCPConfig()
	}

	for {
		if s.isClosed() {
			return ErrServerClosed
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if s.isClosed() {
				return ErrServerClosed
			}
			s.logger.Error(fmt.Sprintf("kcp accept error: %v", err))
			if s.handlers.onError != nil {
				s.handlers.onError(nil, err)
			}
			continue
		}

		if sess, ok := conn.(*kcp.UDPSession); ok {
			sess.SetWindowSize(cfg.SendWindow, cfg.RecvWindow)
			sess.SetNoDelay(cfg.NoDelay, cfg.Interval, cfg.Resend, cfg.NC)
			sess.SetStreamMode(true)
			if cfg.Mtu > 0 {
				sess.SetMtu(cfg.Mtu)
			}
		}

		s.wg.Add(1)
		go s.handleKCPConn(conn)
	}
}

// handleKCPConn 处理KCP连接
func (s *Server) handleKCPConn(netConn net.Conn) {
	defer s.wg.Done()

	// 创建连接对象
	conn := newKCPConn(netConn, s)

	// 添加到连接管理器
	if !s.connManager.Add(conn) {
		s.logger.Warn(fmt.Sprintf("max connections reached, reject kcp connection from %s", netConn.RemoteAddr()))
		_ = netConn.Close()
		return
	}
	defer s.connManager.Remove(conn.ID())

	s.metrics.totalConns.Add(1)
	s.logger.Info(fmt.Sprintf("kcp connection established: %s from %s", conn.ID(), conn.RemoteAddr()))

	// 触发连接建立回调
	if s.handlers.onConnect != nil {
		s.handlers.onConnect(conn)
	}

	// 启动读取循环
	conn.readLoop()

	// 连接断开
	s.logger.Info(fmt.Sprintf("kcp connection closed: %s", conn.ID()))

	// 触发连接断开回调
	if s.handlers.onDisconnect != nil {
		s.handlers.onDisconnect(conn)
	}
}
