package mtypes

import (
	"sync"
)

type Map struct {
	data map[any]any
	keys []any
	mu   sync.RWMutex
}

func NewMap(n int) *Map {
	return &Map{
		data: make(map[any]any, n),
		keys: []any{},
	}
}

func (m *Map) Range(fn func(k, v any) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range m.keys {
		v, _ := m.data[k]
		if !fn(k, v) {
			break
		}
	}
}

func (m *Map) Set(k, v any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var newKeys []any
	for _, key := range m.keys {
		if key != k {
			newKeys = append(newKeys, key)
		}
	}
	m.keys = newKeys
	m.keys = append(m.keys, k)
	m.data[k] = v
}

func (m *Map) Get(k any) (v any, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok = m.data[k]
	return v, ok
}

func (m *Map) Delete(k any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var newKeys []any
	for _, key := range m.keys {
		if key != k {
			newKeys = append(newKeys, key)
		}
	}
	m.keys = newKeys
	delete(m.data, k)
}

func (m *Map) Keys() []any {
	return m.keys
}
