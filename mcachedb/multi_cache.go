package mcachedb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// multiCache 默认与内部常量（不适合走配置文件）
const (
	defaultMultiCacheL1MaxSize         = 100000
	defaultMultiCacheL1CleanupInterval = 5 * time.Minute
	defaultMultiCacheSyncInterval      = 500 * time.Millisecond
	defaultMultiCacheFlushInterval     = 1 * time.Second

	l2Timeout                  = 1 * time.Second
	l3Timeout                  = 5 * time.Second
	syncToL2Timeout            = 10 * time.Second
	syncToL2BatchSize          = 100
	l2RecoveryInterval         = 30 * time.Second
	l2RecoverySuccessThreshold = 2
)

// MultiCache 多级缓存 (L1内存 -> L2Redis -> L3数据库)
type MultiCache struct {
	l1                  *Cache  // 一级缓存：本地内存
	l2                  L2Store // 二级缓存：Redis
	l3                  DBStore // 三级存储：数据库
	config              *MultiCacheConfig
	stats               MultiCacheStats
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	closeOnce           sync.Once
	mu                  sync.RWMutex
	l2Down              bool
	l2RecoverySuccesses int // 连续探测成功次数
}

// MultiCacheConfig 多级缓存配置
type MultiCacheConfig struct {
	L1MaxSize         int
	L1CleanupInterval time.Duration
	WriteToL2OnSet    bool
	SyncInterval      time.Duration
	FlushInterval     time.Duration
	WriteMode         WriteMode // L1 写入模式
}

// DefaultMultiCacheConfig 默认配置
func DefaultMultiCacheConfig() *MultiCacheConfig {
	return &MultiCacheConfig{
		L1MaxSize:         defaultMultiCacheL1MaxSize,
		L1CleanupInterval: defaultMultiCacheL1CleanupInterval,
		WriteToL2OnSet:    false,
		SyncInterval:      defaultMultiCacheSyncInterval,
		FlushInterval:     defaultMultiCacheFlushInterval,
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
func NewMultiCache(dbStore DBStore, l2Store L2Store, config *MultiCacheConfig) (*MultiCache, error) {
	if dbStore == nil {
		return nil, fmt.Errorf("dbStore cannot be nil")
	}

	if config == nil {
		config = DefaultMultiCacheConfig()
	}
	if config.SyncInterval <= 0 {
		config.SyncInterval = defaultMultiCacheSyncInterval
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultMultiCacheFlushInterval
	}
	if config.L1CleanupInterval <= 0 {
		config.L1CleanupInterval = defaultMultiCacheL1CleanupInterval
	}

	mc := &MultiCache{
		l3:     dbStore,
		l2:     l2Store,
		config: config,
		stopCh: make(chan struct{}),
	}

	// 创建 L1 内存缓存（固定使用异步模式，写入策略由 MultiCache 层统一控制）
	l1Store := &multiCacheStore{mc: mc}
	var err error
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

		mc.wg.Add(1)
		go mc.l2RecoveryLoop()
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
			// L1 中已标记删除但尚未 flush，不查询 L2/L3
			if mc.l1.IsDeleted(k) {
				continue
			}
			missKeys = append(missKeys, k)
		}
	}
	if len(missKeys) == 0 {
		return result, nil
	}

	// 2. 查 L2 (Redis)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		l2Result, err := mc.l2.MGet(ctx, missKeys)
		cancel()

		if err != nil {
			mc.markL2Down()
		} else {
			for k, v := range l2Result {
				result[k] = v
				atomic.AddInt64(&mc.stats.L2Hits, 1)
				// 回填 L1（不标记 dirty，避免 flush 时重复写 L3）
				mc.l1.Load(v)
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
	ctx, cancel := context.WithTimeout(context.Background(), l3Timeout)
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
		mc.l1.Load(v)
	}
	if mc.l2 != nil && !mc.isL2Down() && len(l3Result) > 0 {
		entities := make([]Entity, 0, len(l3Result))
		for _, v := range l3Result {
			entities = append(entities, v)
		}
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
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

	// L1 中已标记删除但尚未 flush，避免查询 L2/L3 并回填旧数据
	if mc.l1.IsDeleted(key) {
		return nil, nil
	}

	// 2. 查 L2 (Redis)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		entity, err = mc.l2.Get(ctx, key)
		cancel()

		if err != nil {
			mc.markL2Down()
		} else if entity != nil {
			atomic.AddInt64(&mc.stats.L2Hits, 1)
			// 回填 L1（不标记 dirty）
			mc.l1.Load(entity)
			return entity, nil
		}
	}

	// 3. 查 L3 (数据库)
	ctx, cancel := context.WithTimeout(context.Background(), l3Timeout)
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
	mc.l1.Load(entity)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		mc.l2.Set(ctx, entity)
		cancel()
	}

	return entity, nil
}

// Set 设置
func (mc *MultiCache) Set(entity Entity) error {
	if mc.config.WriteMode == WriteModeCacheAside {
		return mc.setCacheAside(entity)
	}

	// 写入 L1
	if err := mc.l1.Set(entity); err != nil {
		return err
	}

	// 可选：同步写入 L2
	if mc.config.WriteToL2OnSet && mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		err := mc.l2.Set(ctx, entity)
		cancel()
		if err != nil {
			// 策略1：回滚 L1 并返回错误（强一致）
			mc.l1.Delete(entity.CacheKey())
			return err

			// 策略2：标记 L2 Down 并降级为 L1+L3（高可用）
			// mc.markL2Down()
		}
		// 已经同步写 L2，不需要 syncToL2Loop 再搬运一次
		mc.l1.clearL2Dirty(entity.CacheKey())
	}

	return nil
}

// setCacheAside 缓存旁路模式：先写数据库，成功后删除 L1/L2 缓存
func (mc *MultiCache) setCacheAside(entity Entity) error {
	key := entity.CacheKey()

	// 判断是插入还是更新，必须先成功读到 L3 才能决定
	ctx, cancel := context.WithTimeout(context.Background(), l3Timeout)
	existing, err := mc.l3.Get(ctx, key)
	cancel()
	if err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(context.Background(), l3Timeout)
	defer cancel()
	if existing == nil {
		err = mc.l3.Insert(ctx, entity)
	} else {
		err = mc.l3.Update(ctx, entity)
	}

	if err != nil {
		return err
	}

	// 删除 L1 内存缓存（不标记脏数据，避免后续 flush 重复写 DB）
	mc.l1.Remove(key)

	// 删除 L2 Redis 缓存
	if mc.l2 != nil && !mc.isL2Down() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
		if err := mc.l2.Delete(ctx2, key); err != nil {
			mc.markL2Down()
		}
		cancel2()
	}

	return nil
}

// Delete 删除
func (mc *MultiCache) Delete(key string) error {
	if mc.config.WriteMode == WriteModeCacheAside {
		return mc.deleteCacheAside(key)
	}

	if err := mc.l1.Delete(key); err != nil {
		return err
	}

	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		mc.l2.Delete(ctx, key)
		cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), l3Timeout)
	defer cancel()
	return mc.l3.Delete(ctx, key)
}

// deleteCacheAside 缓存旁路删除：先删数据库，再删 L1/L2
func (mc *MultiCache) deleteCacheAside(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), l3Timeout)
	if err := mc.l3.Delete(ctx, key); err != nil {
		cancel()
		return err
	}
	cancel()

	// 直接清理 L1 内存，不标记脏数据
	mc.l1.Remove(key)

	if mc.l2 != nil && !mc.isL2Down() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
		if err := mc.l2.Delete(ctx2, key); err != nil {
			mc.markL2Down()
		}
		cancel2()
	}

	return nil
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
	mc.closeOnce.Do(func() {
		close(mc.stopCh)
		mc.wg.Wait()

		// 最后刷盘
		mc.Flush()

		if mc.l2 != nil {
			mc.l2.Close()
		}
		mc.l3.Close()
	})

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
	mc.l2RecoverySuccesses = 0
}

// resetL2Down 重置 L2 可用状态（由恢复探测调用）
func (mc *MultiCache) resetL2Down() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.l2Down = false
	mc.l2RecoverySuccesses = 0
}

// l2RecoveryLoop L2 故障恢复探测循环
func (mc *MultiCache) l2RecoveryLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(l2RecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if mc.isL2Down() && mc.tryRecoverL2() {
				mc.resetL2Down()
			}
		case <-mc.stopCh:
			return
		}
	}
}

// tryRecoverL2 尝试恢复 L2，连续成功阈值达到后返回 true
func (mc *MultiCache) tryRecoverL2() bool {
	if mc.l2 == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
	defer cancel()

	if err := mc.l2.Ping(ctx); err != nil {
		mc.mu.Lock()
		mc.l2RecoverySuccesses = 0
		mc.mu.Unlock()
		return false
	}

	mc.mu.Lock()
	mc.l2RecoverySuccesses++
	successes := mc.l2RecoverySuccesses
	if successes >= l2RecoverySuccessThreshold {
		mc.l2Down = false
		mc.l2RecoverySuccesses = 0
	}
	mc.mu.Unlock()

	return successes >= l2RecoverySuccessThreshold
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

	ctx, cancel := context.WithTimeout(context.Background(), syncToL2Timeout)
	defer cancel()

	// 批量获取并写入 L2（已删除的 key 同步删除 L2）
	batch := make([]Entity, 0, syncToL2BatchSize)
	deleteKeys := make([]string, 0)
	for _, key := range keys {
		if mc.l1.IsDeleted(key) {
			deleteKeys = append(deleteKeys, key)
			continue
		}

		entity, err := mc.l1.Get(key)
		if err != nil || entity == nil {
			continue
		}
		batch = append(batch, entity)

		if len(batch) >= syncToL2BatchSize {
			if err := mc.l2.MSet(ctx, batch); err != nil {
				mc.markL2Down()
				return
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := mc.l2.MSet(ctx, batch); err != nil {
			mc.markL2Down()
			return
		}
	}

	for _, key := range deleteKeys {
		ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
		if err := mc.l2.Delete(ctx2, key); err != nil {
			mc.markL2Down()
			cancel2()
			return
		}
		cancel2()
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
