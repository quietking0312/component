package mtool

import "sync"

type IndexMap[K comparable, V any] struct {
	data   []V
	index  map[K]int
	getKey func(value V) K
	mu     sync.RWMutex
}

func NewIndexMap[K comparable, V any](getKey func(V) K) *IndexMap[K, V] {
	return &IndexMap[K, V]{
		data:   make([]V, 0),
		index:  make(map[K]int),
		getKey: getKey,
	}
}

func (m *IndexMap[K, V]) Set(value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getKey == nil {
		panic("getKey function is nil")
	}
	key := m.getKey(value)
	if idx, exists := m.index[key]; exists {
		// 更新已存在的值
		m.data[idx] = value
	} else {
		// 添加新值
		idx = len(m.data)
		m.data = append(m.data, value)
		m.index[key] = idx
	}
}

func (m *IndexMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if idx, ok := m.index[key]; ok && idx >= 0 && idx < len(m.data) {
		return m.data[idx], true
	}
	return *new(V), false
}

func (m *IndexMap[K, V]) GetByIndex(idx int) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if idx >= 0 && idx < len(m.data) {
		return m.data[idx], true
	}
	return *new(V), false
}

func (m *IndexMap[K, V]) Delete(key K) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.index[key]
	if !exists {
		return false
	}

	// 从数组中删除
	m.data = append(m.data[:idx], m.data[idx+1:]...)
	delete(m.index, key)

	// 更新后续元素的索引
	for k, i := range m.index {
		if i > idx {
			m.index[k] = i - 1
		}
	}

	return true
}

func (m *IndexMap[K, V]) DeleteByIndex(idx int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idx < 0 || idx >= len(m.data) {
		return false
	}

	// 找到对应的键
	var key K
	for k, i := range m.index {
		if i == idx {
			key = k
			break
		}
	}

	// 删除元素
	m.data = append(m.data[:idx], m.data[idx+1:]...)
	delete(m.index, key)

	// 更新索引
	for k, i := range m.index {
		if i > idx {
			m.index[k] = i - 1
		}
	}

	return true
}

func (m *IndexMap[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// Keys 返回所有键
func (m *IndexMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]K, 0, len(m.data))
	for k, idx := range m.index {
		if idx >= 0 && idx < len(m.data) {
			keys = append(keys, k)
		}
	}
	return keys
}

func (m *IndexMap[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := make([]V, len(m.data))
	copy(values, m.data)
	return values
}

// Has 检查键是否存在
func (m *IndexMap[K, V]) Has(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.index[key]
	return exists
}

func (m *IndexMap[K, V]) IndexOf(key K) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	idx, exists := m.index[key]
	return idx, exists
}

// Range 遍历所有元素
func (m *IndexMap[K, V]) Range(f func(key K, value V) bool) {
	m.mu.RLock()
	// 复制数据避免长时间持有锁
	data := make([]V, len(m.data))
	copy(data, m.data)
	index := make(map[K]int, len(m.index))
	for k, v := range m.index {
		index[k] = v
	}
	m.mu.RUnlock()

	for key, idx := range index {
		if idx >= 0 && idx < len(data) {
			if !f(key, data[idx]) {
				break
			}
		}
	}
}

func (m *IndexMap[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make([]V, 0)
	m.index = make(map[K]int)
}

// Update 批量更新
func (m *IndexMap[K, V]) Update(values []V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, value := range values {
		key := m.getKey(value)
		if idx, exists := m.index[key]; exists {
			m.data[idx] = value
		} else {
			idx = len(m.data)
			m.data = append(m.data, value)
			m.index[key] = idx
		}
	}
}
