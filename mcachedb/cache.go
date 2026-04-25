package mcachedb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Cache 带数据库持久化的缓存
type Cache struct {
	config *Config
	store  DBStore

	// 内存缓存
	mu   sync.RWMutex
	data map[string]*entry

	// 脏数据队列（带顺序）
	dirtyMu   sync.Mutex
	dirtyKeys []string
	dirtyMap  map[string]bool

	// L2 待同步队列（只同步变动数据）
	l2DirtyMu   sync.Mutex
	l2DirtyKeys []string
	l2DirtyMap  map[string]bool

	// 控制
	stopCh  chan struct{}
	flushCh chan struct{} // 触发刷新的信号
	wg      sync.WaitGroup

	// 统计
	stats Stats

	// 刷新状态
	flushing int32

	// 写入队列（用于异步写入工作池）
	writeCh chan *writeTask
}

// writeTask 写入任务
type writeTask struct {
	ctx     context.Context
	entries []Entity
	done    chan error
}

// New 创建缓存
func New(store DBStore, opts ...Option) (*Cache, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	c := &Cache{
		config:      config,
		store:       store,
		data:        make(map[string]*entry),
		dirtyKeys:   make([]string, 0),
		dirtyMap:    make(map[string]bool),
		l2DirtyKeys: make([]string, 0),
		l2DirtyMap:  make(map[string]bool),
		stopCh:      make(chan struct{}),
		flushCh:     make(chan struct{}, 1),
		writeCh:     make(chan *writeTask, 100),
	}

	// 启动后台协程
	if config.FlushMode == FlushModeInterval {
		c.wg.Add(1)
		go c.flushLoop()
	}

	// 启动写入工作协程
	for i := 0; i < config.WriteWorkers; i++ {
		c.wg.Add(1)
		go c.writeWorker()
	}

	// 启动清理协程
	if config.CleanupInterval > 0 {
		c.wg.Add(1)
		go c.cleanupLoop()
	}

	return c, nil
}

// Get 获取实体
// 1. 先查缓存
// 2. 缓存未命中查数据库
// 3. 回填缓存
func (c *Cache) Get(key string) (Entity, error) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()

	if ok && !e.isExpired() && !e.deleted {
		atomic.AddInt64(&c.stats.CacheHits, 1)
		if c.config.OnCacheHit != nil {
			c.config.OnCacheHit(key)
		}
		return e.entity.Copy(), nil
	}

	atomic.AddInt64(&c.stats.CacheMisses, 1)
	if c.config.OnCacheMiss != nil {
		c.config.OnCacheMiss(key)
	}

	// 查数据库
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entity, err := c.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}

	// 回填缓存
	c.mu.Lock()
	c.data[key] = &entry{
		entity:    entity.Copy(),
		createdAt: time.Now(),
		expireAt:  c.getExpireTime(),
		dirty:     false,
		deleted:   false,
		isNew:     false, // 从数据库回填的不是新记录
	}
	c.mu.Unlock()

	atomic.AddInt64(&c.stats.DBReads, 1)

	return entity, nil
}

// MGet 批量获取
func (c *Cache) MGet(keys []string) (map[string]Entity, error) {
	result := make(map[string]Entity)
	missKeys := make([]string, 0)

	// 查缓存
	c.mu.RLock()
	for _, key := range keys {
		if e, ok := c.data[key]; ok && !e.isExpired() && !e.deleted {
			result[key] = e.entity.Copy()
			atomic.AddInt64(&c.stats.CacheHits, 1)
		} else {
			missKeys = append(missKeys, key)
		}
	}
	c.mu.RUnlock()

	if len(missKeys) == 0 {
		return result, nil
	}

	atomic.AddInt64(&c.stats.CacheMisses, int64(len(missKeys)))

	// 查数据库
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entities, err := c.store.MGet(ctx, missKeys)
	if err != nil {
		return result, err
	}

	// 回填缓存
	c.mu.Lock()
	for key, entity := range entities {
		result[key] = entity
		c.data[key] = &entry{
			entity:    entity.Copy(),
			createdAt: time.Now(),
			expireAt:  c.getExpireTime(),
			dirty:     false,
			deleted:   false,
			isNew:     false,
		}
	}
	c.mu.Unlock()

	atomic.AddInt64(&c.stats.DBReads, 1)

	return result, nil
}

// Set 设置实体
func (c *Cache) Set(entity Entity) error {
	if entity == nil {
		return fmt.Errorf("entity cannot be nil")
	}

	key := entity.CacheKey()
	if key == "" {
		return fmt.Errorf("entity key cannot be empty")
	}

	switch c.config.WriteMode {
	case WriteModeSync:
		return c.setSync(entity)
	case WriteModeWriteThrough:
		return c.setWriteThrough(entity)
	default: // WriteModeAsync
		return c.setAsync(entity)
	}
}

// setAsync 异步写入（默认）
func (c *Cache) setAsync(entity Entity) error {
	key := entity.CacheKey()

	// 增加版本号
	entity.IncrementVersion()

	c.mu.Lock()
	// 检查缓存大小限制
	if len(c.data) >= c.config.MaxCacheSize {
		if _, ok := c.data[key]; !ok {
			c.mu.Unlock()
			return fmt.Errorf("cache is full")
		}
	}

	c.data[key] = &entry{
		entity:    entity.Copy(),
		createdAt: time.Now(),
		expireAt:  c.getExpireTime(),
		dirty:     true,
		deleted:   false,
		isNew:     true,
	}
	c.mu.Unlock()

	// 添加到脏队列
	c.addDirty(key)
	c.addL2Dirty(key)

	// 检查是否需要触发刷新
	if c.config.FlushMode == FlushModeImmediate {
		c.triggerFlush()
	}

	return nil
}

// setSync 同步写入
func (c *Cache) setSync(entity Entity) error {
	key := entity.CacheKey()

	// 先写数据库
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 判断是插入还是更新
	existing, _ := c.Get(key)
	var err error
	if existing == nil {
		err = c.store.Insert(ctx, entity)
	} else {
		err = c.store.Update(ctx, entity)
	}

	if err != nil {
		return err
	}

	atomic.AddInt64(&c.stats.DBWrites, 1)

	// 更新缓存
	c.mu.Lock()
	c.data[key] = &entry{
		entity:    entity.Copy(),
		createdAt: time.Now(),
		expireAt:  c.getExpireTime(),
		dirty:     false,
		deleted:   false,
		isNew:     false, // 同步写入后不再是新记录
	}
	c.mu.Unlock()

	c.addL2Dirty(key)

	return nil
}

// setWriteThrough 直写模式
func (c *Cache) setWriteThrough(entity Entity) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 先写数据库
	var err error
	existing, _ := c.store.Get(ctx, entity.CacheKey())
	if existing == nil {
		err = c.store.Insert(ctx, entity)
	} else {
		err = c.store.Update(ctx, entity)
	}

	if err != nil {
		return err
	}

	// 成功后更新缓存
	return c.setAsync(entity)
}

// Delete 删除
func (c *Cache) Delete(key string) error {
	switch c.config.WriteMode {
	case WriteModeSync:
		return c.deleteSync(key)
	case WriteModeWriteThrough:
		return c.deleteWriteThrough(key)
	default:
		return c.deleteAsync(key)
	}
}

// deleteAsync 异步删除
func (c *Cache) deleteAsync(key string) error {
	c.mu.Lock()
	if e, ok := c.data[key]; ok {
		e.deleted = true
		e.dirty = true
		e.entity.SetDeleted(true)
	}
	c.mu.Unlock()

	c.addDirty(key)
	c.addL2Dirty(key)

	if c.config.FlushMode == FlushModeImmediate {
		c.triggerFlush()
	}

	return nil
}

// deleteSync 同步删除
func (c *Cache) deleteSync(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.store.Delete(ctx, key); err != nil {
		return err
	}

	atomic.AddInt64(&c.stats.DBWrites, 1)

	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()

	return nil
}

// deleteWriteThrough 直写删除
func (c *Cache) deleteWriteThrough(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.store.Delete(ctx, key); err != nil {
		return err
	}

	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()

	return nil
}

// addDirty 添加到脏队列
func (c *Cache) addDirty(key string) {
	c.dirtyMu.Lock()
	defer c.dirtyMu.Unlock()

	if !c.dirtyMap[key] {
		c.dirtyMap[key] = true
		c.dirtyKeys = append(c.dirtyKeys, key)
	}

	// 检查是否达到批量大小
	if len(c.dirtyKeys) >= c.config.BatchSize {
		c.dirtyMu.Unlock()
		c.triggerFlush()
		c.dirtyMu.Lock()
	}
}

// triggerFlush 触发刷新
func (c *Cache) triggerFlush() {
	select {
	case c.flushCh <- struct{}{}:
	default:
	}
}

// Flush 立即刷新到数据库
func (c *Cache) Flush() error {
	return c.doFlush()
}

// doFlush 执行刷新
func (c *Cache) doFlush() error {
	// 检查是否正在刷新
	if !atomic.CompareAndSwapInt32(&c.flushing, 0, 1) {
		return nil
	}
	defer atomic.StoreInt32(&c.flushing, 0)

	// 获取脏数据
	c.dirtyMu.Lock()
	if len(c.dirtyKeys) == 0 {
		c.dirtyMu.Unlock()
		return nil
	}

	dirtyKeys := make([]string, len(c.dirtyKeys))
	copy(dirtyKeys, c.dirtyKeys)
	c.dirtyKeys = c.dirtyKeys[:0]
	c.dirtyMap = make(map[string]bool)
	c.dirtyMu.Unlock()

	// 收集实体
	c.mu.RLock()
	toInsert := make([]Entity, 0)
	toUpdate := make([]Entity, 0)
	toDelete := make([]string, 0)

	for _, key := range dirtyKeys {
		if e, ok := c.data[key]; ok {
			if e.deleted {
				toDelete = append(toDelete, key)
			} else if e.dirty {
				// 通过 isNew 标记判断是否是新增
				if e.isNew {
					toInsert = append(toInsert, e.entity)
				} else {
					toUpdate = append(toUpdate, e.entity)
				}
			}
		}
	}
	c.mu.RUnlock()

	if c.config.OnFlushStart != nil {
		c.config.OnFlushStart(len(dirtyKeys))
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 执行写入
	var err error

	// 批量删除
	if len(toDelete) > 0 {
		for i := 0; i < c.config.RetryCount; i++ {
			err = c.store.BatchDelete(ctx, toDelete)
			if err == nil {
				break
			}
			time.Sleep(c.config.RetryInterval)
		}
		if err != nil {
			c.handleFlushError(err, dirtyKeys)
			return err
		}
	}

	// 批量插入
	if len(toInsert) > 0 && err == nil {
		for i := 0; i < c.config.RetryCount; i++ {
			err = c.store.BatchInsert(ctx, toInsert)
			if err == nil {
				break
			}
			time.Sleep(c.config.RetryInterval)
		}
		if err != nil {
			c.handleFlushError(err, dirtyKeys)
			return err
		}
	}

	// 批量更新
	if len(toUpdate) > 0 && err == nil {
		for i := 0; i < c.config.RetryCount; i++ {
			err = c.store.BatchUpdate(ctx, toUpdate)
			if err == nil {
				break
			}
			time.Sleep(c.config.RetryInterval)
		}
		if err != nil {
			c.handleFlushError(err, dirtyKeys)
			return err
		}
	}

	duration := time.Since(startTime)

	// 清除 dirty 标记和 isNew 标记
	c.mu.Lock()
	for _, key := range dirtyKeys {
		if e, ok := c.data[key]; ok {
			e.dirty = false
			e.isNew = false // 写入成功后不再是新记录
			if e.deleted {
				delete(c.data, key)
			}
		}
	}
	c.mu.Unlock()

	// 更新统计
	atomic.AddInt64(&c.stats.FlushCount, 1)
	atomic.AddInt64(&c.stats.DBWrites, int64(len(dirtyKeys)))
	c.stats.LastFlushTime = time.Now()
	c.stats.FlushTotalTime += duration

	if c.config.OnFlushSuccess != nil {
		c.config.OnFlushSuccess(len(dirtyKeys), duration)
	}

	return nil
}

// handleFlushError 处理刷新错误
func (c *Cache) handleFlushError(err error, keys []string) {
	atomic.AddInt64(&c.stats.DBWriteErrors, 1)

	if c.config.OnFlushError != nil {
		entities := make([]Entity, 0, len(keys))
		c.mu.RLock()
		for _, key := range keys {
			if e, ok := c.data[key]; ok {
				entities = append(entities, e.entity)
			}
		}
		c.mu.RUnlock()
		c.config.OnFlushError(err, entities)
	}

	// 重新加入脏队列
	c.dirtyMu.Lock()
	for _, key := range keys {
		if !c.dirtyMap[key] {
			c.dirtyMap[key] = true
			c.dirtyKeys = append(c.dirtyKeys, key)
		}
	}
	c.dirtyMu.Unlock()
}

// addL2Dirty 添加到 L2 待同步队列
func (c *Cache) addL2Dirty(key string) {
	c.l2DirtyMu.Lock()
	defer c.l2DirtyMu.Unlock()

	if !c.l2DirtyMap[key] {
		c.l2DirtyMap[key] = true
		c.l2DirtyKeys = append(c.l2DirtyKeys, key)
	}
}

// pullL2DirtyKeys 取出并清空 L2 待同步队列
func (c *Cache) pullL2DirtyKeys() []string {
	c.l2DirtyMu.Lock()
	defer c.l2DirtyMu.Unlock()

	if len(c.l2DirtyKeys) == 0 {
		return nil
	}

	keys := make([]string, len(c.l2DirtyKeys))
	copy(keys, c.l2DirtyKeys)
	c.l2DirtyKeys = c.l2DirtyKeys[:0]
	c.l2DirtyMap = make(map[string]bool)
	return keys
}

// flushLoop 刷新协程
func (c *Cache) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.doFlush()
		case <-c.flushCh:
			c.doFlush()
		case <-c.stopCh:
			// 停止前刷新剩余数据
			c.doFlush()
			return
		}
	}
}

// writeWorker 写入工作协程（用于处理 writeCh 中的任务）
func (c *Cache) writeWorker() {
	defer c.wg.Done()

	for {
		select {
		case task := <-c.writeCh:
			// 处理写入任务
			err := c.doFlush()
			if task.done != nil {
				task.done <- err
			}
		case <-c.stopCh:
			return
		}
	}
}

// cleanupLoop 清理协程
func (c *Cache) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup 清理过期数据
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, e := range c.data {
		if e.isExpired() && !e.dirty {
			delete(c.data, key)
		}
	}
}

// getExpireTime 获取过期时间
func (c *Cache) getExpireTime() time.Time {
	if c.config.DefaultExpiration > 0 {
		return time.Now().Add(c.config.DefaultExpiration)
	}
	return time.Time{}
}

// Stats 获取统计信息
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	cacheSize := len(c.data)
	c.mu.RUnlock()

	c.dirtyMu.Lock()
	dirtyCount := len(c.dirtyKeys)
	c.dirtyMu.Unlock()

	stats := Stats{
		CacheHits:      atomic.LoadInt64(&c.stats.CacheHits),
		CacheMisses:    atomic.LoadInt64(&c.stats.CacheMisses),
		CacheSize:      cacheSize,
		DirtyCount:     int64(dirtyCount),
		DBReads:        atomic.LoadInt64(&c.stats.DBReads),
		DBWrites:       atomic.LoadInt64(&c.stats.DBWrites),
		DBWriteErrors:  atomic.LoadInt64(&c.stats.DBWriteErrors),
		FlushCount:     atomic.LoadInt64(&c.stats.FlushCount),
		FlushTotalTime: c.stats.FlushTotalTime,
		LastFlushTime:  c.stats.LastFlushTime,
		IsFlushing:     atomic.LoadInt32(&c.flushing) == 1,
	}

	return stats
}

// Len 获取缓存大小
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Keys 获取所有key
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.data))
	for key := range c.data {
		keys = append(keys, key)
	}
	return keys
}

// DirtyCount 获取脏数据数量
func (c *Cache) DirtyCount() int {
	c.dirtyMu.Lock()
	defer c.dirtyMu.Unlock()
	return len(c.dirtyKeys)
}

// Close 关闭缓存
func (c *Cache) Close() error {
	close(c.stopCh)

	// 等待所有协程完成
	c.wg.Wait()

	// 最后一次刷新
	c.doFlush()

	// 关闭存储
	if c.store != nil {
		c.store.Close()
	}

	return nil
}

// Clear 清空缓存
func (c *Cache) Clear() {
	c.mu.Lock()
	c.data = make(map[string]*entry)
	c.mu.Unlock()

	c.dirtyMu.Lock()
	c.dirtyKeys = c.dirtyKeys[:0]
	c.dirtyMap = make(map[string]bool)
	c.dirtyMu.Unlock()
}
