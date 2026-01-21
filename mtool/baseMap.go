package mtool

import (
	"sync"
	"sync/atomic"
)

const (
	BaseMapModeCOW  = 0 // 适合读多写少的模式
	BaseMapModeLock = 1 // 适合读写均匀的模式
)

// 最基础的 map + sync.Locker
// set, del 采用了 copy-on-write 模式 适合读多写少的场景
type BaseMap[K comparable, V any] struct {
	data atomic.Pointer[map[K]V]
	mu   sync.RWMutex
	mode int
}

func NewBaseMap[K comparable, V any](mode int) *BaseMap[K, V] {
	m := BaseMapModeCOW
	if mode == BaseMapModeLock {
		m = BaseMapModeLock
	}
	return &BaseMap[K, V]{
		mode: m,
	}
}

func (m *BaseMap[K, V]) snapshot() map[K]V {
	data := m.data.Load()
	if data == nil {
		return nil
	}
	snapshot := make(map[K]V, len(*data))
	for k, v := range *data {
		snapshot[k] = v
	}
	return snapshot
}

func (m *BaseMap[K, V]) Swap(data map[K]V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data.Swap(&data)
}

func (m *BaseMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data := m.data.Load()
	var newData map[K]V
	if data == nil {
		newData = make(map[K]V)
	} else if m.mode == BaseMapModeCOW {
		newData = m.snapshot()
	} else if m.mode == BaseMapModeLock {
		newData = *data
	}
	newData[key] = value
	m.data.Store(&newData)
}

func (m *BaseMap[K, V]) Del(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data := m.data.Load()
	var newData map[K]V
	if data == nil {
		newData = make(map[K]V)
	} else if m.mode == BaseMapModeCOW {
		newData = m.snapshot()
	} else if m.mode == BaseMapModeLock {
		newData = *data
	}
	delete(newData, key)
	m.data.Store(&newData)
}

func (m *BaseMap[K, V]) Get(key K) (V, bool) {
	if m.mode == BaseMapModeLock {
		m.mu.RLock()
		defer m.mu.RUnlock()
	}
	data := m.data.Load()
	if data == nil {
		return *new(V), false
	}
	v, ok := (*data)[key]
	if ok {
		return v, true
	}
	return *new(V), false
}

func (m *BaseMap[K, V]) Range(fc func(key K, value V) bool) {
	data := m.data.Load()
	if data == nil {
		return
	}
	if m.mode == BaseMapModeLock {
		m.mu.RLock()
		copyData := m.snapshot()
		m.mu.RUnlock()
		for k, v := range copyData {
			if !fc(k, v) {
				break
			}
		}
	} else {
		for k, v := range *data {
			if !fc(k, v) {
				break
			}
		}
	}
}

func (m *BaseMap[K, V]) Keys() []K {
	data := m.data.Load()
	if data == nil {
		return make([]K, 0)
	}
	if m.mode == BaseMapModeLock {
		m.mu.RLock()
		defer m.mu.RUnlock()
	}
	return GetMapKeys(*data)
}

func (m *BaseMap[K, V]) Values() []V {
	data := m.data.Load()
	if data == nil {
		return make([]V, 0)
	}
	if m.mode == BaseMapModeLock {
		m.mu.RLock()
		defer m.mu.RUnlock()
	}
	return GetMapValues(*data)
}

func (m *BaseMap[K, V]) Len() int {
	data := m.data.Load()
	if data == nil {
		return 0
	}
	if m.mode == BaseMapModeLock {
		m.mu.RLock()
		defer m.mu.RUnlock()
	}
	return len(*data)
}
