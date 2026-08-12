package mcachedb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MultiCache 内部常量
const (
	defaultMultiCacheL1MaxSize         = 100000
	defaultMultiCacheL1CleanupInterval = 5 * time.Minute
	defaultMultiCacheSyncInterval      = 500 * time.Millisecond
	defaultMultiCacheFlushInterval     = 1 * time.Second
	defaultMultiCacheBatchSize         = 100
	defaultMultiCacheRetryCount        = 3
	defaultMultiCacheRetryInterval     = 100 * time.Millisecond

	l2Timeout                  = 1 * time.Second
	l3Timeout                  = 5 * time.Second
	syncToL2Timeout            = 10 * time.Second
	syncToL2BatchSize          = 100
	l2RecoveryInterval         = 30 * time.Second
	l2RecoverySuccessThreshold = 2
	defaultFlushTimeout        = 30 * time.Second
)

// MultiCacheConfig 多级缓存配置
type MultiCacheConfig struct {
	L1MaxSize         int
	L1CleanupInterval time.Duration
	SyncInterval      time.Duration
	FlushInterval     time.Duration
	// WriteMode 写入模式：
	//   WriteModeAsync（默认）：写 L1，L2/L3 均由后台异步同步
	//   WriteModeWriteL2：Set 时同步写 L1+L2，L3 后台异步刷盘
	//   WriteModeCacheAside：先写 L3，成功后删除 L1/L2；下次读时回填
	WriteMode WriteMode
	// BatchSize 累积脏数据达到此数量时触发 flush
	BatchSize int
	// RetryCount flush 失败后重试次数
	RetryCount int
	// RetryInterval 重试间隔
	RetryInterval time.Duration
	// Logger 可选。注入后输出 flush 失败、L2 故障切换、L2 恢复等关键事件日志；
	// 传 nil 则静默。
	Logger Logger
}

// DefaultMultiCacheConfig 默认配置
func DefaultMultiCacheConfig() *MultiCacheConfig {
	return &MultiCacheConfig{
		L1MaxSize:         defaultMultiCacheL1MaxSize,
		L1CleanupInterval: defaultMultiCacheL1CleanupInterval,
		SyncInterval:      defaultMultiCacheSyncInterval,
		FlushInterval:     defaultMultiCacheFlushInterval,
		WriteMode:         WriteModeAsync,
		BatchSize:         defaultMultiCacheBatchSize,
		RetryCount:        defaultMultiCacheRetryCount,
		RetryInterval:     defaultMultiCacheRetryInterval,
	}
}

// MultiCacheStats 多级缓存统计
type MultiCacheStats struct {
	// 命中计数
	L1Hits int64
	L2Hits int64
	L3Hits int64
	Misses int64

	// L1 当前状态
	L1Size       int
	L1DirtyCount int64
	L2DirtyCount int64
	L2Down       bool

	// 刷盘统计
	FlushCount     int64
	FlushErrors    int64
	FlushTotalTime time.Duration
	LastFlushTime  time.Time
}

// HitRate 总命中率
func (s *MultiCacheStats) HitRate() float64 {
	total := s.L1Hits + s.L2Hits + s.L3Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.L1Hits+s.L2Hits+s.L3Hits) / float64(total)
}

// MultiCache 多级缓存 (L1内存 → L2 Redis → L3 数据库)
type MultiCache struct {
	l1     *Cache
	l2     L2Store
	l3     DBStore
	config *MultiCacheConfig

	// L3 flush 追踪（仅 Set 操作产生，Delete 同步执行）
	dirtyMu         sync.Mutex
	dirtyKeys       []string
	dirtyMap        map[string]bool
	dirtySeqCounter uint64
	flushMu         sync.Mutex // 序列化 flush 执行，替代 CAS flushing 标志
	flushCh         chan struct{}

	// L2 同步追踪（upsert + delete）
	l2DirtyMu   sync.Mutex
	l2DirtyKeys []string
	l2DirtyMap  map[string]bool
	l2DeleteMap map[string]bool

	// 统计
	stats   MultiCacheStats
	statsMu sync.Mutex

	// L2 健康状态
	l2Down              bool
	l2RecoverySuccesses int
	mu                  sync.RWMutex

	logger Logger

	// 生命周期
	stopCh    chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// NewMultiCache 创建多级缓存
func NewMultiCache(dbStore DBStore, l2Store L2Store, config *MultiCacheConfig) (*MultiCache, error) {
	if dbStore == nil {
		return nil, fmt.Errorf("dbStore cannot be nil")
	}
	if config == nil {
		config = DefaultMultiCacheConfig()
	}
	if config.L1MaxSize <= 0 {
		config.L1MaxSize = defaultMultiCacheL1MaxSize
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
	if config.BatchSize <= 0 {
		config.BatchSize = defaultMultiCacheBatchSize
	}
	if config.RetryCount <= 0 {
		config.RetryCount = defaultMultiCacheRetryCount
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = defaultMultiCacheRetryInterval
	}

	l1, err := New(
		WithMaxCacheSize(config.L1MaxSize),
		WithCleanupInterval(config.L1CleanupInterval),
	)
	if err != nil {
		return nil, err
	}

	logger := Logger(nopLogger{})
	if config.Logger != nil {
		logger = config.Logger
	}

	mc := &MultiCache{
		l1:          l1,
		l2:          l2Store,
		l3:          dbStore,
		config:      config,
		dirtyMap:    make(map[string]bool),
		l2DirtyMap:  make(map[string]bool),
		l2DeleteMap: make(map[string]bool),
		flushCh:     make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
		logger:      logger,
	}

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

// ─── 读路径 ───────────────────────────────────────────────────────────────────

// Get 从 L1→L2→L3 获取实体，逐级回填
func (mc *MultiCache) Get(key string) (Entity, error) {
	// L1
	entity, err := mc.l1.Get(key)
	if err != nil {
		return nil, err
	}
	if entity != nil {
		atomic.AddInt64(&mc.stats.L1Hits, 1)
		return entity, nil
	}

	// L2
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		entity, err = mc.l2.Get(ctx, key)
		cancel()
		if err != nil {
			mc.markL2Down()
		} else if entity != nil {
			atomic.AddInt64(&mc.stats.L2Hits, 1)
			mc.l1.Load(entity)
			return entity, nil
		}
	}

	// L3
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
	mc.l1.Load(entity)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel = context.WithTimeout(context.Background(), l2Timeout)
		if err := mc.l2.Set(ctx, entity); err != nil {
			mc.markL2Down()
		}
		cancel()
	}
	return entity, nil
}

// MGet 批量获取，逐级回填
func (mc *MultiCache) MGet(keys []string) (map[string]Entity, error) {
	if len(keys) == 0 {
		return make(map[string]Entity), nil
	}

	result := make(map[string]Entity, len(keys))

	// L1
	l1Result, err := mc.l1.MGet(keys)
	if err != nil {
		return nil, err
	}
	for k, v := range l1Result {
		result[k] = v
		atomic.AddInt64(&mc.stats.L1Hits, 1)
	}

	missKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if _, ok := result[k]; !ok {
			missKeys = append(missKeys, k)
		}
	}
	if len(missKeys) == 0 {
		return result, nil
	}

	// L2
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
				mc.l1.Load(v)
			}
		}
	}

	stillMiss := make([]string, 0, len(missKeys))
	for _, k := range missKeys {
		if _, ok := result[k]; !ok {
			stillMiss = append(stillMiss, k)
		}
	}
	if len(stillMiss) == 0 {
		return result, nil
	}

	// L3
	ctx, cancel := context.WithTimeout(context.Background(), l3Timeout)
	l3Result, err := mc.l3.MGet(ctx, stillMiss)
	cancel()
	if err != nil {
		return result, err
	}

	missCount := 0
	for _, k := range stillMiss {
		if _, ok := l3Result[k]; !ok {
			missCount++
		}
	}
	atomic.AddInt64(&mc.stats.Misses, int64(missCount))
	atomic.AddInt64(&mc.stats.L3Hits, int64(len(l3Result)))

	for k, v := range l3Result {
		result[k] = v
		mc.l1.Load(v)
	}
	if mc.l2 != nil && !mc.isL2Down() && len(l3Result) > 0 {
		entities := make([]Entity, 0, len(l3Result))
		for _, v := range l3Result {
			entities = append(entities, v)
		}
		ctx, cancel = context.WithTimeout(context.Background(), l2Timeout)
		if err := mc.l2.MSet(ctx, entities); err != nil {
			mc.markL2Down()
		}
		cancel()
	}

	return result, nil
}

// ─── 写路径 ───────────────────────────────────────────────────────────────────

// Set 写入实体
func (mc *MultiCache) Set(entity Entity) error {
	if err := validateEntity(entity); err != nil {
		return err
	}
	if mc.config.WriteMode == WriteModeCacheAside {
		return mc.setCacheAside(entity)
	}
	return mc.setAsync(entity)
}

// setAsync 写 L1（含 dirty 标记），后台异步同步 L2/L3
func (mc *MultiCache) setAsync(entity Entity) error {
	key := entity.CacheKey()
	entity.IncrementVersion()

	seq := atomic.AddUint64(&mc.dirtySeqCounter, 1)
	if !mc.l1.setEntry(entity, seq) {
		return fmt.Errorf("L1 cache is full, cannot write key %s", key)
	}

	mc.addDirty(key)
	mc.addL2Upsert(key)

	if mc.config.WriteMode == WriteModeWriteL2 && mc.l2 != nil && !mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		err := mc.l2.Set(ctx, entity)
		cancel()
		if err != nil {
			mc.markL2Down()
			return err
		}
		// 已同步写 L2，不需要 syncToL2Loop 再搬运
		mc.clearL2Upsert(key)
	}

	return nil
}

// setCacheAside 先写 L3，成功后删 L1/L2
func (mc *MultiCache) setCacheAside(entity Entity) error {
	key := entity.CacheKey()

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
		entity.IncrementVersion()
		err = mc.l3.Update(ctx, entity)
	}
	if err != nil {
		return err
	}

	mc.l1.Remove(key)
	if mc.l2 != nil && !mc.isL2Down() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
		if err2 := mc.l2.Delete(ctx2, key); err2 != nil {
			mc.markL2Down()
		}
		cancel2()
	}
	return nil
}

// Delete 删除实体：先移除 L1，再同步删 L3，最后删 L2。
// 先移除 L1 可防止在 L3 删除期间被重新回填；L3 同步执行保证其他节点立即可见。
func (mc *MultiCache) Delete(key string) error {
	mc.l1.Remove(key)

	ctx, cancel := context.WithTimeout(context.Background(), l3Timeout)
	err := mc.l3.Delete(ctx, key)
	cancel()
	if err != nil {
		return err
	}

	if mc.l2 != nil && !mc.isL2Down() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
		if err2 := mc.l2.Delete(ctx2, key); err2 != nil {
			mc.markL2Down()
			mc.addL2Delete(key)
		}
		cancel2()
	} else if mc.l2 != nil {
		mc.addL2Delete(key)
	}
	return nil
}

// ─── Flush ────────────────────────────────────────────────────────────────────

// Flush 立即将 L1 脏数据刷盘到 L3，阻塞直到本次 flush 完成。
func (mc *MultiCache) Flush() error {
	return mc.doFlush()
}

// FlushToL3 Flush 的别名
func (mc *MultiCache) FlushToL3() error {
	return mc.doFlush()
}

// SyncToL2 立即将待同步变更推送到 L2
func (mc *MultiCache) SyncToL2() {
	mc.syncToL2()
}

// doFlush 阻塞式刷盘：等待当前正在进行的 flush 完成后再执行。
// 供公开 Flush() 和 Close() 调用，保证返回时数据已落盘。
func (mc *MultiCache) doFlush() error {
	mc.flushMu.Lock()
	defer mc.flushMu.Unlock()
	return mc.flush()
}

// tryFlush 非阻塞式刷盘：若已有 flush 在进行则跳过。
// 供后台 goroutine 调用，避免因 flush 耗时过长导致 goroutine 堆积。
func (mc *MultiCache) tryFlush() error {
	if !mc.flushMu.TryLock() {
		return nil
	}
	defer mc.flushMu.Unlock()
	return mc.flush()
}

// flush 实际刷盘逻辑，调用前须持有 flushMu。
func (mc *MultiCache) flush() error {
	snapshotSeq := atomic.LoadUint64(&mc.dirtySeqCounter)

	mc.dirtyMu.Lock()
	if len(mc.dirtyKeys) == 0 {
		mc.dirtyMu.Unlock()
		return nil
	}
	dirtyKeys := make([]string, len(mc.dirtyKeys))
	copy(dirtyKeys, mc.dirtyKeys)
	mc.dirtyKeys = mc.dirtyKeys[:0]
	mc.dirtyMap = make(map[string]bool)
	mc.dirtyMu.Unlock()

	toInsert, toUpdate := mc.l1.getDirtyEntities(dirtyKeys)

	if len(toInsert) == 0 && len(toUpdate) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultFlushTimeout)
	defer cancel()

	startTime := time.Now()
	var err error

	if len(toInsert) > 0 {
		for i := 0; i < mc.config.RetryCount; i++ {
			err = mc.l3.BatchInsert(ctx, toInsert)
			if err == nil {
				break
			}
			time.Sleep(mc.config.RetryInterval)
		}
		if err != nil {
			mc.logger.Errorf("mcachedb: BatchInsert failed after %d retries (count=%d): %v", mc.config.RetryCount, len(toInsert), err)
			mc.handleFlushError(dirtyKeys)
			return err
		}
		// BatchInsert 已落盘，立即清除 isNew 标记。
		// 若随后 BatchUpdate 失败需要重入队，这批 key 将以 Update 路径重试，避免重复 Insert。
		mc.l1.clearDirtyEntries(entityKeys(toInsert), snapshotSeq)
	}

	if len(toUpdate) > 0 {
		for i := 0; i < mc.config.RetryCount; i++ {
			err = mc.l3.BatchUpdate(ctx, toUpdate)
			if err == nil {
				break
			}
			time.Sleep(mc.config.RetryInterval)
		}
		if err != nil {
			mc.logger.Errorf("mcachedb: BatchUpdate failed after %d retries (count=%d): %v", mc.config.RetryCount, len(toUpdate), err)
			// 仅将 Update 失败的 key 重新入队；Insert 已成功，无需重入
			mc.handleFlushError(entityKeys(toUpdate))
			return err
		}
		mc.l1.clearDirtyEntries(entityKeys(toUpdate), snapshotSeq)
	}

	duration := time.Since(startTime)
	atomic.AddInt64(&mc.stats.FlushCount, 1)
	mc.statsMu.Lock()
	mc.stats.FlushTotalTime += duration
	mc.stats.LastFlushTime = time.Now()
	mc.statsMu.Unlock()

	return nil
}

// entityKeys 提取实体列表中的 CacheKey
func entityKeys(entities []Entity) []string {
	keys := make([]string, len(entities))
	for i, e := range entities {
		keys[i] = e.CacheKey()
	}
	return keys
}

func (mc *MultiCache) handleFlushError(dirtyKeys []string) {
	atomic.AddInt64(&mc.stats.FlushErrors, 1)
	mc.dirtyMu.Lock()
	for _, key := range dirtyKeys {
		if !mc.dirtyMap[key] {
			mc.dirtyMap[key] = true
			mc.dirtyKeys = append(mc.dirtyKeys, key)
		}
	}
	mc.dirtyMu.Unlock()
}

// ─── L3 脏数据队列 ─────────────────────────────────────────────────────────────

func (mc *MultiCache) addDirty(key string) {
	mc.dirtyMu.Lock()
	if !mc.dirtyMap[key] {
		mc.dirtyMap[key] = true
		mc.dirtyKeys = append(mc.dirtyKeys, key)
	}
	needFlush := len(mc.dirtyKeys) >= mc.config.BatchSize
	mc.dirtyMu.Unlock()

	if needFlush {
		mc.triggerFlush()
	}
}

func (mc *MultiCache) triggerFlush() {
	select {
	case mc.flushCh <- struct{}{}:
	default:
	}
}

func (mc *MultiCache) dirtyCount() int {
	mc.dirtyMu.Lock()
	n := len(mc.dirtyKeys)
	mc.dirtyMu.Unlock()
	return n
}

// ─── L2 同步队列 ──────────────────────────────────────────────────────────────

func (mc *MultiCache) addL2Upsert(key string) {
	mc.l2DirtyMu.Lock()
	defer mc.l2DirtyMu.Unlock()
	if !mc.l2DirtyMap[key] {
		mc.l2DirtyMap[key] = true
		mc.l2DirtyKeys = append(mc.l2DirtyKeys, key)
	}
	// 覆盖同窗口内先前的 Delete 操作，确保 Set 后 L2 执行 upsert 而非 delete
	delete(mc.l2DeleteMap, key)
}

func (mc *MultiCache) addL2Delete(key string) {
	mc.l2DirtyMu.Lock()
	defer mc.l2DirtyMu.Unlock()
	if !mc.l2DirtyMap[key] {
		mc.l2DirtyMap[key] = true
		mc.l2DirtyKeys = append(mc.l2DirtyKeys, key)
	}
	mc.l2DeleteMap[key] = true
}

func (mc *MultiCache) clearL2Upsert(key string) {
	mc.l2DirtyMu.Lock()
	defer mc.l2DirtyMu.Unlock()
	if !mc.l2DirtyMap[key] {
		return
	}
	delete(mc.l2DirtyMap, key)
	delete(mc.l2DeleteMap, key)
	filtered := make([]string, 0, len(mc.l2DirtyKeys))
	for _, k := range mc.l2DirtyKeys {
		if k != key {
			filtered = append(filtered, k)
		}
	}
	mc.l2DirtyKeys = filtered
}

func (mc *MultiCache) pullL2DirtyKeys() (upsert, deletes []string) {
	mc.l2DirtyMu.Lock()
	defer mc.l2DirtyMu.Unlock()
	if len(mc.l2DirtyKeys) == 0 {
		return nil, nil
	}
	for _, k := range mc.l2DirtyKeys {
		if mc.l2DeleteMap[k] {
			deletes = append(deletes, k)
		} else {
			upsert = append(upsert, k)
		}
	}
	mc.l2DirtyKeys = mc.l2DirtyKeys[:0]
	mc.l2DirtyMap = make(map[string]bool)
	mc.l2DeleteMap = make(map[string]bool)
	return
}

func (mc *MultiCache) l2DirtyCount() int {
	mc.l2DirtyMu.Lock()
	n := len(mc.l2DirtyKeys)
	mc.l2DirtyMu.Unlock()
	return n
}

// ─── 后台 goroutine ───────────────────────────────────────────────────────────

func (mc *MultiCache) flushToL3Loop() {
	defer mc.wg.Done()
	ticker := time.NewTicker(mc.config.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			mc.tryFlush() // 后台定时触发：非阻塞，已在刷则跳过
		case <-mc.flushCh:
			mc.tryFlush() // 批量阈值触发：同上
		case <-mc.stopCh:
			mc.doFlush() // 关闭时：阻塞等待最终刷盘完成
			return
		}
	}
}

func (mc *MultiCache) syncToL2Loop() {
	defer mc.wg.Done()
	ticker := time.NewTicker(mc.config.SyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			mc.syncToL2()
		case <-mc.stopCh:
			mc.syncToL2() // 关闭时做最终同步，防止 L2 脏数据丢失
			return
		}
	}
}

func (mc *MultiCache) syncToL2() {
	if mc.l2 == nil || mc.isL2Down() {
		return
	}
	upsertKeys, deleteKeys := mc.pullL2DirtyKeys()
	if len(upsertKeys) == 0 && len(deleteKeys) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncToL2Timeout)
	defer cancel()

	batch := make([]Entity, 0, syncToL2BatchSize)
	for _, key := range upsertKeys {
		entity, _ := mc.l1.Get(key)
		if entity == nil {
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

// ─── L2 健康状态 ──────────────────────────────────────────────────────────────

func (mc *MultiCache) isL2Down() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.l2Down
}

func (mc *MultiCache) markL2Down() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if !mc.l2Down {
		mc.logger.Warnf("mcachedb: L2 store is down, falling back to L3-only mode")
	}
	mc.l2Down = true
	mc.l2RecoverySuccesses = 0
}

func (mc *MultiCache) resetL2Down() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.l2Down = false
	mc.l2RecoverySuccesses = 0
	mc.logger.Warnf("mcachedb: L2 store recovered, resuming L2 sync")
}

// ─── 统计与生命周期 ───────────────────────────────────────────────────────────

// Stats 返回命中统计及 L1/flush 状态
func (mc *MultiCache) Stats() MultiCacheStats {
	mc.statsMu.Lock()
	flushTotalTime := mc.stats.FlushTotalTime
	lastFlushTime := mc.stats.LastFlushTime
	mc.statsMu.Unlock()

	return MultiCacheStats{
		L1Hits:         atomic.LoadInt64(&mc.stats.L1Hits),
		L2Hits:         atomic.LoadInt64(&mc.stats.L2Hits),
		L3Hits:         atomic.LoadInt64(&mc.stats.L3Hits),
		Misses:         atomic.LoadInt64(&mc.stats.Misses),
		L1Size:         mc.l1.Len(),
		L1DirtyCount:   int64(mc.dirtyCount()),
		L2DirtyCount:   int64(mc.l2DirtyCount()),
		L2Down:         mc.isL2Down(),
		FlushCount:     atomic.LoadInt64(&mc.stats.FlushCount),
		FlushErrors:    atomic.LoadInt64(&mc.stats.FlushErrors),
		FlushTotalTime: flushTotalTime,
		LastFlushTime:  lastFlushTime,
	}
}

// Transaction 在 CacheAside 事务中执行 fn（仅 WriteModeCacheAside 模式可用）。
// fn 应通过 tx.Set / tx.Delete / tx.Get 登记操作。
// fn 返回 nil 则提交，返回非 nil 则取消（L3 尚未写入，无需额外回滚）。
func (mc *MultiCache) Transaction(fn func(*Tx) error) error {
	if mc.config.WriteMode != WriteModeCacheAside {
		return fmt.Errorf("Transaction requires WriteModeCacheAside; current mode: %d", mc.config.WriteMode)
	}
	tx := newTx(mc)
	defer tx.cancel()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.commit()
}

// Close 关闭缓存，最后刷盘
func (mc *MultiCache) Close() error {
	mc.closeOnce.Do(func() {
		close(mc.stopCh)
		mc.wg.Wait()
		mc.doFlush()
		if mc.l2 != nil {
			mc.l2.Close()
		}
		mc.l3.Close()
		mc.l1.Close()
	})
	return nil
}
