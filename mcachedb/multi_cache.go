package mcachedb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MultiCache 多级缓存 (L1内存 -> L2Redis -> L3数据库)
type MultiCache struct {
	l1     *Cache      // 一级缓存：本地内存
	l2     *RedisStore // 二级缓存：Redis
	l3     DBStore     // 三级存储：数据库
	config *MultiCacheConfig
	stats  MultiCacheStats
	stopCh chan struct{}
	wg     sync.WaitGroup
	mu     sync.RWMutex
	l2Down bool
}

// MultiCacheConfig 多级缓存配置
type MultiCacheConfig struct {
	L1MaxSize         int
	L1CleanupInterval time.Duration
	L2RedisConfig     *RedisConfig
	WriteToL2OnSet    bool
	SyncInterval      time.Duration
	FlushInterval     time.Duration
	L2Downgrade       bool
}

// DefaultMultiCacheConfig 默认配置
func DefaultMultiCacheConfig() *MultiCacheConfig {
	return &MultiCacheConfig{
		L1MaxSize:         100000,
		L1CleanupInterval: 5 * time.Minute,
		L2RedisConfig:     DefaultRedisConfig(),
		WriteToL2OnSet:    false,
		SyncInterval:      500 * time.Millisecond,
		FlushInterval:     1 * time.Second,
		L2Downgrade:       true,
	}
}

// MultiCacheStats 统计
type MultiCacheStats struct {
	L1Hits int64
	L2Hits int64
	L3Hits int64
	Misses int64
}

// HitRate 命中率
func (s *MultiCacheStats) HitRate() float64 {
	total := s.L1Hits + s.L2Hits + s.L3Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.L1Hits+s.L2Hits+s.L3Hits) / float64(total)
}

// NewMultiCache 创建多级缓存
func NewMultiCache(dbStore DBStore, config *MultiCacheConfig) (*MultiCache, error) {
	if dbStore == nil {
		return nil, fmt.Errorf("dbStore cannot be nil")
	}

	if config == nil {
		config = DefaultMultiCacheConfig()
	}

	// 创建 L2 Redis
	var l2Store *RedisStore
	var err error
	if config.L2RedisConfig != nil {
		l2Store, err = NewRedisStore(config.L2RedisConfig)
		if err != nil {
			if !config.L2Downgrade {
				return nil, err
			}
			l2Store = nil
		}
	}

	mc := &MultiCache{
		l3:     dbStore,
		l2:     l2Store,
		config: config,
		stopCh: make(chan struct{}),
	}

	// 创建 L1 内存缓存
	l1Store := &multiCacheStore{mc: mc}
	mc.l1, err = New(l1Store,
		WithMaxCacheSize(config.L1MaxSize),
		WithCleanupInterval(config.L1CleanupInterval),
	)
	if err != nil {
		return nil, err
	}

	// 启动后台协程
	if l2Store != nil {
		mc.wg.Add(1)
		go mc.syncToL2Loop()
	}

	mc.wg.Add(1)
	go mc.flushToL3Loop()

	return mc, nil
}

// MGet 批量获取（L1 -> L2 -> L3），汇总返回
func (mc *MultiCache) MGet(keys []string) (map[string]Entity, error) {
	if len(keys) == 0 {
		return make(map[string]Entity), nil
	}

	result := make(map[string]Entity, len(keys))

	// 1. 查 L1
	l1Result, err := mc.l1.MGet(keys)
	if err != nil {
		return nil, err
	}
	for k, v := range l1Result {
		result[k] = v
		atomic.AddInt64(&mc.stats.L1Hits, 1)
	}

	// 收集 L1 miss
	missKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if _, ok := result[k]; !ok {
			missKeys = append(missKeys, k)
		}
	}
	if len(missKeys) == 0 {
		return result, nil
	}

	// 2. 查 L2 (Redis)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		l2Result, err := mc.l2.MGet(ctx, missKeys)
		cancel()

		if err != nil {
			mc.markL2Down()
		} else {
			for k, v := range l2Result {
				result[k] = v
				atomic.AddInt64(&mc.stats.L2Hits, 1)
				// 回填 L1
				mc.l1.Set(v)
			}
		}
	}

	// 收集仍然 miss 的 keys
	stillMiss := make([]string, 0, len(missKeys))
	for _, k := range missKeys {
		if _, ok := result[k]; !ok {
			stillMiss = append(stillMiss, k)
		}
	}
	if len(stillMiss) == 0 {
		return result, nil
	}

	// 3. 查 L3 (数据库)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	l3Result, err := mc.l3.MGet(ctx, stillMiss)
	cancel()

	if err != nil {
		return result, err
	}

	if len(l3Result) == 0 {
		atomic.AddInt64(&mc.stats.Misses, int64(len(stillMiss)))
		return result, nil
	}

	// 计算实际 miss 数
	missCount := 0
	for _, k := range stillMiss {
		if _, ok := l3Result[k]; !ok {
			missCount++
		}
	}
	atomic.AddInt64(&mc.stats.Misses, int64(missCount))
	atomic.AddInt64(&mc.stats.L3Hits, int64(len(l3Result)))

	// 回填 L1 和 L2
	for k, v := range l3Result {
		result[k] = v
		mc.l1.Set(v)
	}
	if mc.l2 != nil && !mc.isL2Down() && len(l3Result) > 0 {
		entities := make([]Entity, 0, len(l3Result))
		for _, v := range l3Result {
			entities = append(entities, v)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		mc.l2.MSet(ctx, entities)
		cancel()
	}

	return result, nil
}

// Get 获取（L1 -> L2 -> L3）
func (mc *MultiCache) Get(key string) (Entity, error) {
	// 1. 查 L1
	entity, err := mc.l1.Get(key)
	if err != nil {
		return nil, err
	}
	if entity != nil {
		atomic.AddInt64(&mc.stats.L1Hits, 1)
		return entity, nil
	}

	// 2. 查 L2 (Redis)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		entity, err = mc.l2.Get(ctx, key)
		cancel()

		if err != nil {
			mc.markL2Down()
		} else if entity != nil {
			atomic.AddInt64(&mc.stats.L2Hits, 1)
			// 回填 L1
			mc.l1.Set(entity)
			return entity, nil
		}
	}

	// 3. 查 L3 (数据库)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	entity, err = mc.l3.Get(ctx, key)
	cancel()

	if err != nil {
		return nil, err
	}
	if entity == nil {
		atomic.AddInt64(&mc.stats.Misses, 1)
		return nil, nil
	}

	atomic.AddInt64(&mc.stats.L3Hits, 1)

	// 回填 L1 和 L2
	mc.l1.Set(entity)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		mc.l2.Set(ctx, entity)
		cancel()
	}

	return entity, nil
}

// Set 设置
func (mc *MultiCache) Set(entity Entity) error {
	// 写入 L1
	if err := mc.l1.Set(entity); err != nil {
		return err
	}

	// 可选：同步写入 L2
	if mc.config.WriteToL2OnSet && mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := mc.l2.Set(ctx, entity); err != nil {
			// 策略1：回滚 L1 并返回错误（强一致）
			mc.l1.Delete(entity.CacheKey())
			return err

			// 策略2：标记 L2 Down 并降级为 L1+L3（高可用）
			// mc.markL2Down()
		}
	}

	return nil
}

// Delete 删除
func (mc *MultiCache) Delete(key string) error {
	mc.l1.Delete(key)

	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		mc.l2.Delete(ctx, key)
		cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mc.l3.Delete(ctx, key)
}

// Flush 立即刷盘到数据库
func (mc *MultiCache) Flush() error {
	return mc.l1.Flush()
}

// Stats 获取统计
func (mc *MultiCache) Stats() MultiCacheStats {
	return MultiCacheStats{
		L1Hits: atomic.LoadInt64(&mc.stats.L1Hits),
		L2Hits: atomic.LoadInt64(&mc.stats.L2Hits),
		L3Hits: atomic.LoadInt64(&mc.stats.L3Hits),
		Misses: atomic.LoadInt64(&mc.stats.Misses),
	}
}

// Close 关闭
func (mc *MultiCache) Close() error {
	close(mc.stopCh)
	mc.wg.Wait()

	// 最后刷盘
	mc.Flush()

	if mc.l2 != nil {
		mc.l2.Close()
	}
	mc.l3.Close()

	return nil
}

func (mc *MultiCache) isL2Down() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.l2Down
}

func (mc *MultiCache) markL2Down() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.l2Down = true
}

// syncToL2Loop 后台同步 L1 到 L2
func (mc *MultiCache) syncToL2Loop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.syncToL2()
		case <-mc.stopCh:
			return
		}
	}
}

// syncToL2 将 L1 中发生变动的数据同步到 L2
func (mc *MultiCache) syncToL2() {
	if mc.l2 == nil || mc.isL2Down() {
		return
	}

	// 只获取自上次同步以来有变动的 key
	keys := mc.l1.pullL2DirtyKeys()
	if len(keys) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 批量获取并写入 L2（已删除的 key 也会被正常覆盖/同步）
	batch := make([]Entity, 0, 100)
	for _, key := range keys {
		entity, err := mc.l1.Get(key)
		if err != nil || entity == nil {
			continue
		}
		batch = append(batch, entity)

		if len(batch) >= 100 {
			mc.l2.MSet(ctx, batch)
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		mc.l2.MSet(ctx, batch)
	}
}

// flushToL3Loop 后台刷盘到 L3
func (mc *MultiCache) flushToL3Loop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.l1.Flush()
		case <-mc.stopCh:
			mc.l1.Flush()
			return
		}
	}
}

// multiCacheStore 是 L1 缓存的存储适配器
type multiCacheStore struct {
	mc *MultiCache
}

func (s *multiCacheStore) Get(ctx context.Context, key string) (Entity, error) {
	return nil, nil // L1 不从这读
}

func (s *multiCacheStore) MGet(ctx context.Context, keys []string) (map[string]Entity, error) {
	return make(map[string]Entity), nil
}

func (s *multiCacheStore) Insert(ctx context.Context, entity Entity) error {
	return s.mc.l3.Insert(ctx, entity)
}

func (s *multiCacheStore) Update(ctx context.Context, entity Entity) error {
	return s.mc.l3.Update(ctx, entity)
}

func (s *multiCacheStore) Delete(ctx context.Context, key string) error {
	return s.mc.l3.Delete(ctx, key)
}

func (s *multiCacheStore) BatchInsert(ctx context.Context, entities []Entity) error {
	return s.mc.l3.BatchInsert(ctx, entities)
}

func (s *multiCacheStore) BatchUpdate(ctx context.Context, entities []Entity) error {
	return s.mc.l3.BatchUpdate(ctx, entities)
}

func (s *multiCacheStore) BatchDelete(ctx context.Context, keys []string) error {
	return s.mc.l3.BatchDelete(ctx, keys)
}

func (s *multiCacheStore) Close() error {
	return nil
}
