package mcachedb

import (
	"context"
	"time"
)

// Entity 实体接口，定义缓存实体的基本契约
type Entity interface {
	CacheKey() string
	IsDeleted() bool
	SetDeleted(deleted bool)
	Version() int64
	IncrementVersion()
	Copy() Entity
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// L2Store L2 存储接口
type L2Store interface {
	Get(ctx context.Context, key string) (Entity, error)
	MGet(ctx context.Context, keys []string) (map[string]Entity, error)
	Set(ctx context.Context, entity Entity) error
	MSet(ctx context.Context, entities []Entity) error
	Delete(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Close() error
}

// DBStore 数据库存储接口（L3）
type DBStore interface {
	Get(ctx context.Context, key string) (Entity, error)
	MGet(ctx context.Context, keys []string) (map[string]Entity, error)
	Insert(ctx context.Context, entity Entity) error
	Update(ctx context.Context, entity Entity) error
	Delete(ctx context.Context, key string) error
	BatchInsert(ctx context.Context, entities []Entity) error
	BatchUpdate(ctx context.Context, entities []Entity) error
	BatchDelete(ctx context.Context, keys []string) error
	Close() error
}

// WriteMode 写入模式（MultiCache 使用）
type WriteMode int

const (
	// WriteModeAsync 写 L1，后台异步同步 L2/L3（默认）
	WriteModeAsync WriteMode = iota
	// WriteModeCacheAside 先写 L3，成功后删除 L1/L2；下次读回填
	WriteModeCacheAside
	// WriteModeWriteL2 同步写 L1+L2，L3 后台异步刷盘；跨进程共享热数据
	WriteModeWriteL2
)

// Config L1 内存缓存配置
type Config struct {
	// MaxCacheSize L1 最大条目数，0 表示不限
	MaxCacheSize int
	// DefaultExpiration 默认 TTL，0 表示永不过期
	DefaultExpiration time.Duration
	// CleanupInterval 清理过期条目的间隔，0 表示不自动清理
	CleanupInterval time.Duration
}

const (
	defaultMaxCacheSize    = 100000
	defaultCleanupInterval = 5 * time.Minute
)

// DefaultConfig 返回默认 L1 配置
func DefaultConfig() *Config {
	return &Config{
		MaxCacheSize:    defaultMaxCacheSize,
		CleanupInterval: defaultCleanupInterval,
	}
}

// Option Cache 配置选项
type Option func(*Config)

// WithMaxCacheSize 设置最大缓存大小
func WithMaxCacheSize(size int) Option {
	return func(c *Config) { c.MaxCacheSize = size }
}

// WithCleanupInterval 设置清理间隔
func WithCleanupInterval(d time.Duration) Option {
	return func(c *Config) { c.CleanupInterval = d }
}

// WithDefaultExpiration 设置默认过期时间
func WithDefaultExpiration(d time.Duration) Option {
	return func(c *Config) { c.DefaultExpiration = d }
}

// Stats L1 缓存统计
type Stats struct {
	CacheHits   int64
	CacheMisses int64
	CacheSize   int
}

// HitRate 命中率
func (s *Stats) HitRate() float64 {
	total := s.CacheHits + s.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(s.CacheHits) / float64(total)
}

// entry 缓存条目（内部使用）
// dirty/dirtySeq/isNew 由 MultiCache 在写入时注入，Cache 本身不修改这些字段。
type entry struct {
	entity    Entity
	createdAt time.Time
	expireAt  time.Time

	// 以下字段由 MultiCache 管理，用于异步刷盘追踪
	dirty    bool   // 待 flush 到 L3
	dirtySeq uint64 // 标记 dirty 时的序列号，用于 flush 与 Set 的竞态控制
	isNew    bool   // true = 需要 Insert（首次写入），false = 需要 Update
}

func (e *entry) isExpired() bool {
	if e.expireAt.IsZero() {
		return false
	}
	return time.Now().After(e.expireAt)
}
