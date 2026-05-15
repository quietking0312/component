package msock

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/xtaci/kcp-go/v5"
)

// KCPConfig KCP配置
type KCPConfig struct {
	// 发送窗口大小
	SendWindow int
	// 接收窗口大小
	RecvWindow int
	// 数据包大小
	Mtu int
	// 是否启用FEC
	EnableFEC bool
	// FEC数据分片数
	DataShards int
	// FEC校验分片数
	ParityShards int
	// 是否启用加密
	EnableCrypt bool
	// 加密密钥
	CryptKey string
}

// DefaultKCPConfig 返回默认KCP配置
func DefaultKCPConfig() *KCPConfig {
	return &KCPConfig{
		SendWindow:   128,
		RecvWindow:   128,
		Mtu:          1350,
		EnableFEC:    false,
		DataShards:   10,
		ParityShards: 3,
		EnableCrypt:  false,
		CryptKey:     "",
	}
}

// kcpConn KCP连接实现
type kcpConn struct {
	*baseConn
	net.Conn
	server *Server
	codec  Codec
}

// newKCPConn 创建KCP连接
func newKCPConn(conn net.Conn, server *Server) *kcpConn {
	c := &kcpConn{
		baseConn: newBaseConn(ConnTypeKCP),
		Conn:     conn,
		server:   server,
		codec:    server.codec,
	}
	c.initSendQueue(128)
	c.sendWg.Add(1)
	go c.sendLoop()
	return c
}

// LocalAddr 返回本地地址
func (c *kcpConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

// RemoteAddr 返回远程地址
func (c *kcpConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// Send 发送消息
func (c *kcpConn) Send(msg Message) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}

	return c.SendBytes(data)
}

// sendLoop KCP发送协程
func (c *kcpConn) sendLoop() {
	defer c.sendWg.Done()
	for data := range c.sendCh {
		writeTimeout := 10 * time.Second
		if c.server != nil && c.server.config.WriteTimeout > 0 {
			writeTimeout = c.server.config.WriteTimeout
		}
		if err := c.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			c.Close()
			return
		}
		if _, err := c.Conn.Write(data); err != nil {
			c.Close()
			return
		}
	}
}

// SendBytes 发送原始字节
func (c *kcpConn) SendBytes(data []byte) error {
	return c.sendAsync(data)
}

// Close 关闭连接
func (c *kcpConn) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.close()
	c.waitSendDone()
	return c.Conn.Close()
}

// SetReadDeadline 设置读取超时
func (c *kcpConn) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *kcpConn) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}

// readLoop 读取循环（两阶段解码）
func (c *kcpConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.server.logger.Error(fmt.Sprintf("panic in kcp readLoop: %v", r))
		}
		c.Close()
	}()

	codec := c.server.codec
	headerSize := codec.HeaderSize()
	header := make([]byte, headerSize)

	for {
		if c.IsClosed() {
			return
		}

		if c.server.config.ReadTimeout > 0 {
			if err := c.SetReadDeadline(time.Now().Add(c.server.config.ReadTimeout)); err != nil {
				c.server.logger.Warn(fmt.Sprintf("set read deadline error: %v", err))
				return
			}
		}

		var msg Message

		if headerSize == 0 {
			lc, ok := codec.(*LineCodec)
			if !ok {
				c.server.logger.Error("codec headerSize=0 but is not LineCodec")
				return
			}
			line, err := scanLine(c.Conn, lc.MaxPacketSize())
			if err != nil {
				if !c.IsClosed() {
					c.server.logger.Debug(fmt.Sprintf("kcp read error: %v", err))
				}
				return
			}
			msg = NewMessage(0, line)
		} else {
			if _, err := io.ReadFull(c.Conn, header); err != nil {
				if !c.IsClosed() {
					c.server.logger.Debug(fmt.Sprintf("kcp read header error: %v", err))
				}
				return
			}

			routeID, bodyLen, err := codec.DecodeHeader(header)
			if err != nil {
				c.server.logger.Error(fmt.Sprintf("kcp decode header error: %v", err))
				return
			}

			var body []byte
			if bodyLen > 0 {
				body = acquireBody(bodyLen)
				if _, err = io.ReadFull(c.Conn, body); err != nil {
					releaseBody(body)
					if !c.IsClosed() {
						c.server.logger.Debug(fmt.Sprintf("kcp read body error: %v", err))
					}
					return
				}
			}

			msg, err = codec.DecodeBody(routeID, body)
			if bodyLen > 0 {
				releaseBody(body)
			}
			if err != nil {
				c.server.logger.Error(fmt.Sprintf("kcp decode body error: %v", err))
				return
			}
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					c.server.logger.Error(fmt.Sprintf("panic in handler: %v, conn: %s", r, c.ID()))
				}
			}()
			c.server.handleMessage(c, msg)
		}()
	}
}

// runKCPServer 运行KCP服务器（在Server.runKCP中调用）
func (s *Server) runKCPServer(config ...*KCPConfig) error {
	cfg := DefaultKCPConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	var listener net.Listener
	var err error

	if cfg.EnableCrypt && cfg.CryptKey != "" {
		// 使用加密的KCP
		listener, err = kcp.ListenWithOptions(s.config.Address, nil, 0, 0)
		if err != nil {
			s.logger.Error(fmt.Sprintf("kcp listen error: %v", err))
			return ErrListenFailed
		}
	} else {
		// 不使用加密的KCP
		listener, err = kcp.Listen(s.config.Address)
		if err != nil {
			s.logger.Error(fmt.Sprintf("kcp listen error: %v", err))
			return ErrListenFailed
		}
	}

	// 设置KCP参数
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

		// 设置KCP连接参数
		if kcpConn, ok := conn.(*kcp.UDPSession); ok {
			kcpConn.SetWindowSize(128, 128)
			kcpConn.SetNoDelay(1, 10, 2, 1)
			kcpConn.SetStreamMode(true)
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
