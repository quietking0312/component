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
		BaseEntity: *m.BaseEntity.Copy(),
		Name:       m.Name,
		Value:      m.Value,
	}
}

func (m *MockEntity) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MockEntity) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
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

func (m *MockRedisStore) Close() error { return nil }

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

// ─── 测试 ─────────────────────────────────────────────────────────────────────

func TestMultiCache_Basic(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "test", 100)

	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
	})
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

func TestMultiCache_SetAndGet(t *testing.T) {
	dbStore := NewMockDBStore()
	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	err = cache.Set(NewMockEntity("1", "alice", 100))
	assert.NoError(t, err)

	e, err := cache.Get("1")
	assert.NoError(t, err)
	assert.Equal(t, "alice", e.(*MockEntity).Name)

	time.Sleep(200 * time.Millisecond)
	assert.GreaterOrEqual(t, dbStore.GetCallCount("BatchInsert"), 1)
}

func TestMultiCache_WriteModeWriteL2(t *testing.T) {
	dbStore := NewMockDBStore()
	l2Store := NewMockRedisStore()

	cache, err := NewMultiCache(dbStore, l2Store, &MultiCacheConfig{
		L1MaxSize:     1000,
		WriteMode:     WriteModeWriteL2,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
	})
	assert.NoError(t, err)
	defer cache.Close()

	err = cache.Set(NewMockEntity("1", "alice", 100))
	assert.NoError(t, err)
	assert.Equal(t, 1, l2Store.GetSetOps())

	got, err := l2Store.Get(context.Background(), "1")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "alice", got.(*MockEntity).Name)

	// 已同步写 L2 的 key 不应被 syncToL2Loop 重复搬运
	cache.syncToL2()
	assert.Equal(t, 1, l2Store.GetSetOps())
}

func TestMultiCache_Delete(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "test", 100)

	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	cache.Get("1")

	err = cache.Delete("1")
	assert.NoError(t, err)

	// Delete 同步写 L3，无需等待 flush
	assert.Nil(t, dbStore.data["1"])

	got, err := cache.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestMultiCache_CacheAsideMode(t *testing.T) {
	dbStore := NewMockDBStore()
	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
		WriteMode:     WriteModeCacheAside,
	})
	assert.NoError(t, err)
	defer cache.Close()

	// 写入新数据
	err = cache.Set(NewMockEntity("1", "alice", 100))
	assert.NoError(t, err)
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

	// 更新：应走 Update
	err = cache.Set(NewMockEntity("1", "bob", 200))
	assert.NoError(t, err)
	assert.Equal(t, 1, dbStore.GetCallCount("Update"))
	assert.Equal(t, "bob", dbStore.data["1"].(*MockEntity).Name)

	// 删除
	err = cache.Delete("1")
	assert.NoError(t, err)
	assert.Equal(t, 1, dbStore.GetCallCount("Delete"))
	assert.Nil(t, dbStore.data["1"])

	l1Entity, err = cache.l1.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, l1Entity)
}

func TestMultiCache_BackfillNotDirty(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "alice", 100)

	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 100 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)

	// 从 L3 回填的数据不应触发 BatchInsert/BatchUpdate
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, dbStore.GetCallCount("BatchInsert"))
	assert.Equal(t, 0, dbStore.GetCallCount("BatchUpdate"))
}

func TestRedisStore_TypePreservation(t *testing.T) {
	prototype := NewMockEntity("", "", 0)
	store := &RedisStore{entityType: prototype}

	entity := NewMockEntity("1", "alice", 100)
	data, err := entity.Marshal()
	assert.NoError(t, err)

	got, err := store.unmarshalEntity(data)
	assert.NoError(t, err)
	assert.IsType(t, &MockEntity{}, got)
	assert.Equal(t, "alice", got.(*MockEntity).Name)
	assert.Equal(t, 100, got.(*MockEntity).Value)
}

func TestMultiCache_PendingDeleteNotBackfilled(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "alice", 100)

	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
	})
	assert.NoError(t, err)
	defer cache.Close()

	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)

	// Delete 同步删 L3，不依赖 flush
	err = cache.Delete("1")
	assert.NoError(t, err)
	assert.Nil(t, dbStore.data["1"])

	// 删除后读取应返回 nil
	entity, err = cache.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, entity)
}

func TestMultiCache_L2Recovery(t *testing.T) {
	dbStore := NewMockDBStore()
	l2Store := NewMockRedisStore()

	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{L1MaxSize: 1000, FlushInterval: 1 * time.Hour})
	assert.NoError(t, err)
	defer cache.Close()

	cache.l2 = l2Store
	cache.markL2Down()
	assert.True(t, cache.isL2Down())

	l2Store.SetPingErr(errors.New("redis down"))
	assert.False(t, cache.tryRecoverL2())
	assert.True(t, cache.isL2Down())
	assert.Equal(t, 1, l2Store.GetPingCalls())

	l2Store.SetPingErr(nil)
	assert.False(t, cache.tryRecoverL2())
	assert.True(t, cache.isL2Down())

	assert.True(t, cache.tryRecoverL2())
	assert.False(t, cache.isL2Down())

	assert.False(t, cache.tryRecoverL2())
	assert.False(t, cache.isL2Down())
}

func TestMultiCache_syncToL2Error(t *testing.T) {
	dbStore := NewMockDBStore()
	l2Store := NewMockRedisStore()

	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
	})
	assert.NoError(t, err)
	defer cache.Close()

	cache.l2 = l2Store

	// 通过 MultiCache.Set 写入，这样才会标记 L2 dirty
	entity := NewMockEntity("1", "alice", 100)
	err = cache.Set(entity)
	assert.NoError(t, err)

	l2Store.SetMSetErr(errors.New("redis error"))
	assert.False(t, cache.isL2Down())

	cache.syncToL2()
	assert.True(t, cache.isL2Down())
}

func TestMultiCache_CloseIdempotent(t *testing.T) {
	dbStore := NewMockDBStore()
	cache, err := NewMultiCache(dbStore, nil, nil)
	assert.NoError(t, err)

	assert.NotPanics(t, func() {
		assert.NoError(t, cache.Close())
		assert.NoError(t, cache.Close())
	})
}

func TestMultiCache_Concurrent(t *testing.T) {
	dbStore := NewMockDBStore()
	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     10000,
		SyncInterval:  50 * time.Millisecond,
		FlushInterval: 50 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache.Set(NewMockEntity(string(rune('0'+n%10)), "user", n))
		}(i)
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)
	assert.GreaterOrEqual(t, len(dbStore.data), 10)
}

func TestDistTx_RollbackCacheAside(t *testing.T) {
	dbStore := NewMockDBStore()
	dbStore.data["1"] = NewMockEntity("1", "original", 100)

	cache, err := NewMultiCache(dbStore, nil, &MultiCacheConfig{
		L1MaxSize:     1000,
		WriteMode:     WriteModeCacheAside,
		SyncInterval:  1 * time.Hour,
		FlushInterval: 1 * time.Hour,
	})
	assert.NoError(t, err)
	defer cache.Close()

	// 1. 注册新增操作后回滚：DB 不应出现该 key
	tx := NewDistTx()
	assert.NoError(t, tx.AddSet(cache, NewMockEntity("2", "new", 200)))
	assert.NoError(t, tx.Rollback())
	assert.Nil(t, dbStore.data["2"])
	assert.Equal(t, 0, dbStore.GetCallCount("Insert"))

	// 2. 注册更新操作后回滚：DB 保持原值
	tx2 := NewDistTx()
	assert.NoError(t, tx2.AddSet(cache, NewMockEntity("1", "updated", 999)))
	assert.NoError(t, tx2.Rollback())
	assert.Equal(t, "original", dbStore.data["1"].(*MockEntity).Name)
	assert.Equal(t, 0, dbStore.GetCallCount("Update"))

	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.Equal(t, "original", entity.(*MockEntity).Name)

	// 3. 注册删除操作后回滚：DB 保持原值
	tx3 := NewDistTx()
	assert.NoError(t, tx3.AddDelete(cache, "1"))
	assert.NoError(t, tx3.Rollback())
	assert.NotNil(t, dbStore.data["1"])
	assert.Equal(t, 0, dbStore.GetCallCount("Delete"))

	entity, err = cache.Get("1")
	assert.NoError(t, err)
	assert.Equal(t, "original", entity.(*MockEntity).Name)
}
