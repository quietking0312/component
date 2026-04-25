package marr

import (
	"fmt"
)

// ExampleCompare 基础比较示例
func ExampleCompare() {
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}

	// 使用 keepOrder=true 保持原数组顺序
	result := Compare(a, b, true)

	fmt.Printf("数组 A: %v\n", a)
	fmt.Printf("数组 B: %v\n", b)
	fmt.Printf("重合部分: %v\n", result.Intersection)
	fmt.Printf("只在 A 中: %v\n", result.OnlyInA)
	fmt.Printf("只在 B 中: %v\n", result.OnlyInB)

	// Output:
	// 数组 A: [1 2 3 4 5]
	// 数组 B: [4 5 6 7 8]
	// 重合部分: [4 5]
	// 只在 A 中: [1 2 3]
	// 只在 B 中: [6 7 8]
}

// ExampleCompare_keepOrder 保持原数组顺序示例
func ExampleCompare_keepOrder() {
	a := []int{5, 4, 3, 2, 1}
	b := []int{8, 7, 6, 5, 4}

	// 第三个参数 true 表示保持原数组顺序
	result := Compare(a, b, true)

	fmt.Printf("数组 A: %v\n", a)
	fmt.Printf("数组 B: %v\n", b)
	fmt.Printf("重合部分（保持A的顺序）: %v\n", result.Intersection)
	fmt.Printf("只在 A 中（保持A的顺序）: %v\n", result.OnlyInA)
	fmt.Printf("只在 B 中（保持B的顺序）: %v\n", result.OnlyInB)

	// Output:
	// 数组 A: [5 4 3 2 1]
	// 数组 B: [8 7 6 5 4]
	// 重合部分（保持A的顺序）: [5 4]
	// 只在 A 中（保持A的顺序）: [3 2 1]
	// 只在 B 中（保持B的顺序）: [8 7 6]
}

// ExampleIntersection 求交集示例
func ExampleIntersection() {
	a := []string{"apple", "banana", "cherry", "date"}
	b := []string{"banana", "date", "fig", "grape"}

	// 使用 keepOrder=true 保持 A 数组的顺序
	result := Intersection(a, b, true)

	fmt.Printf("数组 A: %v\n", a)
	fmt.Printf("数组 B: %v\n", b)
	fmt.Printf("交集: %v\n", result)

	// Output:
	// 数组 A: [apple banana cherry date]
	// 数组 B: [banana date fig grape]
	// 交集: [banana date]
}

// ExampleDifference 求差集示例
func ExampleDifference() {
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}

	// 在 A 中但不在 B 中的元素（保持顺序）
	onlyInA := Difference(a, b, true)

	// 双向差集（保持顺序）
	onlyInA2, onlyInB := DifferenceBoth(a, b, true)

	fmt.Printf("只在 A 中: %v\n", onlyInA)
	fmt.Printf("双向差集 - 只在 A: %v, 只在 B: %v\n", onlyInA2, onlyInB)

	// Output:
	// 只在 A 中: [1 2 3]
	// 双向差集 - 只在 A: [1 2 3], 只在 B: [6 7 8]
}

// ExampleUnique 数组去重示例
func ExampleUnique() {
	arr := []int{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}

	// 保持顺序（推荐，结果可预期）
	result := Unique(arr, true)

	fmt.Printf("原数组: %v\n", arr)
	fmt.Printf("去重后: %v\n", result)

	// Output:
	// 原数组: [1 2 2 3 3 3 4 4 4 4]
	// 去重后: [1 2 3 4]
}

// ExampleSet 集合操作示例
func ExampleSet() {
	// 创建集合并进行各种操作
	s1 := NewSetFromSlice([]int{1, 2, 3, 4, 5})
	s2 := NewSetFromSlice([]int{4, 5, 6, 7, 8})

	fmt.Printf("集合1: %v\n", s1.ToSlice())
	fmt.Printf("集合2: %v\n", s2.ToSlice())
	fmt.Printf("交集: %v\n", s1.Intersection(s2).ToSlice())
	fmt.Printf("并集: %v\n", s1.Union(s2).ToSlice())
	fmt.Printf("差集 (s1-s2): %v\n", s1.Difference(s2).ToSlice())
	fmt.Printf("对称差集: %v\n", s1.SymmetricDifference(s2).ToSlice())
	fmt.Printf("s1 是否为 s2 的子集: %v\n", s1.IsSubset(s2))
}

// ExampleCompareByKey 结构体数组比较示例
func ExampleCompareByKey() {
	type User struct {
		ID   int
		Name string
		Age  int
	}

	// 两个用户数组，可能有部分用户 ID 相同
	usersA := []User{
		{ID: 1, Name: "Alice", Age: 25},
		{ID: 2, Name: "Bob", Age: 30},
		{ID: 3, Name: "Charlie", Age: 35},
	}

	usersB := []User{
		{ID: 2, Name: "Bob", Age: 30},
		{ID: 3, Name: "Charlie", Age: 35},
		{ID: 4, Name: "David", Age: 40},
	}

	// 使用 ID 作为比较键
	result := CompareByKey(usersA, usersB, func(u User) int { return u.ID })

	fmt.Println("基于 ID 的比较结果：")
	fmt.Printf("重合用户: %v\n", result.Intersection)
	fmt.Printf("只在 A 中的用户: %v\n", result.OnlyInA)
	fmt.Printf("只在 B 中的用户: %v\n", result.OnlyInB)
}

// ExampleGroupBy 分组示例
func ExampleGroupBy() {
	type Product struct {
		Category string
		Name     string
		Price    float64
	}

	products := []Product{
		{Category: "水果", Name: "苹果", Price: 5.0},
		{Category: "水果", Name: "香蕉", Price: 3.0},
		{Category: "蔬菜", Name: "西红柿", Price: 4.0},
		{Category: "蔬菜", Name: "黄瓜", Price: 2.5},
	}

	// 按类别分组
	grouped := GroupBy(products, func(p Product) string { return p.Category })

	fmt.Printf("水果类商品数量: %d\n", len(grouped["水果"]))
	fmt.Printf("蔬菜类商品数量: %d\n", len(grouped["蔬菜"]))

	// Output:
	// 水果类商品数量: 2
	// 蔬菜类商品数量: 2
}

// ExampleToMap 转换为映射示例
func ExampleToMap() {
	type User struct {
		ID   int
		Name string
	}

	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	// 转换为以 ID 为键的映射
	userMap := ToMap(users, func(u User) int { return u.ID })

	fmt.Printf("ID=1 的用户: %v\n", userMap[1])
	fmt.Printf("ID=2 的用户: %v\n", userMap[2])
}

// ExampleCompareResult_stats 查看比较统计信息示例
func ExampleCompareResult_stats() {
	a := []int{1, 1, 2, 2, 3, 3, 4, 5} // 有重复元素
	b := []int{4, 4, 5, 5, 6, 6, 7, 8}

	result := Compare(a, b)

	fmt.Printf("数组 A 总数: %d\n", result.Stats.TotalA)
	fmt.Printf("数组 B 总数: %d\n", result.Stats.TotalB)
	fmt.Printf("数组 A 去重后: %d\n", result.Stats.TotalA-result.Stats.DuplicateCountA)
	fmt.Printf("数组 B 去重后: %d\n", result.Stats.TotalB-result.Stats.DuplicateCountB)
	fmt.Printf("重合数量: %d\n", result.Stats.IntersectionCount)
	fmt.Printf("只在 A 中: %d\n", result.Stats.OnlyInACount)
	fmt.Printf("只在 B 中: %d\n", result.Stats.OnlyInBCount)

	// Output:
	// 数组 A 总数: 8
	// 数组 B 总数: 8
	// 数组 A 去重后: 5
	// 数组 B 去重后: 5
	// 重合数量: 2
	// 只在 A 中: 3
	// 只在 B 中: 3
}
