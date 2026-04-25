package msock

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// baseConn 基础连接实现
type baseConn struct {
	id       string
	connType ConnType
	ctx      context.Context
	cancel   context.CancelFunc
	values   sync.Map
	closed   int32
	closeCh  chan struct{}

	sendCh chan []byte
	sendWg sync.WaitGroup
}

// newBaseConn 创建基础连接
func newBaseConn(connType ConnType) *baseConn {
	ctx, cancel := context.WithCancel(context.Background())
	return &baseConn{
		id:       uuid.New().String(),
		connType: connType,
		ctx:      ctx,
		cancel:   cancel,
		closed:   0,
		closeCh:  make(chan struct{}),
	}
}

// ID 返回连接ID
func (c *baseConn) ID() string {
	return c.id
}

// Type 返回连接类型
func (c *baseConn) Type() ConnType {
	return c.connType
}

// Context 返回连接上下文
func (c *baseConn) Context() context.Context {
	return c.ctx
}

// SetValue 存储键值对
func (c *baseConn) SetValue(key, value interface{}) {
	c.values.Store(key, value)
}

// GetValue 获取值
func (c *baseConn) GetValue(key interface{}) (interface{}, bool) {
	return c.values.Load(key)
}

// initSendQueue 初始化发送队列
func (c *baseConn) initSendQueue(capacity int) {
	c.sendCh = make(chan []byte, capacity)
}

// sendAsync 异步发送数据到队列
func (c *baseConn) sendAsync(data []byte) error {
	if c.IsClosed() {
		return ErrConnClosed
	}
	select {
	case c.sendCh <- data:
		return nil
	default:
		return ErrSendChannelFull
	}
}

// waitSendDone 等待发送协程完成
func (c *baseConn) waitSendDone() {
	c.sendWg.Wait()
}

// close 关闭连接
func (c *baseConn) close() {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.cancel()
		close(c.closeCh)
		if c.sendCh != nil {
			close(c.sendCh)
		}
	}
}

// IsClosed 检查连接是否已关闭
func (c *baseConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

// waitClose 等待连接关闭
func (c *baseConn) waitClose() <-chan struct{} {
	return c.closeCh
}

// tcpConn TCP连接实现
type tcpConn struct {
	*baseConn
	net.Conn
	server *Server
	codec  Codec
	reader *bufferedReader
}

// newTCPConn 创建TCP连接
func newTCPConn(conn net.Conn, server *Server) *tcpConn {
	c := &tcpConn{
		baseConn: newBaseConn(ConnTypeTCP),
		Conn:     conn,
		server:   server,
		codec:    server.codec,
		reader:   newBufferedReader(conn, server.config.ReadBufferSize),
	}
	c.initSendQueue(128)
	c.sendWg.Add(1)
	go c.sendLoop()
	return c
}

// LocalAddr 返回本地地址
func (c *tcpConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

// RemoteAddr 返回远程地址
func (c *tcpConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// Send 发送消息
func (c *tcpConn) Send(msg Message) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}

	return c.SendBytes(data)
}

// sendLoop TCP发送协程
func (c *tcpConn) sendLoop() {
	defer c.sendWg.Done()
	for data := range c.sendCh {
		writeTimeout := 10 * time.Second
		if c.server != nil && c.server.config.WriteTimeout > 0 {
			writeTimeout = c.server.config.WriteTimeout
		}
		if err := c.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			continue
		}
		_, _ = c.Conn.Write(data)
	}
}

// SendBytes 发送原始字节
func (c *tcpConn) SendBytes(data []byte) error {
	return c.sendAsync(data)
}

// Close 关闭连接
func (c *tcpConn) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.close()
	c.waitSendDone()
	return c.Conn.Close()
}

// SetReadDeadline 设置读取超时
func (c *tcpConn) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *tcpConn) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}

// readLoop 读取循环
func (c *tcpConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.server.logger.Errorf("panic in readLoop: %v", r)
		}
		c.Close()
	}()

	for {
		if c.IsClosed() {
			return
		}

		if c.server.config.ReadTimeout > 0 {
			if err := c.SetReadDeadline(time.Now().Add(c.server.config.ReadTimeout)); err != nil {
				c.server.logger.Warnf("set read deadline error: %v", err)
				return
			}
		}

		// 读取数据
		data, err := c.reader.Read()
		if err != nil {
			if !c.IsClosed() {
				c.server.logger.Debugf("read error: %v", err)
			}
			return
		}

		// 解码消息
		for len(data) > 0 {
			msg, n, err := c.server.codec.Decode(data)
			if err != nil {
				c.server.logger.Errorf("decode error: %v", err)
				return
			}
			if n == 0 {
				// 数据不足，保留在缓冲区
				c.reader.Unread(data)
				break
			}

			// 安全处理消息：handler panic 不传播到 readLoop，不关闭连接
			func() {
				defer func() {
					if r := recover(); r != nil {
						c.server.logger.Errorf("panic in handler: %v, conn: %s", r, c.ID())
					}
				}()
				c.server.handleMessage(c, msg)
			}()
			data = data[n:]
		}
	}
}

// bufferedReader 带缓冲的读取器
type bufferedReader struct {
	conn   net.Conn
	buffer []byte
	start  int
	end    int
}

// newBufferedReader 创建缓冲读取器
func newBufferedReader(conn net.Conn, size int) *bufferedReader {
	return &bufferedReader{
		conn:   conn,
		buffer: make([]byte, size),
	}
}

// Read 读取数据
func (r *bufferedReader) Read() ([]byte, error) {
	// 如果缓冲区有未处理的数据
	if r.start < r.end {
		data := make([]byte, r.end-r.start)
		copy(data, r.buffer[r.start:r.end])
		r.start = r.end
		return data, nil
	}

	// 从连接读取
	r.start = 0
	r.end = 0
	n, err := r.conn.Read(r.buffer)
	if err != nil {
		return nil, err
	}
	r.end = n

	data := make([]byte, n)
	copy(data, r.buffer[:n])
	return data, nil
}

// Unread 将数据放回缓冲区
func (r *bufferedReader) Unread(data []byte) {
	// 简单实现：将数据复制到缓冲区开头
	// 实际生产环境可能需要更复杂的环形缓冲区
	if len(data) > len(r.buffer) {
		// 数据太大，扩展缓冲区
		r.buffer = append(data, r.buffer...)
	} else {
		copy(r.buffer, data)
	}
	r.start = 0
	r.end = len(data)
}

// ConnManager 连接管理器
type ConnManager struct {
	mu      sync.RWMutex
	conns   map[string]Conn
	maxConn int
}

// NewConnManager 创建连接管理器
func NewConnManager(maxConn int) *ConnManager {
	return &ConnManager{
		conns:   make(map[string]Conn),
		maxConn: maxConn,
	}
}

// Add 添加连接
func (m *ConnManager) Add(conn Conn) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.maxConn > 0 && len(m.conns) >= m.maxConn {
		return false
	}

	m.conns[conn.ID()] = conn
	return true
}

// Remove 移除连接
func (m *ConnManager) Remove(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conns, connID)
}

// Get 获取连接
func (m *ConnManager) Get(connID string) (Conn, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.conns[connID]
	return conn, ok
}

// GetAll 获取所有连接
func (m *ConnManager) GetAll() []Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]Conn, 0, len(m.conns))
	for _, conn := range m.conns {
		conns = append(conns, conn)
	}
	return conns
}

// Count 返回连接数量
func (m *ConnManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conns)
}

// BroadcastBytes 广播原始字节到所有连接（使用协程池 + 批量发送）
func (m *ConnManager) BroadcastBytes(data []byte) {
	conns := m.GetAll()
	if len(conns) == 0 {
		return
	}

	batchSize := 256
	workerCount := (len(conns) + batchSize - 1) / batchSize
	if workerCount > 8 {
		workerCount = 8
		batchSize = (len(conns) + workerCount - 1) / workerCount
	}

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		start := i * batchSize
		if start >= len(conns) {
			break
		}
		end := start + batchSize
		if end > len(conns) {
			end = len(conns)
		}
		batch := conns[start:end]

		wg.Add(1)
		go func(b []Conn) {
			defer wg.Done()
			for _, conn := range b {
				if !conn.IsClosed() {
					_ = conn.SendBytes(data)
				}
			}
		}(batch)
	}
	wg.Wait()
}

// CloseAll 关闭所有连接
func (m *ConnManager) CloseAll() {
	conns := m.GetAll()
	for _, conn := range conns {
		conn.Close()
	}
}
