package marr

// Comparable 可比较类型约束（用于排序）
type Comparable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// Set 集合结构体
type Set[T comparable] struct {
	data map[T]struct{}
}

// NewSet 创建新集合
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		data: make(map[T]struct{}),
	}
}

// NewSetFromSlice 从切片创建集合
func NewSetFromSlice[T comparable](slice []T) *Set[T] {
	s := NewSet[T]()
	for _, v := range slice {
		s.Add(v)
	}
	return s
}

// Add 添加元素
func (s *Set[T]) Add(v T) {
	s.data[v] = struct{}{}
}

// Remove 删除元素
func (s *Set[T]) Remove(v T) {
	delete(s.data, v)
}

// Contains 判断是否包含元素
func (s *Set[T]) Contains(v T) bool {
	_, ok := s.data[v]
	return ok
}

// Size 返回集合大小
func (s *Set[T]) Size() int {
	return len(s.data)
}

// IsEmpty 判断是否为空
func (s *Set[T]) IsEmpty() bool {
	return len(s.data) == 0
}

// Clear 清空集合
func (s *Set[T]) Clear() {
	s.data = make(map[T]struct{})
}

// ToSlice 转换为切片
func (s *Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s.data))
	for v := range s.data {
		result = append(result, v)
	}
	return result
}

// Union 并集
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for v := range s.data {
		result.Add(v)
	}
	for v := range other.data {
		result.Add(v)
	}
	return result
}

// Intersection 交集（重合部分）
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	// 遍历较小的集合以提高效率
	small, large := s, other
	if s.Size() > other.Size() {
		small, large = other, s
	}
	for v := range small.data {
		if large.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

// Difference 差集（在 s 中但不在 other 中的元素）
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for v := range s.data {
		if !other.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

// SymmetricDifference 对称差集（只在其中一个集合中的元素）
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for v := range s.data {
		if !other.Contains(v) {
			result.Add(v)
		}
	}
	for v := range other.data {
		if !s.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

// Equal 判断两个集合是否相等
func (s *Set[T]) Equal(other *Set[T]) bool {
	if s.Size() != other.Size() {
		return false
	}
	for v := range s.data {
		if !other.Contains(v) {
			return false
		}
	}
	return true
}

// IsSubset 判断是否为子集
func (s *Set[T]) IsSubset(other *Set[T]) bool {
	if s.Size() > other.Size() {
		return false
	}
	for v := range s.data {
		if !other.Contains(v) {
			return false
		}
	}
	return true
}

// IsSuperset 判断是否为超集
func (s *Set[T]) IsSuperset(other *Set[T]) bool {
	return other.IsSubset(s)
}

// Clone 克隆集合
func (s *Set[T]) Clone() *Set[T] {
	result := NewSet[T]()
	for v := range s.data {
		result.Add(v)
	}
	return result
}
