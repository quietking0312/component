package mcachedb

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockStore 内存存储实现
type MockStore struct {
	mu    sync.RWMutex
	data  map[string]Entity
	calls map[string]int
}

func NewMockStore() *MockStore {
	return &MockStore{
		data:  make(map[string]Entity),
		calls: make(map[string]int),
	}
}

func (m *MockStore) Get(ctx context.Context, key string) (Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.calls["Get"]++
	if e, ok := m.data[key]; ok {
		return e.Copy(), nil
	}
	return nil, nil
}

func (m *MockStore) MGet(ctx context.Context, keys []string) (map[string]Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.calls["MGet"]++
	result := make(map[string]Entity)
	for _, key := range keys {
		if e, ok := m.data[key]; ok {
			result[key] = e.Copy()
		}
	}
	return result, nil
}

func (m *MockStore) Insert(ctx context.Context, entity Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["Insert"]++
	m.data[entity.CacheKey()] = entity.Copy()
	return nil
}

func (m *MockStore) Update(ctx context.Context, entity Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["Update"]++
	m.data[entity.CacheKey()] = entity.Copy()
	return nil
}

func (m *MockStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["Delete"]++
	delete(m.data, key)
	return nil
}

func (m *MockStore) BatchInsert(ctx context.Context, entities []Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["BatchInsert"]++
	for _, e := range entities {
		m.data[e.CacheKey()] = e.Copy()
	}
	return nil
}

func (m *MockStore) BatchUpdate(ctx context.Context, entities []Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["BatchUpdate"]++
	for _, e := range entities {
		m.data[e.CacheKey()] = e.Copy()
	}
	return nil
}

func (m *MockStore) BatchDelete(ctx context.Context, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["BatchDelete"]++
	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *MockStore) Close() error { return nil }

func (m *MockStore) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.calls[method]
}

func TestCache_AsyncWrite(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithFlushInterval(100*time.Millisecond), WithBatchSize(10))
	defer cache.Close()

	user := NewUser("1", "alice", "alice@test.com", 25)
	cache.Set(user)

	// 内存中立即有
	entity, _ := cache.Get("1")
	assert.NotNil(t, entity)

	// 数据库还没有
	assert.Equal(t, 0, store.GetCallCount("BatchInsert"))

	// 等待刷新
	time.Sleep(200 * time.Millisecond)
	assert.GreaterOrEqual(t, store.GetCallCount("BatchInsert"), 1)
}

func TestCache_CacheHit(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithFlushInterval(1*time.Second))
	defer cache.Close()

	// 先放入存储，模拟数据库已有数据
	user := NewUser("1", "alice", "alice@test.com", 25)
	store.data["1"] = user

	// 第一次读取，应该未命中，从数据库回填
	cache.Get("1")

	// 后续多次读取，应该命中缓存
	for i := 0; i < 9; i++ {
		cache.Get("1")
	}

	stats := cache.Stats()
	assert.Equal(t, int64(9), stats.CacheHits)
	assert.Equal(t, int64(1), stats.CacheMisses)
}

func TestCache_Delete(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithFlushInterval(100*time.Millisecond))
	defer cache.Close()

	user := NewUser("1", "alice", "alice@test.com", 25)
	cache.Set(user)
	cache.Delete("1")

	entity, _ := cache.Get("1")
	assert.Nil(t, entity)
}
