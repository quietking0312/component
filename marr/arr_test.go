package marr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	// 测试基本操作
	s := NewSet[int]()
	assert.True(t, s.IsEmpty())

	s.Add(1)
	s.Add(2)
	s.Add(3)
	assert.Equal(t, 3, s.Size())
	assert.False(t, s.IsEmpty())

	assert.True(t, s.Contains(1))
	assert.True(t, s.Contains(2))
	assert.False(t, s.Contains(4))

	s.Remove(2)
	assert.False(t, s.Contains(2))
	assert.Equal(t, 2, s.Size())

	// 测试 ToSlice
	slice := s.ToSlice()
	assert.Equal(t, 2, len(slice))

	// 测试 Clone
	s2 := s.Clone()
	assert.Equal(t, s.Size(), s2.Size())
	assert.True(t, s.Equal(s2))
}

func TestSetOperations(t *testing.T) {
	s1 := NewSetFromSlice([]int{1, 2, 3, 4, 5})
	s2 := NewSetFromSlice([]int{4, 5, 6, 7, 8})

	// 测试交集
	intersection := s1.Intersection(s2)
	assert.Equal(t, 2, intersection.Size())
	assert.True(t, intersection.Contains(4))
	assert.True(t, intersection.Contains(5))

	// 测试差集
	diff := s1.Difference(s2)
	assert.Equal(t, 3, diff.Size())
	assert.True(t, diff.Contains(1))
	assert.True(t, diff.Contains(2))
	assert.True(t, diff.Contains(3))

	// 测试并集
	union := s1.Union(s2)
	assert.Equal(t, 8, union.Size())

	// 测试对称差集
	symDiff := s1.SymmetricDifference(s2)
	assert.Equal(t, 6, symDiff.Size())
	assert.True(t, symDiff.Contains(1))
	assert.True(t, symDiff.Contains(6))
	assert.False(t, symDiff.Contains(4))
}

func TestIsSubset(t *testing.T) {
	s1 := NewSetFromSlice([]int{1, 2, 3})
	s2 := NewSetFromSlice([]int{1, 2, 3, 4, 5})
	s3 := NewSetFromSlice([]int{1, 2, 6})

	assert.True(t, s1.IsSubset(s2))
	assert.False(t, s2.IsSubset(s1))
	assert.False(t, s1.IsSubset(s3))

	assert.True(t, s2.IsSuperset(s1))
	assert.False(t, s1.IsSuperset(s2))
}

func TestCompare(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}

	result := Compare(a, b)

	// 验证交集
	assert.Equal(t, 2, len(result.Intersection))
	assert.Contains(t, result.Intersection, 4)
	assert.Contains(t, result.Intersection, 5)

	// 验证只在 A 中的元素
	assert.Equal(t, 3, len(result.OnlyInA))
	assert.Contains(t, result.OnlyInA, 1)
	assert.Contains(t, result.OnlyInA, 2)
	assert.Contains(t, result.OnlyInA, 3)

	// 验证只在 B 中的元素
	assert.Equal(t, 3, len(result.OnlyInB))
	assert.Contains(t, result.OnlyInB, 6)
	assert.Contains(t, result.OnlyInB, 7)
	assert.Contains(t, result.OnlyInB, 8)

	// 验证并集
	assert.Equal(t, 8, len(result.Union))

	// 验证对称差集
	assert.Equal(t, 6, len(result.SymmetricDifference))

	// 验证统计信息
	assert.Equal(t, 5, result.Stats.TotalA)
	assert.Equal(t, 5, result.Stats.TotalB)
	assert.Equal(t, 2, result.Stats.IntersectionCount)
	assert.Equal(t, 3, result.Stats.OnlyInACount)
	assert.Equal(t, 3, result.Stats.OnlyInBCount)
}

func TestCompareWithDuplicates(t *testing.T) {
	a := []int{1, 1, 2, 2, 3, 3}
	b := []int{2, 2, 3, 3, 4, 4}

	result := Compare(a, b)

	// 去重后的结果
	assert.Equal(t, 2, len(result.Intersection)) // 2, 3
	assert.Equal(t, 1, len(result.OnlyInA))      // 1
	assert.Equal(t, 1, len(result.OnlyInB))      // 4

	// 验证重复统计
	assert.Equal(t, 3, result.Stats.DuplicateCountA) // 6 - 3 = 3
	assert.Equal(t, 3, result.Stats.DuplicateCountB) // 6 - 3 = 3
}

func TestCompareWithOrder(t *testing.T) {
	a := []int{5, 4, 3, 2, 1}
	b := []int{8, 7, 6, 5, 4}

	// 保持顺序
	result := Compare(a, b, true)

	// 验证交集保持 A 的顺序
	assert.Equal(t, []int{5, 4}, result.Intersection)

	// 验证 OnlyInA 保持 A 的顺序
	assert.Equal(t, []int{3, 2, 1}, result.OnlyInA)

	// 验证 OnlyInB 保持 B 的顺序
	assert.Equal(t, []int{8, 7, 6}, result.OnlyInB)
}

func TestIntersection(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}

	result := Intersection(a, b)
	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, 4)
	assert.Contains(t, result, 5)
}

func TestDifference(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}

	result := Difference(a, b)
	assert.Equal(t, 3, len(result))
	assert.Contains(t, result, 1)
	assert.Contains(t, result, 2)
	assert.Contains(t, result, 3)
}

func TestDifferenceBoth(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}

	onlyInA, onlyInB := DifferenceBoth(a, b)

	assert.Equal(t, 3, len(onlyInA))
	assert.Contains(t, onlyInA, 1)
	assert.Contains(t, onlyInA, 2)
	assert.Contains(t, onlyInA, 3)

	assert.Equal(t, 3, len(onlyInB))
	assert.Contains(t, onlyInB, 6)
	assert.Contains(t, onlyInB, 7)
	assert.Contains(t, onlyInB, 8)
}

func TestUnion(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{3, 4, 5}

	result := Union(a, b)
	assert.Equal(t, 5, len(result))
}

func TestSymmetricDifference(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}

	result := SymmetricDifference(a, b)
	assert.Equal(t, 6, len(result))
	assert.Contains(t, result, 1)
	assert.Contains(t, result, 2)
	assert.Contains(t, result, 3)
	assert.Contains(t, result, 6)
	assert.Contains(t, result, 7)
	assert.Contains(t, result, 8)
}

func TestHasIntersection(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{3, 4, 5}
	c := []int{4, 5, 6}

	assert.True(t, HasIntersection(a, b))
	assert.False(t, HasIntersection(a, c))
}

func TestIsSubsetArray(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{1, 2, 3, 4, 5}
	c := []int{1, 2, 6}

	assert.True(t, IsSubset(a, b))
	assert.False(t, IsSubset(b, a))
	assert.False(t, IsSubset(a, c))

	assert.True(t, IsSuperset(b, a))
	assert.False(t, IsSuperset(a, b))
}

func TestEqual(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{3, 2, 1}
	c := []int{1, 2, 3, 4}

	// Equal 不考虑顺序
	assert.True(t, Equal(a, b))
	assert.False(t, Equal(a, c))

	// EqualStrict 考虑顺序
	assert.False(t, EqualStrict(a, b))
	assert.True(t, EqualStrict(a, a))
}

func TestUnique(t *testing.T) {
	arr := []int{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}

	// 不保持顺序
	result := Unique(arr)
	assert.Equal(t, 4, len(result))

	// 保持顺序
	resultWithOrder := Unique(arr, true)
	assert.Equal(t, 4, len(resultWithOrder))
	assert.Equal(t, []int{1, 2, 3, 4}, resultWithOrder)
}

func TestSortAndCompare(t *testing.T) {
	a := []int{5, 3, 1}
	b := []int{4, 3, 2}

	result := SortAndCompare(a, b)

	// 交集
	assert.Equal(t, 1, len(result.Intersection))
	assert.Contains(t, result.Intersection, 3)

	// OnlyInA
	assert.Equal(t, 2, len(result.OnlyInA))
	assert.Contains(t, result.OnlyInA, 1)
	assert.Contains(t, result.OnlyInA, 5)
}

func TestCompareWithStrings(t *testing.T) {
	a := []string{"apple", "banana", "cherry"}
	b := []string{"banana", "cherry", "date"}

	result := Compare(a, b)

	assert.Equal(t, 2, len(result.Intersection))
	assert.Contains(t, result.Intersection, "banana")
	assert.Contains(t, result.Intersection, "cherry")

	assert.Equal(t, 1, len(result.OnlyInA))
	assert.Contains(t, result.OnlyInA, "apple")

	assert.Equal(t, 1, len(result.OnlyInB))
	assert.Contains(t, result.OnlyInB, "date")
}

func TestCompareByKey(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	a := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	b := []User{
		{ID: 2, Name: "Bob2"}, // ID 相同，Name 不同
		{ID: 3, Name: "Charlie2"},
		{ID: 4, Name: "David"},
	}

	result := CompareByKey(a, b, func(u User) int { return u.ID })

	// 交集应该返回 A 中的元素
	assert.Equal(t, 2, len(result.Intersection))

	// OnlyInA
	assert.Equal(t, 1, len(result.OnlyInA))
	assert.Equal(t, 1, result.OnlyInA[0].ID)

	// OnlyInB
	assert.Equal(t, 1, len(result.OnlyInB))
	assert.Equal(t, 4, result.OnlyInB[0].ID)
}

func TestUniqueByKey(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	arr := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 1, Name: "Alice2"}, // 重复的 ID
		{ID: 3, Name: "Charlie"},
	}

	result := UniqueByKey(arr, func(u User) int { return u.ID })

	assert.Equal(t, 3, len(result))
}

func TestGroupBy(t *testing.T) {
	type User struct {
		Age  int
		Name string
	}

	arr := []User{
		{Age: 20, Name: "Alice"},
		{Age: 20, Name: "Bob"},
		{Age: 30, Name: "Charlie"},
	}

	result := GroupBy(arr, func(u User) int { return u.Age })

	assert.Equal(t, 2, len(result[20]))
	assert.Equal(t, 1, len(result[30]))
}

func TestToMap(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	arr := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 1, Name: "Alice2"}, // 重复的 ID，后面的会覆盖前面的
	}

	result := ToMap(arr, func(u User) int { return u.ID })

	assert.Equal(t, 2, len(result))
	assert.Equal(t, "Alice2", result[1].Name) // 后面的覆盖了前面的
}

func TestEmptyArrays(t *testing.T) {
	empty := []int{}
	nonEmpty := []int{1, 2, 3}

	result := Compare(empty, nonEmpty)

	assert.Equal(t, 0, len(result.Intersection))
	assert.Equal(t, 0, len(result.OnlyInA))
	assert.Equal(t, 3, len(result.OnlyInB))
}

func TestIdenticalArrays(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}

	result := Compare(a, b)

	assert.Equal(t, 3, len(result.Intersection))
	assert.Equal(t, 0, len(result.OnlyInA))
	assert.Equal(t, 0, len(result.OnlyInB))
	assert.Equal(t, 3, len(result.Union))
	assert.Equal(t, 0, len(result.SymmetricDifference))
}

func BenchmarkIntersection(b *testing.B) {
	a := make([]int, 1000)
	b2 := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		a[i] = i
		b2[i] = i + 500
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Intersection(a, b2)
	}
}

func BenchmarkCompare(b *testing.B) {
	a := make([]int, 1000)
	b2 := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		a[i] = i
		b2[i] = i + 500
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compare(a, b2)
	}
}
