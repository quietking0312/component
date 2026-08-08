package msock

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/cpu"
)

// bodyPool 复用 body 读取缓冲，减少 GC 压力
var bodyPool = sync.Pool{
	New: func() interface{} { return make([]byte, 4096) },
}

// acquireBody 从池中取出一块至少 n 字节的缓冲
func acquireBody(n int) []byte {
	b := bodyPool.Get().([]byte)
	if cap(b) >= n {
		return b[:n]
	}
	return make([]byte, n)
}

// releaseBody 归还缓冲到池
func releaseBody(b []byte) {
	bodyPool.Put(b[:cap(b)])
}

// baseConn 基础连接实现
type baseConn struct {
	id            string
	connType      ConnType
	ctx           context.Context
	cancel        context.CancelFunc
	values        sync.Map
	closed        int32
	closeCh       chan struct{}
	lastHeartbeat int64 // unix nano，atomic 读写

	sendCh chan []byte
	sendWg sync.WaitGroup
}

// newBaseConn 创建基础连接，id 由外部传入
func newBaseConn(connType ConnType, id string) *baseConn {
	ctx, cancel := context.WithCancel(context.Background())
	return &baseConn{
		id:            id,
		connType:      connType,
		ctx:           ctx,
		cancel:        cancel,
		closed:        0,
		closeCh:       make(chan struct{}),
		lastHeartbeat: time.Now().UnixNano(),
	}
}

// TouchHeartbeat 更新最后心跳时间
func (c *baseConn) TouchHeartbeat() {
	atomic.StoreInt64(&c.lastHeartbeat, time.Now().UnixNano())
}

// LastHeartbeat 返回最后心跳时间
func (c *baseConn) LastHeartbeat() time.Time {
	return time.Unix(0, atomic.LoadInt64(&c.lastHeartbeat))
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

// baseConner 内部接口，所有嵌入 baseConn 的连接类型自动满足
type baseConner interface {
	getBaseConn() *baseConn
}

// getBaseConn 实现 baseConner 接口
func (c *baseConn) getBaseConn() *baseConn {
	return c
}

// waitClose 等待连接关闭
func (c *baseConn) waitClose() <-chan struct{} {
	return c.closeCh
}

// ConnManager 分片锁连接管理器
// 32 个分片，按连接 ID 首字节哈希，锁竞争降低为原来的 1/32
type ConnManager struct {
	shards  [32]connShard
	maxConn int
	total   atomic.Int64
}

type connShard struct {
	mu    sync.RWMutex
	conns map[string]Conn
	_     cpu.CacheLinePad
}

// NewConnManager 创建连接管理器
func NewConnManager(maxConn int) *ConnManager {
	m := &ConnManager{maxConn: maxConn}
	for i := range m.shards {
		m.shards[i].conns = make(map[string]Conn)
	}
	return m
}

// shard 按连接 ID 首字节选取分片
func (m *ConnManager) shard(connID string) *connShard {
	if len(connID) == 0 {
		return &m.shards[0]
	}
	return &m.shards[connID[0]&31]
}

// Add 添加连接，超过最大连接数返回 false
func (m *ConnManager) Add(conn Conn) bool {
	if m.maxConn > 0 && int(m.total.Load()) >= m.maxConn {
		return false
	}
	s := m.shard(conn.ID())
	s.mu.Lock()
	if m.maxConn > 0 && int(m.total.Load()) >= m.maxConn {
		s.mu.Unlock()
		return false
	}
	s.conns[conn.ID()] = conn
	m.total.Add(1)
	s.mu.Unlock()
	return true
}

// Remove 移除连接
func (m *ConnManager) Remove(connID string) {
	s := m.shard(connID)
	s.mu.Lock()
	_, ok := s.conns[connID]
	if ok {
		delete(s.conns, connID)
	}
	s.mu.Unlock()
	if ok {
		m.total.Add(-1)
	}
}

// Get 获取连接
func (m *ConnManager) Get(connID string) (Conn, bool) {
	s := m.shard(connID)
	s.mu.RLock()
	conn, ok := s.conns[connID]
	s.mu.RUnlock()
	return conn, ok
}

// GetAll 获取所有连接快照
func (m *ConnManager) GetAll() []Conn {
	conns := make([]Conn, 0, int(m.total.Load()))
	for i := range m.shards {
		s := &m.shards[i]
		s.mu.RLock()
		for _, c := range s.conns {
			conns = append(conns, c)
		}
		s.mu.RUnlock()
	}
	return conns
}

// Count 返回当前连接数
func (m *ConnManager) Count() int {
	return int(m.total.Load())
}

// BroadcastBytes 广播原始字节到所有连接，返回成功入队的连接数
func (m *ConnManager) BroadcastBytes(data []byte) int {
	return m.BroadcastBytesFilter(data, nil)
}

// BroadcastBytesTo 广播原始字节到指定 ID 的连接，返回成功入队的连接数
func (m *ConnManager) BroadcastBytesTo(data []byte, ids []string) int {
	if len(ids) == 0 {
		return m.BroadcastBytesFilter(data, nil)
	}
	var succeed int
	for _, id := range ids {
		if conn, ok := m.Get(id); ok {
			if !conn.IsClosed() && conn.SendBytes(data) == nil {
				succeed++
			}
		}
	}
	return succeed
}

// BroadcastBytesFilter 广播原始字节到满足条件的连接，filter 为 nil 则广播到所有连接，返回成功入队的连接数
func (m *ConnManager) BroadcastBytesFilter(data []byte, filter func(Conn) bool) int {
	return m.broadcastAll(data, filter)
}

// connSlicePool 复用连接切片，减少广播时的 GC 压力
var connSlicePool = sync.Pool{
	New: func() interface{} {
		s := make([]Conn, 0, 1024)
		return &s
	},
}

// broadcastAll 全量广播：按 shard 快速复制指针快照后释放读锁，再并行发送
// filter 不为 nil 时只发送满足条件的连接
func (m *ConnManager) broadcastAll(data []byte, filter func(Conn) bool) int {
	sp := connSlicePool.Get().(*[]Conn)
	conns := (*sp)[:0]

	for i := range m.shards {
		s := &m.shards[i]
		s.mu.RLock()
		for _, conn := range s.conns {
			if filter == nil || filter(conn) {
				conns = append(conns, conn)
			}
		}
		s.mu.RUnlock()
	}

	var (
		wg      sync.WaitGroup
		succeed atomic.Int64
	)
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount > len(conns) {
		workerCount = len(conns)
	}
	if workerCount == 0 {
		*sp = conns
		connSlicePool.Put(sp)
		return 0
	}
	batchSize := (len(conns) + workerCount - 1) / workerCount
	for i := 0; i < workerCount; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(conns) {
			end = len(conns)
		}
		wg.Add(1)
		go func(batch []Conn) {
			defer wg.Done()
			var n int64
			for _, conn := range batch {
				if !conn.IsClosed() && conn.SendBytes(data) == nil {
					n++
				}
			}
			if n > 0 {
				succeed.Add(n)
			}
		}(conns[start:end])
	}
	wg.Wait()

	*sp = conns
	connSlicePool.Put(sp)
	return int(succeed.Load())
}

// CloseAll 关闭所有连接
func (m *ConnManager) CloseAll() {
	for _, conn := range m.GetAll() {
		conn.Close()
	}
}
