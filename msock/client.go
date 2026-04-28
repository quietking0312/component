package msock

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lxzan/gws"
	"github.com/xtaci/kcp-go/v5"
)

// Client 网络客户端
type Client struct {
	config   *ServerConfig
	conn     Conn
	codec    Codec
	router   *Router
	logger   Logger
	handlers struct {
		onConnect    func(Conn)
		onDisconnect func(Conn)
		onError      func(error)
	}
	mu      sync.RWMutex
	closed  bool
	closeCh chan struct{}
}

// NewClient 创建客户端
func NewClient(opts ...ServerOption) *Client {
	config := DefaultServerConfig()
	for _, opt := range opts {
		opt(config)
	}

	if config.Codec == nil {
		config.Codec = NewSimpleCodec()
	}

	return &Client{
		config:  config,
		codec:   config.Codec,
		logger:  config.Logger,
		closeCh: make(chan struct{}),
	}
}

// SetRouter 设置路由器
func (c *Client) SetRouter(router *Router) {
	c.router = router
}

// SetCodec 设置编解码器
func (c *Client) SetCodec(codec Codec) {
	c.codec = codec
}

// Connect 连接到服务器
func (c *Client) Connect(addr string) error {
	switch c.config.ConnType {
	case ConnTypeTCP:
		return c.connectTCP(addr)
	case ConnTypeWebSocket:
		return c.connectWebSocket(addr)
	case ConnTypeKCP:
		return c.connectKCP(addr)
	case ConnTypeGWS:
		return c.connectGWS(addr)
	default:
		return ErrUnsupportedProtocol
	}
}

// connectTCP 连接TCP服务器
func (c *Client) connectTCP(addr string) error {
	netConn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		c.logger.Error(fmt.Sprintf("tcp connect error: %v", err))
		return err
	}

	conn := &tcpClientConn{
		tcpConn: &tcpConn{
			baseConn: newBaseConn(ConnTypeTCP),
			Conn:     netConn,
			server:   nil,
			codec:    c.codec,
			reader:   newBufferedReader(netConn, c.config.ReadBufferSize),
		},
		client: c,
	}
	conn.initSendQueue(128)
	conn.sendWg.Add(1)
	go conn.sendLoop()

	c.conn = conn

	// 触发连接回调
	if c.handlers.onConnect != nil {
		c.handlers.onConnect(conn)
	}

	// 启动读取循环
	go conn.readLoop()

	c.logger.Info(fmt.Sprintf("tcp connected to %s", addr))
	return nil
}

// connectWebSocket 连接WebSocket服务器
func (c *Client) connectWebSocket(addr string) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   c.config.ReadBufferSize,
		WriteBufferSize:  c.config.WriteBufferSize,
	}

	ws, _, err := dialer.Dial(addr, nil)
	if err != nil {
		c.logger.Error(fmt.Sprintf("websocket connect error: %v", err))
		return err
	}

	conn := &wsClientConn{
		wsConn: &wsConn{
			baseConn: newBaseConn(ConnTypeWebSocket),
			conn:     ws,
			server:   nil,
			codec:    c.codec,
		},
		client: c,
	}
	conn.initSendQueue(128)
	conn.sendWg.Add(1)
	go conn.sendLoop()

	c.conn = conn

	// 触发连接回调
	if c.handlers.onConnect != nil {
		c.handlers.onConnect(conn)
	}

	// 启动读取循环
	go conn.readLoop()

	c.logger.Info(fmt.Sprintf("websocket connected to %s", addr))
	return nil
}

// connectGWS 连接gws WebSocket服务器
func (c *Client) connectGWS(addr string) error {
	handler := &gwsClientEventHandler{client: c}
	socket, _, err := gws.NewClient(handler, &gws.ClientOption{
		Addr:             addr,
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   c.config.ReadBufferSize,
	})
	if err != nil {
		c.logger.Error(fmt.Sprintf("gws connect error: %v", err))
		return err
	}

	conn := &gwsClientConn{
		gwsConn: newGWSConn(socket, nil, c.codec),
		client:  c,
	}
	c.conn = conn
	socket.Session().Store("msock_conn", conn)

	// 触发连接回调
	if c.handlers.onConnect != nil {
		c.handlers.onConnect(conn)
	}

	// 启动读取循环
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Error(fmt.Sprintf("panic in gws client readLoop: %v", r))
			}
		}()
		socket.ReadLoop()
	}()

	c.logger.Info(fmt.Sprintf("gws connected to %s", addr))
	return nil
}

// connectKCP 连接KCP服务器
func (c *Client) connectKCP(addr string) error {
	conn, err := kcp.Dial(addr)
	if err != nil {
		c.logger.Error(fmt.Sprintf("kcp connect error: %v", err))
		return err
	}

	// 设置KCP参数
	if kcpSess, ok := conn.(*kcp.UDPSession); ok {
		kcpSess.SetWindowSize(128, 128)
		kcpSess.SetNoDelay(1, 10, 2, 1)
		kcpSess.SetStreamMode(true)
	}

	clientConn := &kcpClientConn{
		kcpConn: &kcpConn{
			baseConn: newBaseConn(ConnTypeKCP),
			Conn:     conn,
			server:   nil,
			codec:    c.codec,
			reader:   newBufferedReader(conn, c.config.ReadBufferSize),
		},
		client: c,
	}
	clientConn.initSendQueue(128)
	clientConn.sendWg.Add(1)
	go clientConn.sendLoop()

	c.conn = clientConn

	// 触发连接回调
	if c.handlers.onConnect != nil {
		c.handlers.onConnect(clientConn)
	}

	// 启动读取循环
	go clientConn.readLoop()

	c.logger.Info(fmt.Sprintf("kcp connected to %s", addr))
	return nil
}

// Send 发送消息
func (c *Client) Send(msg Message) error {
	if c.conn == nil {
		return ErrConnClosed
	}
	return c.conn.Send(msg)
}

// SendBytes 发送原始字节
func (c *Client) SendBytes(data []byte) error {
	if c.conn == nil {
		return ErrConnClosed
	}
	return c.conn.SendBytes(data)
}

// Close 关闭连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	close(c.closeCh)

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsClosed 检查客户端是否已关闭
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Conn 获取连接对象
func (c *Client) Conn() Conn {
	return c.conn
}

// OnConnect 设置连接建立回调
func (c *Client) OnConnect(fn func(Conn)) {
	c.handlers.onConnect = fn
}

// OnDisconnect 设置连接断开回调
func (c *Client) OnDisconnect(fn func(Conn)) {
	c.handlers.onDisconnect = fn
}

// OnError 设置错误回调
func (c *Client) OnError(fn func(error)) {
	c.handlers.onError = fn
}

// handleMessage 处理消息
func (c *Client) handleMessage(conn Conn, msg Message) {
	if c.router == nil {
		c.logger.Warn(fmt.Sprintf("router not set, dropping message"))
		return
	}
	c.router.Handle(conn, msg)
}

// ========== 客户端连接实现 ==========

// tcpClientConn TCP客户端连接
type tcpClientConn struct {
	*tcpConn
	client *Client
}

// readLoop 读取循环
func (c *tcpClientConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.client.logger.Error(fmt.Sprintf("panic in client readLoop: %v", r))
		}
		c.Close()
		if c.client.handlers.onDisconnect != nil {
			c.client.handlers.onDisconnect(c)
		}
	}()

	for {
		if c.IsClosed() {
			return
		}

		if c.client.config.ReadTimeout > 0 {
			if err := c.SetReadDeadline(time.Now().Add(c.client.config.ReadTimeout)); err != nil {
				return
			}
		}

		data, err := c.reader.Read()
		if err != nil {
			if !c.IsClosed() && c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}

		for len(data) > 0 {
			msg, n, err := c.client.codec.Decode(data)
			if err != nil {
				if c.client.handlers.onError != nil {
					c.client.handlers.onError(err)
				}
				return
			}
			if n == 0 {
				c.reader.Unread(data)
				break
			}

			c.client.handleMessage(c, msg)
			data = data[n:]
		}
	}
}

// wsClientConn WebSocket客户端连接
type wsClientConn struct {
	*wsConn
	client *Client
}

// readLoop 读取循环
func (c *wsClientConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.client.logger.Error(fmt.Sprintf("panic in client ws readLoop: %v", r))
		}
		c.Close()
		if c.client.handlers.onDisconnect != nil {
			c.client.handlers.onDisconnect(c)
		}
	}()

	for {
		if c.IsClosed() {
			return
		}

		if c.client.config.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.client.config.ReadTimeout))
		}

		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if !c.IsClosed() && c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			c.handleBinaryMessage(data)
		case websocket.TextMessage:
			c.handleTextMessage(data)
		case websocket.CloseMessage:
			return
		}
	}
}

func (c *wsClientConn) handleBinaryMessage(data []byte) {
	for len(data) > 0 {
		msg, n, err := c.client.codec.Decode(data)
		if err != nil {
			if c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}
		if n == 0 {
			break
		}
		c.client.handleMessage(c, msg)
		data = data[n:]
	}
}

func (c *wsClientConn) handleTextMessage(data []byte) {
	msg := NewMessage(0, data)
	c.client.handleMessage(c, msg)
}

// gwsClientEventHandler gws客户端事件处理器
type gwsClientEventHandler struct {
	client *Client
}

func (h *gwsClientEventHandler) OnOpen(socket *gws.Conn) {}

func (h *gwsClientEventHandler) OnClose(socket *gws.Conn, err error) {
	v, ok := socket.Session().Load("msock_conn")
	if !ok {
		return
	}
	conn := v.(*gwsClientConn)
	_ = conn.Close()
	if h.client.handlers.onDisconnect != nil {
		h.client.handlers.onDisconnect(conn)
	}
}

func (h *gwsClientEventHandler) OnPing(socket *gws.Conn, payload []byte) {
	_ = socket.WritePong(nil)
}

func (h *gwsClientEventHandler) OnPong(socket *gws.Conn, payload []byte) {}

func (h *gwsClientEventHandler) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()

	v, ok := socket.Session().Load("msock_conn")
	if !ok {
		return
	}
	conn := v.(*gwsClientConn)

	switch message.Opcode {
	case gws.OpcodeBinary:
		conn.handleBinaryMessage(message.Bytes())
	case gws.OpcodeText:
		conn.handleTextMessage(message.Bytes())
	}
}

// kcpClientConn KCP客户端连接
type kcpClientConn struct {
	*kcpConn
	client *Client
}

// readLoop 读取循环
func (c *kcpClientConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.client.logger.Error(fmt.Sprintf("panic in client kcp readLoop: %v", r))
		}
		c.Close()
		if c.client.handlers.onDisconnect != nil {
			c.client.handlers.onDisconnect(c)
		}
	}()

	for {
		if c.IsClosed() {
			return
		}

		if c.client.config.ReadTimeout > 0 {
			if err := c.SetReadDeadline(time.Now().Add(c.client.config.ReadTimeout)); err != nil {
				return
			}
		}

		data, err := c.reader.Read()
		if err != nil {
			if !c.IsClosed() && c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}

		for len(data) > 0 {
			msg, n, err := c.client.codec.Decode(data)
			if err != nil {
				if c.client.handlers.onError != nil {
					c.client.handlers.onError(err)
				}
				return
			}
			if n == 0 {
				c.reader.Unread(data)
				break
			}

			c.client.handleMessage(c, msg)
			data = data[n:]
		}
	}
}
