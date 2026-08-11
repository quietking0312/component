package mcachedb

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// NewSQLXStoreFromDB 从现有 sqlx.DB 创建存储
func NewSQLXStoreFromDB(db *sqlx.DB, entityType Entity) (*SQLXStore, error) {
	config := DefaultSQLXStoreConfig()
	config.DSN = "" // 使用已有连接

	store := &SQLXStore{
		db:         db,
		config:     config,
		entityType: entityType,
	}

	if err := store.createTable(); err != nil {
		return nil, fmt.Errorf("create table failed: %w", err)
	}

	return store, nil
}

// SimpleConfig 简化配置
type SimpleConfig struct {
	// 数据库
	DB *sqlx.DB

	// Redis（可选，为空则不使用L2）
	RedisAddr   string
	RedisPass   string
	RedisDB     int
	RedisPrefix string

	// 性能配置
	L1Size        int           // L1最大条目数，默认10万
	FlushInterval time.Duration // 刷盘间隔，默认1秒
	SyncInterval  time.Duration // L1->L2同步间隔，默认500ms
}

// NewSimpleCache 创建简化版多级缓存
func NewSimpleCache(entityType Entity, cfg *SimpleConfig) (*MultiCache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	l1Size := cfg.L1Size
	if l1Size <= 0 {
		l1Size = defaultMultiCacheL1MaxSize
	}

	flushInterval := cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = defaultMultiCacheFlushInterval
	}

	syncInterval := cfg.SyncInterval
	if syncInterval <= 0 {
		syncInterval = defaultMultiCacheSyncInterval
	}

	dbStore, err := NewSQLXStoreFromDB(cfg.DB, entityType)
	if err != nil {
		return nil, err
	}

	multiConfig := &MultiCacheConfig{
		L1MaxSize:     l1Size,
		SyncInterval:  syncInterval,
		FlushInterval: flushInterval,
	}

	var l2 L2Store
	if cfg.RedisAddr != "" {
		redisStore, err := NewRedisStore(&RedisConfig{
			Addr:       cfg.RedisAddr,
			Password:   cfg.RedisPass,
			DB:         cfg.RedisDB,
			KeyPrefix:  cfg.RedisPrefix,
			DefaultTTL: defaultRedisTTL,
		}, entityType)
		if err != nil {
			dbStore.Close()
			return nil, err
		}
		l2 = redisStore
	}

	return NewMultiCache(dbStore, l2, multiConfig)
}

// MustNewSimpleCache 创建缓存，失败则panic
func MustNewSimpleCache(entityType Entity, cfg *SimpleConfig) *MultiCache {
	cache, err := NewSimpleCache(entityType, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create cache: %v", err))
	}
	return cache
}

// CacheBuilder 缓存构建器
type CacheBuilder struct {
	db            *sqlx.DB
	entityType    Entity
	redisAddr     string
	redisPass     string
	redisDB       int
	redisPrefix   string
	l1Size        int
	flushInterval time.Duration
	syncInterval  time.Duration
}

// NewCacheBuilder 创建构建器
func NewCacheBuilder(db *sqlx.DB, entityType Entity) *CacheBuilder {
	return &CacheBuilder{
		db:            db,
		entityType:    entityType,
		l1Size:        defaultMultiCacheL1MaxSize,
		flushInterval: defaultMultiCacheFlushInterval,
		syncInterval:  defaultMultiCacheSyncInterval,
	}
}

// WithRedis 设置Redis
func (b *CacheBuilder) WithRedis(addr, password string, db int) *CacheBuilder {
	b.redisAddr = addr
	b.redisPass = password
	b.redisDB = db
	return b
}

// WithRedisPrefix 设置Redis前缀
func (b *CacheBuilder) WithRedisPrefix(prefix string) *CacheBuilder {
	b.redisPrefix = prefix
	return b
}

// WithL1Size 设置L1大小
func (b *CacheBuilder) WithL1Size(size int) *CacheBuilder {
	b.l1Size = size
	return b
}

// WithFlushInterval 设置刷盘间隔
func (b *CacheBuilder) WithFlushInterval(d time.Duration) *CacheBuilder {
	b.flushInterval = d
	return b
}

// WithSyncInterval 设置同步间隔
func (b *CacheBuilder) WithSyncInterval(d time.Duration) *CacheBuilder {
	b.syncInterval = d
	return b
}

// Build 构建缓存
func (b *CacheBuilder) Build() (*MultiCache, error) {
	cfg := &SimpleConfig{
		DB:            b.db,
		L1Size:        b.l1Size,
		FlushInterval: b.flushInterval,
		SyncInterval:  b.syncInterval,
	}

	if b.redisAddr != "" {
		cfg.RedisAddr = b.redisAddr
		cfg.RedisPass = b.redisPass
		cfg.RedisDB = b.redisDB
		cfg.RedisPrefix = b.redisPrefix
	}

	return NewSimpleCache(b.entityType, cfg)
}

// MustBuild 构建缓存，失败则panic
func (b *CacheBuilder) MustBuild() *MultiCache {
	cache, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build cache: %v", err))
	}
	return cache
}

// NewHighReliabilityCache 创建高可靠性缓存（写L1时同步写L2 Redis）
func NewHighReliabilityCache(db *sqlx.DB, redisAddr string, entityType Entity) (*MultiCache, error) {
	dbStore, err := NewSQLXStoreFromDB(db, entityType)
	if err != nil {
		return nil, err
	}

	redisStore, err := NewRedisStore(&RedisConfig{
		Addr:       redisAddr,
		DefaultTTL: defaultRedisTTL,
	}, entityType)
	if err != nil {
		dbStore.Close()
		return nil, err
	}

	cfg := &MultiCacheConfig{
		L1MaxSize:     defaultMultiCacheL1MaxSize,
		WriteMode:     WriteModeWriteL2,
		SyncInterval:  defaultMultiCacheSyncInterval,
		FlushInterval: defaultMultiCacheFlushInterval,
	}

	return NewMultiCache(dbStore, redisStore, cfg)
}

// NewUltraReliabilityCache 创建超高可靠性缓存（写L1同步写L2，且快速刷L3）
func NewUltraReliabilityCache(db *sqlx.DB, redisAddr string, entityType Entity) (*MultiCache, error) {
	dbStore, err := NewSQLXStoreFromDB(db, entityType)
	if err != nil {
		return nil, err
	}

	redisStore, err := NewRedisStore(&RedisConfig{
		Addr:       redisAddr,
		DefaultTTL: defaultRedisTTL,
	}, entityType)
	if err != nil {
		dbStore.Close()
		return nil, err
	}

	cfg := &MultiCacheConfig{
		L1MaxSize:     defaultMultiCacheL1MaxSize,
		WriteMode:     WriteModeWriteL2,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
	}

	return NewMultiCache(dbStore, redisStore, cfg)
}
