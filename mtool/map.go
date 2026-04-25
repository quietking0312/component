package mtool

import (
	"sync"
)

// 有序的 map
// 并发安全
// 因为使用了读写锁，在读取时候不如 sync.map
// 使用了泛型，类型更加安全，内存效率 比sync.map 好

type OrderedMap[K comparable, V any] struct {
	data map[K]V
	keys []K
	mu   sync.RWMutex
}

type OrderedMapSnapshot[K comparable, V any] struct {
	Data map[K]V
	Keys []K
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		data: make(map[K]V),
		keys: []K{},
	}
}

func (m *OrderedMap[K, V]) Snapshot() *OrderedMapSnapshot[K, V] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[K]V, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	keys := make([]K, len(m.keys))
	copy(keys, m.keys)
	return &OrderedMapSnapshot[K, V]{
		Data: data,
		Keys: keys,
	}
}

func (m *OrderedMap[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make([]V, 0, len(m.data))
	keysCopy := make([]K, len(m.keys))
	copy(keysCopy, m.keys)
	for _, k := range keysCopy {
		v, ok := m.data[k]
		if ok {
			data = append(data, v)
		}
	}
	return data
}

func (m *OrderedMap[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.data)
	m.keys = make([]K, 0)
}

func (m *OrderedMap[K, V]) Update(fn func(dataCopy map[K]V, keysCopy []K) (map[K]V, []K)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dataCopy := make(map[K]V, len(m.data))
	for k, v := range m.data {
		dataCopy[k] = v
	}
	keysCopy := make([]K, len(m.keys))
	copy(keysCopy, m.keys)
	newData, newKeys := fn(dataCopy, keysCopy)
	m.data = newData
	m.keys = newKeys
}

func (m *OrderedMap[K, V]) Set(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[k]; !exists {
		m.keys = append(m.keys, k)
	}
	m.data[k] = v
}

func (m *OrderedMap[K, V]) Get(k K) (v V, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok = m.data[k]
	return v, ok
}

func (m *OrderedMap[K, V]) Delete(k K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[k]; !ok {
		return
	}
	var newKeys []K
	for _, key := range m.keys {
		if key != k {
			newKeys = append(newKeys, key)
		}
	}
	m.keys = newKeys
	delete(m.data, k)
}

func (m *OrderedMap[K, V]) Deletes(k []K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var newKeys []K
	deleteSet := make(map[K]bool, len(k))
	for _, a := range k {
		deleteSet[a] = true
	}
	for _, key := range m.keys {
		if !deleteSet[key] {
			newKeys = append(newKeys, key)
		} else {
			delete(m.data, key)
		}
	}
	m.keys = newKeys
}

func (m *OrderedMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, len(m.keys))
	copy(keys, m.keys)
	return keys
}

func (m *OrderedMap[K, V]) Length() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.keys)
}
