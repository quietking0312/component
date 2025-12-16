package mtool

import (
	"sync"
)

type OrderedMap[K comparable, V any] struct {
	data map[K]V
	keys []K
	mu   sync.RWMutex
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		data: make(map[K]V),
		keys: []K{},
	}
}

func (m *OrderedMap[K, V]) Range(fn func(k K, v V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, k := range m.keys {
		v, _ := m.data[k]
		if !fn(k, v) {
			break
		}
	}
}

func (m *OrderedMap[K, V]) Set(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var newKeys []K
	for _, key := range m.keys {
		if key != k {
			newKeys = append(newKeys, key)
		}
	}
	m.keys = newKeys
	m.keys = append(m.keys, k)
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
	var newKeys []K
	for _, key := range m.keys {
		if key != k {
			newKeys = append(newKeys, key)
		}
	}
	m.keys = newKeys
	delete(m.data, k)
}

func (m *OrderedMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, len(m.keys))
	copy(keys, m.keys)
	return keys
}

func (m *OrderedMap[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]V, len(m.keys))
	for i, k := range m.keys {
		values[i] = m.data[k]
	}
	return values
}

func (m *OrderedMap[K, V]) Length() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.keys)
}

func (m *OrderedMap[K, V]) Iter() <-chan struct {
	Key   K
	Value V
} {
	ch := make(chan struct {
		Key   K
		Value V
	})
	go func() {
		m.mu.RLock()
		defer m.mu.RUnlock()
		defer close(ch)
		for _, k := range m.keys {
			ch <- struct {
				Key   K
				Value V
			}{Key: k, Value: m.data[k]}
		}
	}()
	return ch
}
