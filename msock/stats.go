package msock

import "sync/atomic"

// ServerStats 服务端统计快照
type ServerStats struct {
	// CurrentConns 当前连接数
	CurrentConns int64
	// TotalConns 累计建立的连接数
	TotalConns int64
	// TotalMessages 累计处理的消息数
	TotalMessages int64
	// TotalRecvBytes 累计接收字节数
	TotalRecvBytes int64
	// TotalSendBytes 累计发送字节数
	TotalSendBytes int64
}

// serverMetrics 服务端内部计数器（原子操作）
type serverMetrics struct {
	totalConns     atomic.Int64
	totalMessages  atomic.Int64
	totalRecvBytes atomic.Int64
	totalSendBytes atomic.Int64
}

// ClientStats 客户端统计快照
type ClientStats struct {
	// PoolSize 连接池槽位总数
	PoolSize int
	// AvailableConns 当前可用（已连接）的连接数
	AvailableConns int
	// TotalReconnects 累计重连次数
	TotalReconnects int64
	// TotalMessages 累计处理的消息数
	TotalMessages int64
	// TotalRecvBytes 累计接收字节数
	TotalRecvBytes int64
	// TotalSendBytes 累计发送字节数
	TotalSendBytes int64
}

// clientMetrics 客户端内部计数器（原子操作）
type clientMetrics struct {
	totalReconnects atomic.Int64
	totalMessages   atomic.Int64
	totalRecvBytes  atomic.Int64
	totalSendBytes  atomic.Int64
}
