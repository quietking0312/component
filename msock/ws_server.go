package msock

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketUpgrader WebSocket升级器
type WebSocketUpgrader struct {
	upgrader websocket.Upgrader
}

// NewWebSocketUpgrader 创建WebSocket升级器
func NewWebSocketUpgrader() *WebSocketUpgrader {
	return &WebSocketUpgrader{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境应该检查
			},
		},
	}
}

// SetCheckOrigin 设置跨域检查函数
func (u *WebSocketUpgrader) SetCheckOrigin(fn func(r *http.Request) bool) {
	u.upgrader.CheckOrigin = fn
}

// wsConn WebSocket连接实现
type wsConn struct {
	*baseConn
	conn    *websocket.Conn
	server  *Server
	codec   Codec
	writeMu sync.Mutex
}

// newWSConn 创建WebSocket连接
func newWSConn(conn *websocket.Conn, server *Server) *wsConn {
	c := &wsConn{
		baseConn: newBaseConn(ConnTypeWebSocket),
		conn:     conn,
		server:   server,
		codec:    server.codec,
	}
	c.initSendQueue(128)
	c.sendWg.Add(1)
	go c.sendLoop()
	return c
}

// LocalAddr 返回本地地址
func (c *wsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr 返回远程地址
func (c *wsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Send 发送消息
func (c *wsConn) Send(msg Message) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}

	return c.SendBytes(data)
}

// SendBytes 发送二进制消息
func (c *wsConn) SendBytes(data []byte) error {
	return c.sendAsync(data)
}

// sendLoop WebSocket发送协程
func (c *wsConn) sendLoop() {
	defer c.sendWg.Done()
	for data := range c.sendCh {
		writeTimeout := 10 * time.Second
		if c.server != nil && c.server.config.WriteTimeout > 0 {
			writeTimeout = c.server.config.WriteTimeout
		}
		c.writeMu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		err := c.conn.WriteMessage(websocket.BinaryMessage, data)
		c.writeMu.Unlock()
		if err != nil {
			c.Close()
			return
		}
	}
}

// SendText 发送文本消息
func (c *wsConn) SendText(text string) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	writeTimeout := 10 * time.Second
	if c.server != nil && c.server.config.WriteTimeout > 0 {
		writeTimeout = c.server.config.WriteTimeout
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	return c.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

// Close 关闭连接
func (c *wsConn) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.close()
	c.waitSendDone()

	// 发送关闭消息
	c.writeMu.Lock()
	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.writeMu.Unlock()

	return c.conn.Close()
}

// SetReadDeadline 设置读取超时
func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// readLoop 读取循环
func (c *wsConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.server.logger.Error(fmt.Sprintf("panic in ws readLoop: %v", r))
		}
		c.Close()
	}()

	for {
		if c.IsClosed() {
			return
		}

		if c.server.config.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.server.config.ReadTimeout))
		}

		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if !c.IsClosed() {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.server.logger.Warn(fmt.Sprintf("websocket error: %v", err))
				}
			}
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			// 处理二进制消息（安全执行，handler panic 不关闭连接）
			func() {
				defer func() {
					if r := recover(); r != nil {
						c.server.logger.Error(fmt.Sprintf("panic in handler: %v, conn: %s", r, c.ID()))
					}
				}()
				c.handleBinaryMessage(data)
			}()
		case websocket.TextMessage:
			func() {
				defer func() {
					if r := recover(); r != nil {
						c.server.logger.Error(fmt.Sprintf("panic in handler: %v, conn: %s", r, c.ID()))
					}
				}()
				c.handleTextMessage(data)
			}()
		case websocket.PingMessage:
			// 自动回复pong
			c.writeMu.Lock()
			c.conn.WriteMessage(websocket.PongMessage, nil)
			c.writeMu.Unlock()
		case websocket.CloseMessage:
			return
		}
	}
}

// handleBinaryMessage 处理二进制消息（帧内两阶段解码）
func (c *wsConn) handleBinaryMessage(data []byte) {
	codec := c.server.codec
	headerSize := codec.HeaderSize()
	for len(data) > 0 {
		if len(data) < headerSize {
			c.server.logger.Error(fmt.Sprintf("ws frame too short: %d < %d", len(data), headerSize))
			return
		}
		routeID, bodyLen, err := codec.DecodeHeader(data[:headerSize])
		if err != nil {
			c.server.logger.Error(fmt.Sprintf("decode header error: %v", err))
			return
		}
		end := headerSize + bodyLen
		if len(data) < end {
			c.server.logger.Error(fmt.Sprintf("ws frame incomplete: need %d, have %d", end, len(data)))
			return
		}
		msg, err := codec.DecodeBody(routeID, data[headerSize:end])
		if err != nil {
			c.server.logger.Error(fmt.Sprintf("decode body error: %v", err))
			return
		}
		c.server.handleMessage(c, msg)
		data = data[end:]
	}
}

// handleTextMessage 处理文本消息
func (c *wsConn) handleTextMessage(data []byte) {
	// 对于文本消息，可以创建一个特殊的路由ID或者直接作为二进制处理
	msg := NewMessage(0, data)
	c.server.handleMessage(c, msg)
}

// runWebSocketServer 运行WebSocket服务器（在Server.runWebSocket中调用）
func (s *Server) runWebSocketServer() error {
	upgrader := NewWebSocketUpgrader()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error(fmt.Sprintf("websocket upgrade error: %v", err))
			return
		}

		s.wg.Add(1)
		go s.handleWSConn(conn)
	})

	s.httpServer = &http.Server{
		Addr:    s.config.Address,
		Handler: mux,
	}

	s.logger.Info(fmt.Sprintf("websocket server listening on %s/ws", s.config.Address))

	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleWSConn 处理WebSocket连接
func (s *Server) handleWSConn(wsConnObj *websocket.Conn) {
	defer s.wg.Done()

	// 创建连接对象
	conn := newWSConn(wsConnObj, s)

	// 添加到连接管理器
	if !s.connManager.Add(conn) {
		s.logger.Warn(fmt.Sprintf("max connections reached, reject websocket connection from %s", wsConnObj.RemoteAddr()))
		_ = wsConnObj.Close()
		return
	}
	defer s.connManager.Remove(conn.ID())

	s.metrics.totalConns.Add(1)
	s.logger.Info(fmt.Sprintf("websocket connection established: %s from %s", conn.ID(), conn.RemoteAddr()))

	// 触发连接建立回调
	if s.handlers.onConnect != nil {
		s.handlers.onConnect(conn)
	}

	// 启动读取循环
	conn.readLoop()

	// 连接断开
	s.logger.Info(fmt.Sprintf("websocket connection closed: %s", conn.ID()))

	// 触发连接断开回调
	if s.handlers.onDisconnect != nil {
		s.handlers.onDisconnect(conn)
	}
}

// RegisterWSRoute 注册WebSocket处理路由（如果使用HTTP路由）
func (s *Server) RegisterWSRoute(path string, upgrader *WebSocketUpgrader) {
	if s.httpServer == nil {
		return
	}

	// 这里可以实现更复杂的HTTP路由注册
	// 需要与gin或其他HTTP框架集成
}
