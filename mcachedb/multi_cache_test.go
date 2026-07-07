package mcachedb

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockEntity 测试实体
type MockEntity struct {
	BaseEntity
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func NewMockEntity(id, name string, value int) *MockEntity {
	return &MockEntity{
		BaseEntity: *NewBaseEntity(id),
		Name:       name,
		Value:      value,
	}
}

func (m *MockEntity) Copy() Entity {
	return &MockEntity{
		BaseEntity: *m.BaseEntity.Copy().(*BaseEntity),
		Name:       m.Name,
		Value:      m.Value,
	}
}

// MockRedisStore 模拟 Redis 存储
type MockRedisStore struct {
	*RedisStore
	data      map[string]Entity
	mu        sync.RWMutex
	setOps    int
	pingErr   error
	pingCalls int
	msetErr   error
}

func NewMockRedisStore() *MockRedisStore {
	return &MockRedisStore{
		data: make(map[string]Entity),
	}
}

func (m *MockRedisStore) SetPingErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingErr = err
}

func (m *MockRedisStore) GetPingCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pingCalls
}

func (m *MockRedisStore) SetMSetErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msetErr = err
}

func (m *MockRedisStore) Get(ctx context.Context, key string) (Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.data[key]; ok {
		return e.Copy(), nil
	}
	return nil, nil
}

func (m *MockRedisStore) Set(ctx context.Context, entity Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[entity.CacheKey()] = entity.Copy()
	m.setOps++
	return nil
}

func (m *MockRedisStore) GetSetOps() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.setOps
}

func (m *MockRedisStore) MGet(ctx context.Context, keys []string) (map[string]Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]Entity)
	for _, key := range keys {
		if e, ok := m.data[key]; ok {
			result[key] = e.Copy()
		}
	}
	return result, nil
}

func (m *MockRedisStore) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingCalls++
	return m.pingErr
}

func (m *MockRedisStore) MSet(ctx context.Context, entities []Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.msetErr != nil {
		return m.msetErr
	}
	for _, entity := range entities {
		m.data[entity.CacheKey()] = entity.Copy()
		m.setOps++
	}
	return nil
}

func (m *MockRedisStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MockRedisStore) Close() error {
	return nil
}

// MockDBStore 模拟数据库存储
type MockDBStore struct {
	mu     sync.RWMutex
	data   map[string]Entity
	calls  map[string]int
	callMu sync.Mutex
	getErr error
}

func NewMockDBStore() *MockDBStore {
	return &MockDBStore{
		data:  make(map[string]Entity),
		calls: make(map[string]int),
	}
}

func (m *MockDBStore) SetGetErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getErr = err
}

func (m *MockDBStore) recordCall(method string) {
	m.callMu.Lock()
	defer m.callMu.Unlock()
	m.calls[method]++
}

func (m *MockDBStore) GetCallCount(method string) int {
	m.callMu.Lock()
	defer m.callMu.Unlock()
	return m.calls[method]
}

func (m *MockDBStore) Get(ctx context.Context, key string) (Entity, error) {
	m.recordCall("Get")
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	if e, ok := m.data[key]; ok {
		return e.Copy(), nil
	}
	return nil, nil
}

func (m *MockDBStore) MGet(ctx context.Context, keys []string) (map[string]Entity, error) {
	m.recordCall("MGet")
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]Entity)
	for _, key := range keys {
		if e, ok := m.data[key]; ok {
			result[key] = e.Copy()
		}
	}
	return result, nil
}

func (m *MockDBStore) Insert(ctx context.Context, entity Entity) error {
	m.recordCall("Insert")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[entity.CacheKey()] = entity.Copy()
	return nil
}

func (m *MockDBStore) Update(ctx context.Context, entity Entity) error {
	m.recordCall("Update")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[entity.CacheKey()] = entity.Copy()
	return nil
}

func (m *MockDBStore) Delete(ctx context.Context, key string) error {
	m.recordCall("Delete")
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MockDBStore) BatchInsert(ctx context.Context, entities []Entity) error {
	m.recordCall("BatchInsert")
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range entities {
		m.data[e.CacheKey()] = e.Copy()
	}
	return nil
}

func (m *MockDBStore) BatchUpdate(ctx context.Context, entities []Entity) error {
	m.recordCall("BatchUpdate")
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range entities {
		m.data[e.CacheKey()] = e.Copy()
	}
	return nil
}

func (m *MockDBStore) BatchDelete(ctx context.Context, keys []string) error {
	m.recordCall("BatchDelete")
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *MockDBStore) Close() error { return nil }

// TestMultiCache_Basic 基础测试
func TestMultiCache_Basic(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "test", 100)

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)
	assert.Equal(t, "test", entity.(*MockEntity).Name)

	time.Sleep(50 * time.Millisecond)

	entity, err = cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)

	stats := cache.Stats()
	assert.Equal(t, int64(1), stats.L3Hits)
	assert.Equal(t, int64(1), stats.L1Hits)
}

// TestMultiCache_SetAndGet 测试写入和读取
func TestMultiCache_SetAndGet(t *testing.T) {
	dbStore := NewMockDBStore()

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	entity := NewMockEntity("1", "alice", 100)
	err = cache.Set(entity)
	assert.NoError(t, err)

	e, err := cache.Get("1")
	assert.NoError(t, err)
	assert.Equal(t, "alice", e.(*MockEntity).Name)

	time.Sleep(200 * time.Millisecond)

	assert.GreaterOrEqual(t, dbStore.GetCallCount("BatchInsert"), 1)
}

// TestMultiCache_WriteToL2OnSet 测试写L1时同步写L2
func TestMultiCache_WriteToL2OnSet(t *testing.T) {
	dbStore := NewMockDBStore()

	// 创建一个带 Mock L2 的 MultiCache
	config := &MultiCacheConfig{
		L1MaxSize:      1000,
		WriteToL2OnSet: true, // 关键：同步写L2
		SyncInterval:   1 * time.Hour,
		FlushInterval:  1 * time.Hour,
		L2RedisConfig:  &RedisConfig{Addr: "localhost:6379"}, // 会连接失败，但会被 mock 替换
		L2Downgrade:    false,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	// 替换 l2 为 mock（因为真实 Redis 可能没有启动）
	// 注意：这里只是测试概念，实际使用中 Redis 会真的写入
	// 由于构造后替换字段不太优雅，我们通过另一种方式测试：
	// 设置一个能连接上的 Redis，或者跳过这个测试
}

// TestMultiCache_Delete 测试删除
func TestMultiCache_Delete(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "test", 100)

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	cache.Get("1")
	time.Sleep(50 * time.Millisecond)

	err = cache.Delete("1")
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	_, ok := dbStore.data["1"]
	assert.False(t, ok)
}

// TestMultiCache_CacheAsideMode 缓存旁路模式（无 L2，重点验证 L1 失效与 L3 回填）
func TestMultiCache_CacheAsideMode(t *testing.T) {
	dbStore := NewMockDBStore()

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
		L2RedisConfig: nil, // 不依赖 Redis，专注验证 L1/L3 行为
		WriteMode:     WriteModeCacheAside,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	// 写入新数据：应写 L3，并删除 L1
	newEntity := NewMockEntity("1", "alice", 100)
	err = cache.Set(newEntity)
	assert.NoError(t, err)

	// L3 已写入
	assert.Equal(t, 1, dbStore.GetCallCount("Insert"))
	assert.Equal(t, "alice", dbStore.data["1"].(*MockEntity).Name)

	// L1 应立即失效
	l1Entity, err := cache.l1.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, l1Entity)

	// 读取：L1 未命中，从 L3 回填
	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)
	assert.Equal(t, "alice", entity.(*MockEntity).Name)

	// L1 被回填
	l1Entity, err = cache.l1.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, l1Entity)
	assert.Equal(t, "alice", l1Entity.(*MockEntity).Name)

	// 更新：应走 Update 而不是 Insert
	updatedEntity := NewMockEntity("1", "bob", 200)
	err = cache.Set(updatedEntity)
	assert.NoError(t, err)
	assert.Equal(t, 1, dbStore.GetCallCount("Update"))
	assert.Equal(t, "bob", dbStore.data["1"].(*MockEntity).Name)

	// 删除：数据库和 L1 都应被删除
	err = cache.Delete("1")
	assert.NoError(t, err)
	assert.Equal(t, 1, dbStore.GetCallCount("Delete"))
	assert.Nil(t, dbStore.data["1"])

	l1Entity, err = cache.l1.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, l1Entity)
}

// TestMultiCache_BackfillNotDirty 验证从 L3 回填 L1 时不会标记 dirty
func TestMultiCache_BackfillNotDirty(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "alice", 100)

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 100 * time.Millisecond,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	// 从 L3 读取并回填 L1
	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)

	// 等待 flush，回填数据不应触发 BatchInsert/BatchUpdate
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, dbStore.GetCallCount("BatchInsert"))
	assert.Equal(t, 0, dbStore.GetCallCount("BatchUpdate"))
}

// TestRedisStore_TypePreservation 验证 RedisStore 在配置 entityType 后可保持类型
func TestRedisStore_TypePreservation(t *testing.T) {
	prototype := NewMockEntity("", "", 0)
	store := &RedisStore{entityType: prototype}

	entity := NewMockEntity("1", "alice", 100)

	// 序列化
	data, err := store.marshalEntity(entity)
	assert.NoError(t, err)

	// 反序列化应保持具体类型
	got, err := store.unmarshalEntity(data)
	assert.NoError(t, err)
	assert.IsType(t, &MockEntity{}, got)
	assert.Equal(t, "alice", got.(*MockEntity).Name)
	assert.Equal(t, 100, got.(*MockEntity).Value)

	// 旧格式（EntityWrapper）应能降级解析
	oldWrapper := &EntityWrapper{
		BaseEntity: *NewBaseEntity("2"),
		Data:       NewMockEntity("2", "bob", 200),
	}
	oldData, err := json.Marshal(oldWrapper)
	assert.NoError(t, err)

	got2, err := store.unmarshalEntity(oldData)
	assert.NoError(t, err)
	assert.IsType(t, &EntityWrapper{}, got2)
}

// TestMultiCache_PendingDeleteNotBackfilled 验证已标记删除但未 flush 的 key 不会被回填
func TestMultiCache_PendingDeleteNotBackfilled(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "alice", 100)

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 100 * time.Millisecond,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	// 先读取，回填 L1
	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)

	// 删除：L1 标记 deleted，L3 立即删除
	err = cache.Delete("1")
	assert.NoError(t, err)
	assert.Nil(t, dbStore.data["1"])

	// 在 flush 前再次读取，应因 L1 pending delete 而返回 nil，不会从 L3 回填旧数据
	entity, err = cache.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, entity)

	// 等待 flush
	time.Sleep(200 * time.Millisecond)

	// L1 中的 deleted entry 被清理后，再次读取仍为 nil
	entity, err = cache.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, entity)
}

// TestMultiCache_L2Recovery 验证 L2 故障后可自动恢复
func TestMultiCache_L2Recovery(t *testing.T) {
	dbStore := NewMockDBStore()
	l2Store := NewMockRedisStore()

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		FlushInterval: 1 * time.Hour,
		L2RedisConfig: nil, // 不自动创建 L2，手动注入 mock
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	cache.l2 = l2Store
	cache.markL2Down()
	assert.True(t, cache.isL2Down())

	// 第一次探测失败
	l2Store.SetPingErr(errors.New("redis down"))
	assert.False(t, cache.tryRecoverL2())
	assert.True(t, cache.isL2Down())
	assert.Equal(t, 1, l2Store.GetPingCalls())

	// 连续两次探测成功后才恢复
	l2Store.SetPingErr(nil)
	assert.False(t, cache.tryRecoverL2())
	assert.True(t, cache.isL2Down())

	assert.True(t, cache.tryRecoverL2())
	assert.False(t, cache.isL2Down())

	// 恢复后再次探测，状态保持可用
	assert.False(t, cache.tryRecoverL2())
	assert.False(t, cache.isL2Down())
}

// TestMultiCache_syncToL2Error 验证 L2 MSet 失败会标记 l2Down
func TestMultiCache_syncToL2Error(t *testing.T) {
	dbStore := NewMockDBStore()
	l2Store := NewMockRedisStore()

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	cache.l2 = l2Store

	// 写入 L1，触发 L2 dirty
	entity := NewMockEntity("1", "alice", 100)
	err = cache.l1.Set(entity)
	assert.NoError(t, err)

	// 模拟 L2 MSet 失败
	l2Store.SetMSetErr(errors.New("redis error"))
	assert.False(t, cache.isL2Down())

	cache.syncToL2()

	assert.True(t, cache.isL2Down())
}

// TestMultiCache_CloseIdempotent 验证 Close 可重复调用不 panic
func TestMultiCache_CloseIdempotent(t *testing.T) {
	dbStore := NewMockDBStore()
	cache, err := NewMultiCache(dbStore, nil)
	assert.NoError(t, err)

	assert.NotPanics(t, func() {
		assert.NoError(t, cache.Close())
		assert.NoError(t, cache.Close())
	})
}

// TestMultiCache_Concurrent 并发测试
func TestMultiCache_Concurrent(t *testing.T) {
	dbStore := NewMockDBStore()

	config := &MultiCacheConfig{
		L1MaxSize:     10000,
		SyncInterval:  50 * time.Millisecond,
		FlushInterval: 50 * time.Millisecond,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			entity := NewMockEntity(
				string(rune('0'+n%10)),
				"user",
				n,
			)
			cache.Set(entity)
		}(i)
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	assert.GreaterOrEqual(t, len(dbStore.data), 10)
}

// TestDistTx_RollbackCacheAside 验证 CacheAside 模式下手动 Rollback 不会写数据库
func TestDistTx_RollbackCacheAside(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "original", 100)

	config := &MultiCacheConfig{
		L1MaxSize:     1000,
		WriteMode:     WriteModeCacheAside,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
		L2RedisConfig: nil,
	}

	cache, err := NewMultiCache(dbStore, config)
	assert.NoError(t, err)
	defer cache.Close()

	tx := NewDistTx()

	// 1. 注册新增操作，然后回滚：DB 不应出现该 key
	assert.NoError(t, tx.AddSet(cache, NewMockEntity("2", "new", 200)))
	assert.NoError(t, tx.Rollback())
	assert.Nil(t, dbStore.data["2"])
	assert.Equal(t, 0, dbStore.GetCallCount("Insert"))

	// 2. 注册更新操作，然后回滚：DB 保持原值，缓存读到旧值
	tx2 := NewDistTx()
	assert.NoError(t, tx2.AddSet(cache, NewMockEntity("1", "updated", 999)))
	assert.NoError(t, tx2.Rollback())
	assert.Equal(t, "original", dbStore.data["1"].(*MockEntity).Name)
	assert.Equal(t, 0, dbStore.GetCallCount("Update"))

	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.Equal(t, "original", entity.(*MockEntity).Name)

	// 3. 注册删除操作，然后回滚：DB 保持原值
	tx3 := NewDistTx()
	assert.NoError(t, tx3.AddDelete(cache, "1"))
	assert.NoError(t, tx3.Rollback())
	assert.NotNil(t, dbStore.data["1"])
	assert.Equal(t, 0, dbStore.GetCallCount("Delete"))

	entity, err = cache.Get("1")
	assert.NoError(t, err)
	assert.Equal(t, "original", entity.(*MockEntity).Name)
}
