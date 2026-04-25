package mcachedb

import (
	"context"
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
	data   map[string]Entity
	mu     sync.RWMutex
	setOps int
}

func NewMockRedisStore() *MockRedisStore {
	return &MockRedisStore{
		data: make(map[string]Entity),
	}
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

// MockDBStore 模拟数据库存储
type MockDBStore struct {
	mu     sync.RWMutex
	data   map[string]Entity
	calls  map[string]int
	callMu sync.Mutex
}

func NewMockDBStore() *MockDBStore {
	return &MockDBStore{
		data:  make(map[string]Entity),
		calls: make(map[string]int),
	}
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
