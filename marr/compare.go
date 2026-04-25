package marr

import (
	"sort"
)

// CompareResult 数组比较结果
type CompareResult[T comparable] struct {
	// Intersection 重合部分（两个数组都有的元素）
	Intersection []T `json:"intersection"`
	// OnlyInA 只在 A 中的元素
	OnlyInA []T `json:"only_in_a"`
	// OnlyInB 只在 B 中的元素
	OnlyInB []T `json:"only_in_b"`
	// Union 并集（所有元素去重）
	Union []T `json:"union"`
	// SymmetricDifference 对称差集（只在其中一个数组中的元素）
	SymmetricDifference []T `json:"symmetric_difference"`
	// Stats 统计信息
	Stats CompareStats `json:"stats"`
}

// CompareStats 比较统计信息
type CompareStats struct {
	TotalA             int `json:"total_a"`              // A 数组总数
	TotalB             int `json:"total_b"`              // B 数组总数
	IntersectionCount  int `json:"intersection_count"`   // 重合数量
	OnlyInACount       int `json:"only_in_a_count"`      // 只在 A 中的数量
	OnlyInBCount       int `json:"only_in_b_count"`      // 只在 B 中的数量
	UnionCount         int `json:"union_count"`          // 并集数量
	SymmetricDiffCount int `json:"symmetric_diff_count"` // 对称差集数量
	DuplicateCountA    int `json:"duplicate_count_a"`    // A 中重复元素数量
	DuplicateCountB    int `json:"duplicate_count_b"`    // B 中重复元素数量
}

// Compare 比较两个数组，返回重合部分和不重合部分
// 参数 keepOrder 表示是否保持原数组顺序
func Compare[T comparable](a, b []T, keepOrder ...bool) *CompareResult[T] {
	keep := false
	if len(keepOrder) > 0 {
		keep = keepOrder[0]
	}

	// 创建集合进行计算
	setA := NewSetFromSlice(a)
	setB := NewSetFromSlice(b)

	// 计算各部分
	intersectionSet := setA.Intersection(setB)
	onlyInASet := setA.Difference(setB)
	onlyInBSet := setB.Difference(setA)
	unionSet := setA.Union(setB)
	symDiffSet := setA.SymmetricDifference(setB)

	result := &CompareResult[T]{
		Intersection:        intersectionSet.ToSlice(),
		OnlyInA:             onlyInASet.ToSlice(),
		OnlyInB:             onlyInBSet.ToSlice(),
		Union:               unionSet.ToSlice(),
		SymmetricDifference: symDiffSet.ToSlice(),
		Stats: CompareStats{
			TotalA:             len(a),
			TotalB:             len(b),
			IntersectionCount:  intersectionSet.Size(),
			OnlyInACount:       onlyInASet.Size(),
			OnlyInBCount:       onlyInBSet.Size(),
			UnionCount:         unionSet.Size(),
			SymmetricDiffCount: symDiffSet.Size(),
			DuplicateCountA:    len(a) - setA.Size(),
			DuplicateCountB:    len(b) - setB.Size(),
		},
	}

	// 如果需要保持原数组顺序
	if keep {
		result.Intersection = filterAndKeepOrder(a, intersectionSet)
		result.OnlyInA = filterAndKeepOrder(a, onlyInASet)
		result.OnlyInB = filterAndKeepOrder(b, onlyInBSet)
		result.Union = mergeAndKeepOrder(a, b, unionSet)
		result.SymmetricDifference = mergeAndKeepOrder(a, b, symDiffSet)
	}

	return result
}

// filterAndKeepOrder 根据集合过滤并保留原数组顺序
func filterAndKeepOrder[T comparable](arr []T, s *Set[T]) []T {
	seen := NewSet[T]()
	result := make([]T, 0)
	for _, v := range arr {
		if s.Contains(v) && !seen.Contains(v) {
			result = append(result, v)
			seen.Add(v)
		}
	}
	return result
}

// mergeAndKeepOrder 合并两个数组并保留顺序，同时去重
func mergeAndKeepOrder[T comparable](a, b []T, s *Set[T]) []T {
	seen := NewSet[T]()
	result := make([]T, 0)

	// 先遍历 a
	for _, v := range a {
		if s.Contains(v) && !seen.Contains(v) {
			result = append(result, v)
			seen.Add(v)
		}
	}

	// 再遍历 b
	for _, v := range b {
		if s.Contains(v) && !seen.Contains(v) {
			result = append(result, v)
			seen.Add(v)
		}
	}

	return result
}

// Intersection 求数组交集（重合部分）
func Intersection[T comparable](a, b []T, keepOrder ...bool) []T {
	return Compare(a, b, keepOrder...).Intersection
}

// Difference 求数组差集（在 a 中但不在 b 中的元素）
func Difference[T comparable](a, b []T, keepOrder ...bool) []T {
	return Compare(a, b, keepOrder...).OnlyInA
}

// DifferenceBoth 求双向差集（返回只在 a 中的和只在 b 中的）
func DifferenceBoth[T comparable](a, b []T, keepOrder ...bool) (onlyInA, onlyInB []T) {
	result := Compare(a, b, keepOrder...)
	return result.OnlyInA, result.OnlyInB
}

// Union 求数组并集
func Union[T comparable](a, b []T, keepOrder ...bool) []T {
	return Compare(a, b, keepOrder...).Union
}

// SymmetricDifference 求对称差集（只在其中一个数组中的元素）
func SymmetricDifference[T comparable](a, b []T, keepOrder ...bool) []T {
	return Compare(a, b, keepOrder...).SymmetricDifference
}

// HasIntersection 判断两个数组是否有交集
func HasIntersection[T comparable](a, b []T) bool {
	setB := NewSetFromSlice(b)
	for _, v := range a {
		if setB.Contains(v) {
			return true
		}
	}
	return false
}

// IsSubset 判断 a 是否为 b 的子集
func IsSubset[T comparable](a, b []T) bool {
	if len(a) > len(b) {
		return false
	}
	setB := NewSetFromSlice(b)
	for _, v := range a {
		if !setB.Contains(v) {
			return false
		}
	}
	return true
}

// IsSuperset 判断 a 是否为 b 的超集
func IsSuperset[T comparable](a, b []T) bool {
	return IsSubset(b, a)
}

// Equal 判断两个数组包含的元素是否相同（不考虑顺序和重复）
func Equal[T comparable](a, b []T) bool {
	return NewSetFromSlice(a).Equal(NewSetFromSlice(b))
}

// EqualStrict 严格判断两个数组是否相同（考虑顺序，不考虑重复）
func EqualStrict[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Unique 数组去重
func Unique[T comparable](arr []T, keepOrder ...bool) []T {
	set := NewSetFromSlice(arr)
	if len(keepOrder) > 0 && keepOrder[0] {
		return filterAndKeepOrder(arr, set)
	}
	return set.ToSlice()
}

// SortAndCompare 排序后比较（适用于有序类型）
func SortAndCompare[T Comparable](a, b []T) *CompareResult[T] {
	// 去重并排序
	uniqueA := Unique(a)
	uniqueB := Unique(b)

	sort.Slice(uniqueA, func(i, j int) bool { return uniqueA[i] < uniqueA[j] })
	sort.Slice(uniqueB, func(i, j int) bool { return uniqueB[i] < uniqueB[j] })

	return Compare(uniqueA, uniqueB, false)
}
