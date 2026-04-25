package mcachedb

import (
	"context"
	"time"
)

// Entity 实体接口，定义了缓存实体的基本契约
type Entity interface {
	// CacheKey 返回缓存键
	CacheKey() string
	// IsDeleted 是否标记删除
	IsDeleted() bool
	// SetDeleted 标记删除
	SetDeleted(deleted bool)
	// Version 获取版本号（用于乐观锁）
	Version() int64
	// IncrementVersion 增加版本号
	IncrementVersion()
	// Copy 复制实体（深拷贝）
	Copy() Entity
}

// DBStore 数据库存储接口
type DBStore interface {
	// Get 从数据库获取实体
	Get(ctx context.Context, key string) (Entity, error)

	// MGet 批量获取
	MGet(ctx context.Context, keys []string) (map[string]Entity, error)

	// Insert 插入新记录
	Insert(ctx context.Context, entity Entity) error

	// Update 更新记录
	Update(ctx context.Context, entity Entity) error

	// Delete 删除记录
	Delete(ctx context.Context, key string) error

	// BatchInsert 批量插入
	BatchInsert(ctx context.Context, entities []Entity) error

	// BatchUpdate 批量更新
	BatchUpdate(ctx context.Context, entities []Entity) error

	// BatchDelete 批量删除
	BatchDelete(ctx context.Context, keys []string) error

	// Close 关闭连接
	Close() error
}

// WriteMode 写入模式
type WriteMode int

const (
	// WriteModeAsync 异步写入（默认），写入缓存后立即返回，后台批量刷盘
	WriteModeAsync WriteMode = iota
	// WriteModeSync 同步写入，写入缓存同时写入数据库
	WriteModeSync
	// WriteModeWriteThrough 直写模式，先写数据库成功后，再更新缓存
	WriteModeWriteThrough
)

// FlushMode 刷新模式
type FlushMode int

const (
	// FlushModeInterval 定时刷新（默认）
	FlushModeInterval FlushMode = iota
	// FlushModeImmediate 立即刷新（每次操作都触发检查）
	FlushModeImmediate
	// FlushModeManual 手动刷新（需要手动调用 Flush）
	FlushModeManual
)

// Config 配置
type Config struct {
	// 写入模式
	WriteMode WriteMode

	// 刷新模式
	FlushMode FlushMode

	// 刷新间隔（定时刷新模式下有效）
	FlushInterval time.Duration

	// 批量大小，达到此数量触发写入
	BatchSize int

	// 最大缓存条目数
	MaxCacheSize int

	// 默认过期时间，0 表示永不过期
	DefaultExpiration time.Duration

	// 清理过期数据间隔
	CleanupInterval time.Duration

	// 写入失败重试次数
	RetryCount int

	// 重试间隔
	RetryInterval time.Duration

	// 并发写入协程数
	WriteWorkers int

	// 数据库连接池配置
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBMaxLifetime  time.Duration

	// 回调函数
	OnFlushStart   func(dirtyCount int)
	OnFlushSuccess func(count int, duration time.Duration)
	OnFlushError   func(err error, entities []Entity)
	OnCacheMiss    func(key string)
	OnCacheHit     func(key string)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		WriteMode:         WriteModeAsync,
		FlushMode:         FlushModeInterval,
		FlushInterval:     1 * time.Second,
		BatchSize:         100,
		MaxCacheSize:      100000,
		DefaultExpiration: 0, // 默认不过期
		CleanupInterval:   5 * time.Minute,
		RetryCount:        3,
		RetryInterval:     100 * time.Millisecond,
		WriteWorkers:      1,
		DBMaxOpenConns:    20,
		DBMaxIdleConns:    5,
		DBMaxLifetime:     1 * time.Hour,
	}
}

// Option 配置选项
type Option func(*Config)

// WithWriteMode 设置写入模式
func WithWriteMode(mode WriteMode) Option {
	return func(c *Config) {
		c.WriteMode = mode
	}
}

// WithFlushMode 设置刷新模式
func WithFlushMode(mode FlushMode) Option {
	return func(c *Config) {
		c.FlushMode = mode
	}
}

// WithFlushInterval 设置刷新间隔
func WithFlushInterval(d time.Duration) Option {
	return func(c *Config) {
		c.FlushInterval = d
	}
}

// WithBatchSize 设置批量大小
func WithBatchSize(size int) Option {
	return func(c *Config) {
		c.BatchSize = size
	}
}

// WithMaxCacheSize 设置最大缓存大小
func WithMaxCacheSize(size int) Option {
	return func(c *Config) {
		c.MaxCacheSize = size
	}
}

// WithCleanupInterval 设置清理过期数据间隔
func WithCleanupInterval(d time.Duration) Option {
	return func(c *Config) {
		c.CleanupInterval = d
	}
}

// WithDefaultExpiration 设置默认过期时间
func WithDefaultExpiration(d time.Duration) Option {
	return func(c *Config) {
		c.DefaultExpiration = d
	}
}

// WithRetryCount 设置重试次数
func WithRetryCount(count int) Option {
	return func(c *Config) {
		c.RetryCount = count
	}
}

// WithWriteWorkers 设置写入工作协程数
func WithWriteWorkers(n int) Option {
	return func(c *Config) {
		c.WriteWorkers = n
	}
}

// Stats 统计信息
type Stats struct {
	// 缓存统计
	CacheHits   int64
	CacheMisses int64
	CacheSize   int
	DirtyCount  int64

	// 数据库统计
	DBReads       int64
	DBWrites      int64
	DBWriteErrors int64

	// 刷新统计
	FlushCount     int64
	FlushTotalTime time.Duration
	LastFlushTime  time.Time

	// 当前状态
	IsFlushing bool
}

// HitRate 命中率
func (s *Stats) HitRate() float64 {
	total := s.CacheHits + s.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(s.CacheHits) / float64(total)
}

// Entry 缓存条目
type entry struct {
	entity    Entity
	createdAt time.Time
	expireAt  time.Time
	dirty     bool
	deleted   bool
	isNew     bool // 是否是新记录（用于区分插入和更新）
}

// isExpired 检查是否过期
func (e *entry) isExpired() bool {
	if e.expireAt.IsZero() {
		return false
	}
	return time.Now().After(e.expireAt)
}
