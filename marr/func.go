package marr

// CompareFunc 自定义比较函数类型
type CompareFunc[T any] func(a, b T) bool

// KeyFunc 键提取函数类型
type KeyFunc[T any, K comparable] func(T) K

// CompareResultFunc 使用自定义比较函数的比较结果
type CompareResultFunc[T any] struct {
	// Intersection 重合部分（两个数组都有的元素，来自数组 A）
	Intersection []T `json:"intersection"`
	// OnlyInA 只在 A 中的元素
	OnlyInA []T `json:"only_in_a"`
	// OnlyInB 只在 B 中的元素
	OnlyInB []T `json:"only_in_b"`
	// Union 并集（所有元素，去重后）
	Union []T `json:"union"`
	// SymmetricDifference 对称差集
	SymmetricDifference []T `json:"symmetric_difference"`
}

// CompareByKey 使用键提取函数比较两个数组
// 适用于结构体数组，通过提取关键字段进行比较
func CompareByKey[T any, K comparable](a, b []T, keyFn KeyFunc[T, K], keepOrder ...bool) *CompareResultFunc[T] {
	keep := false
	if len(keepOrder) > 0 {
		keep = keepOrder[0]
	}

	// 提取键并创建映射
	keyToValueA := make(map[K]T)
	keyToValueB := make(map[K]T)
	keySetA := NewSet[K]()
	keySetB := NewSet[K]()

	for _, v := range a {
		k := keyFn(v)
		keyToValueA[k] = v
		keySetA.Add(k)
	}

	for _, v := range b {
		k := keyFn(v)
		keyToValueB[k] = v
		keySetB.Add(k)
	}

	// 计算各部分的键
	intersectionKeys := keySetA.Intersection(keySetB)
	onlyInAKeys := keySetA.Difference(keySetB)
	onlyInBKeys := keySetB.Difference(keySetA)
	unionKeys := keySetA.Union(keySetB)
	symDiffKeys := keySetA.SymmetricDifference(keySetB)

	// 根据键获取值
	result := &CompareResultFunc[T]{
		Intersection:        getValuesByKeys(keyToValueA, intersectionKeys),
		OnlyInA:             getValuesByKeys(keyToValueA, onlyInAKeys),
		OnlyInB:             getValuesByKeys(keyToValueB, onlyInBKeys),
		Union:               mergeValuesByKeys(keyToValueA, keyToValueB, unionKeys),
		SymmetricDifference: mergeValuesByKeys(keyToValueA, keyToValueB, symDiffKeys),
	}

	// 保持顺序
	if keep {
		result.Intersection = filterAndKeepOrderByKey(a, intersectionKeys, keyFn)
		result.OnlyInA = filterAndKeepOrderByKey(a, onlyInAKeys, keyFn)
		result.OnlyInB = filterAndKeepOrderByKey(b, onlyInBKeys, keyFn)
		result.Union = mergeAndKeepOrderByKey(a, b, unionKeys, keyFn)
		result.SymmetricDifference = mergeAndKeepOrderByKey(a, b, symDiffKeys, keyFn)
	}

	return result
}

// getValuesByKeys 根据键从映射中获取值
func getValuesByKeys[K comparable, T any](m map[K]T, s *Set[K]) []T {
	result := make([]T, 0, s.Size())
	s.Range(func(k K) bool {
		if v, ok := m[k]; ok {
			result = append(result, v)
		}
		return true
	})
	return result
}

// mergeValuesByKeys 合并两个映射中的值（优先使用 A 中的值）
func mergeValuesByKeys[K comparable, T any](mapA, mapB map[K]T, s *Set[K]) []T {
	result := make([]T, 0, s.Size())
	s.Range(func(k K) bool {
		if v, ok := mapA[k]; ok {
			result = append(result, v)
		} else if v, ok := mapB[k]; ok {
			result = append(result, v)
		}
		return true
	})
	return result
}

// filterAndKeepOrderByKey 根据键集合过滤并保持原数组顺序
func filterAndKeepOrderByKey[T any, K comparable](arr []T, s *Set[K], keyFn KeyFunc[T, K]) []T {
	seen := NewSet[K]()
	result := make([]T, 0)
	for _, v := range arr {
		k := keyFn(v)
		if s.Contains(k) && !seen.Contains(k) {
			result = append(result, v)
			seen.Add(k)
		}
	}
	return result
}

// mergeAndKeepOrderByKey 合并两个数组并保持顺序
func mergeAndKeepOrderByKey[T any, K comparable](a, b []T, s *Set[K], keyFn KeyFunc[T, K]) []T {
	seen := NewSet[K]()
	result := make([]T, 0)

	// 先遍历 a
	for _, v := range a {
		k := keyFn(v)
		if s.Contains(k) && !seen.Contains(k) {
			result = append(result, v)
			seen.Add(k)
		}
	}

	// 再遍历 b
	for _, v := range b {
		k := keyFn(v)
		if s.Contains(k) && !seen.Contains(k) {
			result = append(result, v)
			seen.Add(k)
		}
	}

	return result
}

// IntersectionByKey 使用键提取函数求交集
func IntersectionByKey[T any, K comparable](a, b []T, keyFn KeyFunc[T, K], keepOrder ...bool) []T {
	return CompareByKey(a, b, keyFn, keepOrder...).Intersection
}

// DifferenceByKey 使用键提取函数求差集
func DifferenceByKey[T any, K comparable](a, b []T, keyFn KeyFunc[T, K], keepOrder ...bool) []T {
	return CompareByKey(a, b, keyFn, keepOrder...).OnlyInA
}

// DifferenceBothByKey 使用键提取函数求双向差集
func DifferenceBothByKey[T any, K comparable](a, b []T, keyFn KeyFunc[T, K], keepOrder ...bool) (onlyInA, onlyInB []T) {
	result := CompareByKey(a, b, keyFn, keepOrder...)
	return result.OnlyInA, result.OnlyInB
}

// UnionByKey 使用键提取函数求并集
func UnionByKey[T any, K comparable](a, b []T, keyFn KeyFunc[T, K], keepOrder ...bool) []T {
	return CompareByKey(a, b, keyFn, keepOrder...).Union
}

// UniqueByKey 使用键提取函数去重
func UniqueByKey[T any, K comparable](arr []T, keyFn KeyFunc[T, K], keepOrder ...bool) []T {
	seen := NewSet[K]()
	result := make([]T, 0)

	for _, v := range arr {
		k := keyFn(v)
		if !seen.Contains(k) {
			result = append(result, v)
			seen.Add(k)
		}
	}

	return result
}

// GroupBy 根据键函数对数组进行分组
func GroupBy[T any, K comparable](arr []T, keyFn KeyFunc[T, K]) map[K][]T {
	result := make(map[K][]T)
	for _, v := range arr {
		k := keyFn(v)
		result[k] = append(result[k], v)
	}
	return result
}

// ToMap 将数组转换为映射（键函数返回的键 -> 值）
// 如果数组中有重复键，后面的值会覆盖前面的值
func ToMap[T any, K comparable](arr []T, keyFn KeyFunc[T, K]) map[K]T {
	result := make(map[K]T)
	for _, v := range arr {
		k := keyFn(v)
		result[k] = v
	}
	return result
}

// ToMapWithMerge 将数组转换为映射，对于重复键使用合并函数处理
func ToMapWithMerge[T any, K comparable](arr []T, keyFn KeyFunc[T, K], mergeFn func(existing, new T) T) map[K]T {
	result := make(map[K]T)
	for _, v := range arr {
		k := keyFn(v)
		if existing, ok := result[k]; ok {
			result[k] = mergeFn(existing, v)
		} else {
			result[k] = v
		}
	}
	return result
}
