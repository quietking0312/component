package mcachedb

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Cache 纯内存 L1 缓存
//
// 该层不感知任何持久化逻辑：Set/Delete 只操作内存 map，dirty/pending-delete
// 等状态由 MultiCache 在写入时注入（通过 setEntry 等包内方法）。
type Cache struct {
	config    *Config
	mu        sync.RWMutex
	data      map[string]*entry
	hits      int64
	misses    int64
	stopCh    chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// New 创建 L1 内存缓存
func New(opts ...Option) (*Cache, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	c := &Cache{
		config: config,
		data:   make(map[string]*entry),
		stopCh: make(chan struct{}),
	}

	if config.CleanupInterval > 0 {
		c.wg.Add(1)
		go c.cleanupLoop()
	}

	return c, nil
}

// Get 获取实体。expired 条目返回 nil。
func (c *Cache) Get(key string) (Entity, error) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()

	if !ok || e.isExpired() {
		atomic.AddInt64(&c.misses, 1)
		return nil, nil
	}
	atomic.AddInt64(&c.hits, 1)
	return e.entity.Copy(), nil
}

// MGet 批量获取，跳过 expired 条目
func (c *Cache) MGet(keys []string) (map[string]Entity, error) {
	result := make(map[string]Entity, len(keys))
	c.mu.RLock()
	for _, key := range keys {
		if e, ok := c.data[key]; ok && !e.isExpired() {
			result[key] = e.entity.Copy()
			atomic.AddInt64(&c.hits, 1)
		} else {
			atomic.AddInt64(&c.misses, 1)
		}
	}
	c.mu.RUnlock()
	return result, nil
}

// Set 向 L1 写入实体（clean 写入，不标记 dirty）。
// 返回 true 表示 key 之前不在缓存中（首次写入）。
// 供独立使用 Cache 时调用；MultiCache 内部使用 setEntry。
func (c *Cache) Set(entity Entity) bool {
	if entity == nil {
		return false
	}
	key := entity.CacheKey()
	c.mu.Lock()
	_, existed := c.data[key]
	if c.config.MaxCacheSize > 0 && !existed && len(c.data) >= c.config.MaxCacheSize {
		c.mu.Unlock()
		return false
	}
	c.data[key] = &entry{
		entity:    entity.Copy(),
		createdAt: time.Now(),
		expireAt:  c.getExpireTime(),
	}
	c.mu.Unlock()
	return !existed
}

// setEntry 写入实体并附带 dirty 追踪元数据，由 MultiCache 调用。
// 返回 true 表示写入成功；返回 false 表示 L1 已达容量上限，写入被丢弃。
// 注意：若已有条目标记为 isNew（首次写入且尚未 flush），新写入也保留 isNew=true。
func (c *Cache) setEntry(entity Entity, seq uint64) bool {
	key := entity.CacheKey()
	c.mu.Lock()
	defer c.mu.Unlock()

	old, existed := c.data[key]
	// 若之前已是 dirty+isNew，保留 isNew 以确保刷盘时走 Insert 路径
	preserveIsNew := existed && old.isNew

	if c.config.MaxCacheSize > 0 && !existed && len(c.data) >= c.config.MaxCacheSize {
		return false
	}

	c.data[key] = &entry{
		entity:    entity.Copy(),
		createdAt: time.Now(),
		expireAt:  c.getExpireTime(),
		dirty:     true,
		dirtySeq:  seq,
		isNew:     !existed || preserveIsNew,
	}
	return true
}

// Load 将外部（L2/L3）查询结果回填到 L1，不标记 dirty。
// 以下情况跳过：L1 已有 dirty 或 pending-delete 条目、L1 版本号 >= 传入版本。
func (c *Cache) Load(entity Entity) {
	if entity == nil {
		return
	}
	key := entity.CacheKey()
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.data[key]; ok {
		if e.dirty {
			return
		}
		if e.entity != nil && e.entity.Version() >= entity.Version() {
			return
		}
	}

	c.data[key] = &entry{
		entity:    entity.Copy(),
		createdAt: time.Now(),
		expireAt:  c.getExpireTime(),
	}
}

// Delete 从 L1 中直接移除 key（独立使用时调用）
func (c *Cache) Delete(key string) {
	c.Remove(key)
}

// Remove 直接移除 key，不标记 dirty，也不触发持久化。
func (c *Cache) Remove(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

// IsDirty 返回 key 是否有未 flush 的脏数据
func (c *Cache) IsDirty(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.data[key]
	return ok && e.dirty
}

// getDirtyEntities 收集指定 keys 中需要 flush 的实体，区分 Insert/Update。
// 由 MultiCache.doFlush 调用。
func (c *Cache) getDirtyEntities(keys []string) (toInsert, toUpdate []Entity) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, key := range keys {
		e, ok := c.data[key]
		if !ok || !e.dirty {
			continue
		}
		if e.isNew {
			toInsert = append(toInsert, e.entity)
		} else {
			toUpdate = append(toUpdate, e.entity)
		}
	}
	return
}

// clearDirtyEntries 在 flush 成功后清除 dirty 标记。
// 只清除 dirtySeq <= snapshotSeq 的条目，避免清除 flush 期间新产生的 dirty。
// 由 MultiCache.doFlush 调用。
func (c *Cache) clearDirtyEntries(keys []string, snapshotSeq uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		if e, ok := c.data[key]; ok && e.dirtySeq <= snapshotSeq {
			e.dirty = false
			e.isNew = false
		}
	}
}

// Len 返回当前缓存条目数
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Keys 返回所有 key
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// Clear 清空 L1 缓存
func (c *Cache) Clear() {
	c.mu.Lock()
	c.data = make(map[string]*entry)
	c.mu.Unlock()
}

// Stats 返回简要命中统计
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	size := len(c.data)
	c.mu.RUnlock()
	return Stats{
		CacheHits:   atomic.LoadInt64(&c.hits),
		CacheMisses: atomic.LoadInt64(&c.misses),
		CacheSize:   size,
	}
}

// Close 停止后台清理协程
func (c *Cache) Close() error {
	c.closeOnce.Do(func() {
		close(c.stopCh)
		c.wg.Wait()
	})
	return nil
}

func (c *Cache) getExpireTime() time.Time {
	if c.config.DefaultExpiration > 0 {
		return time.Now().Add(c.config.DefaultExpiration)
	}
	return time.Time{}
}

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

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, e := range c.data {
		if e.isExpired() && !e.dirty {
			delete(c.data, key)
		}
	}
}

// validateEntity 检查实体合法性
func validateEntity(entity Entity) error {
	if entity == nil {
		return fmt.Errorf("entity cannot be nil")
	}
	if entity.CacheKey() == "" {
		return fmt.Errorf("entity key cannot be empty")
	}
	return nil
}
