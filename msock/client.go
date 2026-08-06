package msock

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lxzan/gws"
	"github.com/xtaci/kcp-go/v5"
)

// poolEntry 连接池中的一个槽位
type poolEntry struct {
	mu    sync.Mutex
	conn  Conn
	index int
}

func (e *poolEntry) getConn() Conn {
	e.mu.Lock()
	c := e.conn
	e.mu.Unlock()
	return c
}

func (e *poolEntry) setConn(c Conn) {
	e.mu.Lock()
	e.conn = c
	e.mu.Unlock()
}

// Client 网络客户端（连接池 + 断线重连）
type Client struct {
	config   *ServerConfig
	addr     string
	pool     []*poolEntry
	rrIdx    atomic.Uint64
	codec    Codec
	router   *Router
	logger   Logger
	metrics  clientMetrics
	handlers struct {
		onConnect    func(Conn)
		onDisconnect func(Conn)
		onError      func(error)
	}
	mu      sync.RWMutex
	closed  bool
	closeCh chan struct{}
	wg      sync.WaitGroup
	hbOnce  sync.Once
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
	if config.PoolSize < 1 {
		config.PoolSize = 1
	}

	pool := make([]*poolEntry, config.PoolSize)
	for i := range pool {
		pool[i] = &poolEntry{index: i}
	}

	return &Client{
		config:  config,
		codec:   config.Codec,
		logger:  config.Logger,
		pool:    pool,
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

// Connect 连接到服务器，建立连接池中所有连接
func (c *Client) Connect(addr string) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrConnClosed
	}
	c.addr = addr
	for _, e := range c.pool {
		e.mu.Lock()
		old := e.conn
		e.conn = nil
		e.mu.Unlock()
		if old != nil {
			old.Close()
		}
	}
	c.mu.Unlock()

	var firstErr error
	for _, e := range c.pool {
		if err := c.connectEntry(e, addr); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	c.hbOnce.Do(c.startHeartbeat)
	return firstErr
}

// connectEntry 连接单个槽位，失败时若启用重连则在后台重试
func (c *Client) connectEntry(e *poolEntry, addr string) error {
	conn, err := c.dial(addr)
	if err != nil {
		c.logger.Error(fmt.Sprintf("pool[%d] connect failed: %v", e.index, err))
		if c.config.ReconnectEnable {
			c.wg.Add(1)
			go c.watchAndReconnect(e, nil, addr)
		}
		return err
	}

	e.setConn(conn)
	if c.handlers.onConnect != nil {
		c.handlers.onConnect(conn)
	}
	if c.config.ReconnectEnable {
		c.wg.Add(1)
		go c.watchAndReconnect(e, conn, addr)
	}
	return nil
}

// watchAndReconnect 监视连接断开并以指数退避自动重连。
// conn 为 nil 时直接进入重连循环（首次连接失败的情况）。
// 整个生命周期在一个 goroutine 内完成，避免递归启动。
func (c *Client) watchAndReconnect(e *poolEntry, initialConn Conn, addr string) {
	defer c.wg.Done()

	current := initialConn

	delay := c.config.ReconnectInitDelay
	if delay <= 0 {
		delay = time.Second
	}
	maxDelay := c.config.ReconnectMaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	if delay > maxDelay {
		delay = maxDelay
	}

	for attempt := 1; ; attempt++ {
		// 等待当前连接断开
		if current != nil {
			bc := extractBaseConn(current)
			if bc != nil {
				select {
				case <-bc.waitClose():
				case <-c.closeCh:
					return
				}
			}
			// 仅当槽位仍持有本连接时才清空，避免与 Connect 重入竞争
			e.mu.Lock()
			if e.conn != current {
				// 槽位已被 Connect 重入或其他路径更新，退出本 watcher
				e.mu.Unlock()
				return
			}
			e.conn = nil
			e.mu.Unlock()
			current = nil
		}

		if c.IsClosed() {
			return
		}
		if c.config.ReconnectMaxAttempts > 0 && attempt > c.config.ReconnectMaxAttempts {
			c.logger.Warn(fmt.Sprintf("pool[%d] reconnect exhausted after %d attempts", e.index, c.config.ReconnectMaxAttempts))
			return
		}

		// 若槽位已有新连接（Connect 重入），本 watcher 退出
		e.mu.Lock()
		if e.conn != nil {
			e.mu.Unlock()
			return
		}
		e.mu.Unlock()

		select {
		case <-time.After(delay):
		case <-c.closeCh:
			return
		}

		c.logger.Info(fmt.Sprintf("pool[%d] reconnecting (attempt %d, delay %v)", e.index, attempt, delay))

		newConn, err := c.dial(addr)
		if err != nil {
			c.logger.Warn(fmt.Sprintf("pool[%d] reconnect failed: %v", e.index, err))
			if delay < maxDelay {
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
			}
			continue
		}

		// dial 期间客户端可能已被关闭或槽位已被其他路径恢复
		e.mu.Lock()
		if c.IsClosed() {
			e.mu.Unlock()
			newConn.Close()
			return
		}
		if e.conn != nil {
			e.mu.Unlock()
			newConn.Close()
			return
		}
		e.conn = newConn
		e.mu.Unlock()

		if c.handlers.onConnect != nil {
			c.handlers.onConnect(newConn)
		}
		c.metrics.totalReconnects.Add(1)
		c.logger.Info(fmt.Sprintf("pool[%d] reconnected", e.index))

		// 重置退避计时，以新连接继续循环监视
		delay = c.config.ReconnectInitDelay
		if delay <= 0 {
			delay = time.Second
		}
		attempt = 0
		current = newConn
	}
}

// dial 根据配置类型建立一条连接
func (c *Client) dial(addr string) (Conn, error) {
	switch c.config.ConnType {
	case ConnTypeTCP:
		return c.dialTCP(addr)
	case ConnTypeWebSocket:
		return c.dialWebSocket(addr)
	case ConnTypeKCP:
		return c.dialKCP(addr)
	case ConnTypeGWS:
		return c.dialGWS(addr)
	default:
		return nil, ErrUnsupportedProtocol
	}
}

// pickConn 轮询选取一个可用连接
func (c *Client) pickConn() Conn {
	n := uint64(len(c.pool))
	if n == 0 {
		return nil
	}
	start := c.rrIdx.Add(1) - 1
	for i := uint64(0); i < n; i++ {
		e := c.pool[(start+i)%n]
		e.mu.Lock()
		conn := e.conn
		e.mu.Unlock()
		if conn != nil && !conn.IsClosed() {
			return conn
		}
	}
	return nil
}

// Send 轮询连接池发送消息
func (c *Client) Send(msg Message) error {
	conn := c.pickConn()
	if conn == nil {
		return ErrNoAvailableConn
	}
	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}
	if err = conn.SendBytes(data); err != nil {
		return err
	}
	c.metrics.totalSendBytes.Add(int64(len(data)))
	return nil
}

// SendBytes 轮询连接池发送原始字节
func (c *Client) SendBytes(data []byte) error {
	conn := c.pickConn()
	if conn == nil {
		return ErrNoAvailableConn
	}
	if err := conn.SendBytes(data); err != nil {
		return err
	}
	c.metrics.totalSendBytes.Add(int64(len(data)))
	return nil
}

// Close 关闭客户端及所有连接，等待后台 goroutine 退出
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	close(c.closeCh)
	c.mu.Unlock()

	for _, e := range c.pool {
		e.mu.Lock()
		conn := e.conn
		e.conn = nil
		e.mu.Unlock()
		if conn != nil {
			conn.Close()
		}
	}
	c.wg.Wait()
	return nil
}

// IsClosed 检查客户端是否已关闭
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Conn 返回第一个可用连接（兼容旧接口）
func (c *Client) Conn() Conn {
	return c.pickConn()
}

// PoolSize 返回连接池大小
func (c *Client) PoolSize() int {
	return len(c.pool)
}

// AvailableConns 返回当前可用（已连接）的连接数
func (c *Client) AvailableConns() int {
	n := 0
	for _, e := range c.pool {
		e.mu.Lock()
		if e.conn != nil && !e.conn.IsClosed() {
			n++
		}
		e.mu.Unlock()
	}
	return n
}

// Stats 返回客户端统计快照
func (c *Client) Stats() ClientStats {
	return ClientStats{
		PoolSize:        len(c.pool),
		AvailableConns:  c.AvailableConns(),
		TotalReconnects: c.metrics.totalReconnects.Load(),
		TotalMessages:   c.metrics.totalMessages.Load(),
		TotalRecvBytes:  c.metrics.totalRecvBytes.Load(),
		TotalSendBytes:  c.metrics.totalSendBytes.Load(),
	}
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
	if bc := extractBaseConn(conn); bc != nil {
		bc.TouchHeartbeat()
	}
	c.metrics.totalMessages.Add(1)
	c.metrics.totalRecvBytes.Add(int64(len(msg.Data())))
	if c.router == nil {
		c.logger.Warn("router not set, dropping message")
		return
	}
	c.router.Handle(conn, msg)
}

// ========== dial 实现 ==========

func (c *Client) dialTCP(addr string) (Conn, error) {
	netConn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	conn := &tcpClientConn{
		tcpConn: &tcpConn{
			baseConn:     newBaseConn(ConnTypeTCP, c.config.IDGenerator()),
			Conn:         netConn,
			server:       nil,
			codec:        c.codec,
			writeTimeout: c.config.WriteTimeout,
		},
		client: c,
	}
	conn.initSendQueue(128)
	conn.sendWg.Add(1)
	go conn.sendLoop()
	go conn.readLoop()

	c.logger.Info(fmt.Sprintf("tcp connected to %s", addr))
	return conn, nil
}

func (c *Client) dialWebSocket(addr string) (Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   c.config.ReadBufferSize,
		WriteBufferSize:  c.config.WriteBufferSize,
	}

	ws, _, err := dialer.Dial(addr, nil)
	if err != nil {
		return nil, err
	}

	conn := &wsClientConn{
		wsConn: &wsConn{
			baseConn: newBaseConn(ConnTypeWebSocket, c.config.IDGenerator()),
			conn:     ws,
			server:   nil,
			codec:    c.codec,
		},
		client: c,
	}
	conn.initSendQueue(128)
	conn.sendWg.Add(1)
	go conn.sendLoop()
	go conn.readLoop()

	c.logger.Info(fmt.Sprintf("websocket connected to %s", addr))
	return conn, nil
}

func (c *Client) dialGWS(addr string) (Conn, error) {
	handler := &gwsClientEventHandler{client: c}
	socket, _, err := gws.NewClient(handler, &gws.ClientOption{
		Addr:             addr,
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   c.config.ReadBufferSize,
	})
	if err != nil {
		return nil, err
	}

	conn := &gwsClientConn{
		gwsConn: newGWSConn(socket, nil, c.codec, c.config.IDGenerator()),
		client:  c,
	}
	socket.Session().Store("msock_conn", conn)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Error(fmt.Sprintf("panic in gws client readLoop: %v", r))
			}
		}()
		socket.ReadLoop()
	}()

	c.logger.Info(fmt.Sprintf("gws connected to %s", addr))
	return conn, nil
}

func (c *Client) dialKCP(addr string) (Conn, error) {
	cfg := c.config.KCPConfig
	if cfg == nil {
		cfg = DefaultKCPConfig()
	}

	var (
		rawConn net.Conn
		err     error
	)

	if cfg.EnableCrypt && cfg.CryptKey != "" {
		block, cryptErr := newKCPBlockCrypt(cfg.CryptKey)
		if cryptErr != nil {
			return nil, cryptErr
		}
		rawConn, err = kcp.DialWithOptions(addr, block, cfg.DataShards, cfg.ParityShards)
	} else {
		rawConn, err = kcp.Dial(addr)
	}
	if err != nil {
		return nil, err
	}

	if sess, ok := rawConn.(*kcp.UDPSession); ok {
		sess.SetWindowSize(cfg.SendWindow, cfg.RecvWindow)
		sess.SetNoDelay(cfg.NoDelay, cfg.Interval, cfg.Resend, cfg.NC)
		sess.SetStreamMode(true)
		if cfg.Mtu > 0 {
			sess.SetMtu(cfg.Mtu)
		}
	}

	clientConn := &kcpClientConn{
		kcpConn: &kcpConn{
			baseConn:     newBaseConn(ConnTypeKCP, c.config.IDGenerator()),
			Conn:         rawConn,
			server:       nil,
			codec:        c.codec,
			writeTimeout: c.config.WriteTimeout,
		},
		client: c,
	}
	clientConn.initSendQueue(128)
	clientConn.sendWg.Add(1)
	go clientConn.sendLoop()
	go clientConn.readLoop()

	c.logger.Info(fmt.Sprintf("kcp connected to %s", addr))
	return clientConn, nil
}
