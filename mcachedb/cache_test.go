package mcachedb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockStore 内存存储实现
type MockStore struct {
	mu     sync.RWMutex
	data   map[string]Entity
	calls  map[string]int
	getErr error
}

func NewMockStore() *MockStore {
	return &MockStore{
		data:  make(map[string]Entity),
		calls: make(map[string]int),
	}
}

func (m *MockStore) SetGetErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getErr = err
}

func (m *MockStore) Get(ctx context.Context, key string) (Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.calls["Get"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
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

func TestCache_Load(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithFlushInterval(100*time.Millisecond))
	defer cache.Close()

	user := NewUser("1", "alice", "alice@test.com", 25)
	cache.Load(user)

	// Load 的数据可以命中
	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity)
	assert.Equal(t, "alice", entity.(*User).Username)

	// Load 不标记 dirty，后台 flush 不应写入数据库
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, store.GetCallCount("BatchInsert"))
	assert.Equal(t, 0, store.GetCallCount("BatchUpdate"))
}

func TestCache_IsDeleted(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithFlushInterval(1*time.Hour))
	defer cache.Close()

	user := NewUser("1", "alice", "alice@test.com", 25)
	cache.Set(user)

	assert.False(t, cache.IsDeleted("1"))

	cache.Delete("1")
	assert.True(t, cache.IsDeleted("1"))
	assert.False(t, cache.IsDeleted("2"))
}

func TestCache_FlushConcurrentSet(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithFlushInterval(50*time.Millisecond), WithBatchSize(1000))
	defer cache.Close()

	// 持续写入，与后台 flush 并发
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			user := NewUser(fmt.Sprintf("%d", n), "user", fmt.Sprintf("user%d@test.com", n), n)
			cache.Set(user)
		}(i)
	}
	wg.Wait()

	// 等待 flush 完成
	time.Sleep(300 * time.Millisecond)
	cache.Flush()

	// 所有写入都应落盘
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("%d", i)
		assert.NotNil(t, store.data[key], "key %s should be persisted", key)
	}
}

func TestCache_CacheAsideGetError(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithWriteMode(WriteModeCacheAside))
	defer cache.Close()

	// 模拟 DB Get 失败
	store.SetGetErr(errors.New("db error"))

	user := NewUser("1", "alice", "alice@test.com", 25)
	err := cache.Set(user)
	assert.Error(t, err)
	assert.Equal(t, 0, store.GetCallCount("Insert"))
}

func TestCache_CloseIdempotent(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store)

	assert.NotPanics(t, func() {
		assert.NoError(t, cache.Close())
		assert.NoError(t, cache.Close())
	})
}

func TestCache_CacheAsideWrite(t *testing.T) {
	store := NewMockStore()
	cache, _ := New(store, WithWriteMode(WriteModeCacheAside))
	defer cache.Close()

	user := NewUser("1", "alice", "alice@test.com", 25)

	// 写入：先写数据库，然后删除本地缓存
	err := cache.Set(user)
	assert.NoError(t, err)
	assert.Equal(t, 1, store.GetCallCount("Insert"))

	// 缓存中应立即失效
	entity, err := cache.Get("1")
	assert.NoError(t, err)
	assert.NotNil(t, entity) // Get 会触发从 DB 回填
	assert.Equal(t, "alice", entity.(*User).Username)

	// 数据库中存在
	assert.NotNil(t, store.data["1"])

	// 更新：应走 Update 而不是 Insert
	user2 := NewUser("1", "bob", "bob@test.com", 30)
	err = cache.Set(user2)
	assert.NoError(t, err)
	assert.Equal(t, 1, store.GetCallCount("Update"))

	// 删除：先删数据库，再删缓存
	err = cache.Delete("1")
	assert.NoError(t, err)
	assert.Equal(t, 1, store.GetCallCount("Delete"))
	assert.Nil(t, store.data["1"])

	entity, err = cache.Get("1")
	assert.NoError(t, err)
	assert.Nil(t, entity)
}
