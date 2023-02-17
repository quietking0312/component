package mtypes

import (
	"sync"
)

type Map struct {
	Data map[any]any
	Keys []any
	sync.RWMutex
}

func NewMap(n int) *Map {
	return &Map{
		Data: make(map[any]any, n),
		Keys: []any{},
	}
}

func (m *Map) Range(fn func(k, v any) bool) {
	m.Lock()
	defer m.Unlock()
	for _, k := range m.Keys {
		v, _ := m.Data[k]
		if !fn(k, v) {
			break
		}
	}
}

func (m *Map) Set(k, v any) {
	m.Lock()
	defer m.Unlock()
	var newKeys []any
	for _, key := range m.Keys {
		if key != k {
			newKeys = append(newKeys, key)
		}
	}
	m.Keys = newKeys
	m.Keys = append(m.Keys, k)
	m.Data[k] = v
}

func (m *Map) Get(k any) (v any, ok bool) {
	m.RLock()
	defer m.RUnlock()
	v, ok = m.Data[k]
	return v, ok
}

func (m *Map) Delete(k any) {
	m.Lock()
	defer m.Unlock()
	var newKeys []any
	for _, key := range m.Keys {
		if key != k {
			newKeys = append(newKeys, key)
		}
	}
	m.Keys = newKeys
	delete(m.Data, k)
}
