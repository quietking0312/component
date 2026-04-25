# marr - 数组/集合操作组件

封装数组比较、集合操作等功能，支持查询两个数组的重合部分和不重合部分。

## 特性

- 🔍 **数组比较**：求交集、差集、并集、对称差集
- 📊 **详细统计**：提供比较结果的详细统计信息
- 🔄 **保持顺序**：可选择是否保持原数组顺序
- 🎯 **泛型支持**：支持任意可比较类型的数组
- 🛠️ **结构体支持**：通过键提取函数支持结构体数组比较
- 📦 **集合操作**：完整的集合数据结构及操作

## 快速开始

### 基础数组比较

```go
import "admin_server/component/marr"

a := []int{1, 2, 3, 4, 5}
b := []int{4, 5, 6, 7, 8}

// 完整比较，获取所有信息
result := marr.Compare(a, b)

fmt.Println(result.Intersection)        // [4 5] - 重合部分
fmt.Println(result.OnlyInA)             // [1 2 3] - 只在 A 中
fmt.Println(result.OnlyInB)             // [6 7 8] - 只在 B 中
fmt.Println(result.Union)               // [1 2 3 4 5 6 7 8] - 并集
fmt.Println(result.SymmetricDifference) // [1 2 3 6 7 8] - 对称差集

// 查看统计信息
fmt.Println(result.Stats.TotalA)             // 5
fmt.Println(result.Stats.IntersectionCount)  // 2
fmt.Println(result.Stats.DuplicateCountA)    // 0
```

### 保持原数组顺序

```go
a := []int{5, 4, 3, 2, 1}
b := []int{8, 7, 6, 5, 4}

// 第三个参数 true 表示保持原数组顺序
result := marr.Compare(a, b, true)

// 结果保持 A 数组的顺序
fmt.Println(result.Intersection) // [5 4] (不是 [4 5])
fmt.Println(result.OnlyInA)      // [3 2 1]
```

### 便捷函数

```go
a := []int{1, 2, 3, 4, 5}
b := []int{4, 5, 6, 7, 8}

// 只获取交集
intersection := marr.Intersection(a, b) // [4 5]

// 只获取差集（在 a 中但不在 b 中）
diff := marr.Difference(a, b) // [1 2 3]

// 双向差集
onlyInA, onlyInB := marr.DifferenceBoth(a, b)

// 并集
union := marr.Union(a, b)

// 对称差集
symDiff := marr.SymmetricDifference(a, b)

// 判断是否有交集
hasInter := marr.HasIntersection(a, b) // true

// 判断是否为子集
isSubset := marr.IsSubset([]int{1, 2}, []int{1, 2, 3}) // true
```

### 数组去重

```go
arr := []int{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}

// 去重（不保证顺序）
unique := marr.Unique(arr) // [1 2 3 4]

// 去重并保持原数组顺序
uniqueOrdered := marr.Unique(arr, true) // [1 2 3 4]
```

## 结构体数组比较

对于结构体数组，可以通过键提取函数进行比较：

```go
type User struct {
    ID   int
    Name string
    Age  int
}

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
result := marr.CompareByKey(usersA, usersB, func(u User) int { return u.ID })

fmt.Println(result.Intersection) // [{ID:2 Name:Bob Age:30} {ID:3 Name:Charlie Age:35}]
fmt.Println(result.OnlyInA)      // [{ID:1 Name:Alice Age:25}]
fmt.Println(result.OnlyInB)      // [{ID:4 Name:David Age:40}]
```

### 其他结构体操作

```go
// 根据字段去重
uniqueUsers := marr.UniqueByKey(users, func(u User) int { return u.ID })

// 分组
usersByAge := marr.GroupBy(users, func(u User) int { return u.Age })
// 返回: map[int][]User

// 转换为映射
userMap := marr.ToMap(users, func(u User) int { return u.ID })
// 返回: map[int]User
```

## 集合操作

```go
// 创建集合
s1 := marr.NewSetFromSlice([]int{1, 2, 3, 4, 5})
s2 := marr.NewSetFromSlice([]int{4, 5, 6, 7, 8})

// 基本操作
s1.Add(6)
s1.Remove(1)
exists := s1.Contains(2) // true

// 集合运算
intersection := s1.Intersection(s2)
difference := s1.Difference(s2)
union := s1.Union(s2)
symDiff := s1.SymmetricDifference(s2)

// 判断关系
isSubset := s1.IsSubset(s2)
isSuperset := s1.IsSuperset(s2)
isEqual := s1.Equal(s2)

// 转换为切片
slice := s1.ToSlice()
```

## API 文档

### 数组比较函数

| 函数 | 说明 |
|------|------|
| `Compare(a, b []T, keepOrder ...bool) *CompareResult[T]` | 完整比较两个数组 |
| `Intersection(a, b []T, keepOrder ...bool) []T` | 求交集（重合部分） |
| `Difference(a, b []T, keepOrder ...bool) []T` | 求差集（只在 A 中） |
| `DifferenceBoth(a, b []T, keepOrder ...bool) ([]T, []T)` | 双向差集 |
| `Union(a, b []T, keepOrder ...bool) []T` | 求并集 |
| `SymmetricDifference(a, b []T, keepOrder ...bool) []T` | 对称差集 |
| `HasIntersection(a, b []T) bool` | 判断是否有交集 |
| `IsSubset(a, b []T) bool` | 判断 A 是否为 B 的子集 |
| `IsSuperset(a, b []T) bool` | 判断 A 是否为 B 的超集 |
| `Equal(a, b []T) bool` | 判断两数组元素是否相同（不考虑顺序和重复） |
| `EqualStrict(a, b []T) bool` | 严格判断（考虑顺序） |
| `Unique(arr []T, keepOrder ...bool) []T` | 数组去重 |
| `SortAndCompare(a, b []T) *CompareResult[T]` | 排序后比较 |

### CompareResult 结构

```go
type CompareResult[T comparable] struct {
    Intersection        []T          // 重合部分
    OnlyInA             []T          // 只在 A 中
    OnlyInB             []T          // 只在 B 中
    Union               []T          // 并集
    SymmetricDifference []T          // 对称差集
    Stats               CompareStats // 统计信息
}

type CompareStats struct {
    TotalA             int // A 数组总数
    TotalB             int // B 数组总数
    IntersectionCount  int // 重合数量
    OnlyInACount       int // 只在 A 中的数量
    OnlyInBCount       int // 只在 B 中的数量
    UnionCount         int // 并集数量
    SymmetricDiffCount int // 对称差集数量
    DuplicateCountA    int // A 中重复元素数量
    DuplicateCountB    int // B 中重复元素数量
}
```

### 结构体数组函数

| 函数 | 说明 |
|------|------|
| `CompareByKey(a, b []T, keyFn KeyFunc[T,K], keepOrder ...bool) *CompareResultFunc[T]` | 使用键函数比较 |
| `IntersectionByKey(a, b []T, keyFn KeyFunc[T,K], keepOrder ...bool) []T` | 求交集 |
| `DifferenceByKey(a, b []T, keyFn KeyFunc[T,K], keepOrder ...bool) []T` | 求差集 |
| `UniqueByKey(arr []T, keyFn KeyFunc[T,K], keepOrder ...bool) []T` | 根据键去重 |
| `GroupBy(arr []T, keyFn KeyFunc[T,K]) map[K][]T` | 分组 |
| `ToMap(arr []T, keyFn KeyFunc[T,K]) map[K]T` | 转换为映射 |

### 集合方法

| 方法 | 说明 |
|------|------|
| `NewSet[T]()` | 创建新集合 |
| `NewSetFromSlice(slice []T)` | 从切片创建集合 |
| `Add(v T)` | 添加元素 |
| `Remove(v T)` | 删除元素 |
| `Contains(v T) bool` | 判断是否包含 |
| `Size() int` | 返回大小 |
| `IsEmpty() bool` | 判断是否为空 |
| `ToSlice() []T` | 转换为切片 |
| `Union(other *Set[T]) *Set[T]` | 并集 |
| `Intersection(other *Set[T]) *Set[T]` | 交集 |
| `Difference(other *Set[T]) *Set[T]` | 差集 |
| `SymmetricDifference(other *Set[T]) *Set[T]` | 对称差集 |
| `Equal(other *Set[T]) bool` | 判断是否相等 |
| `IsSubset(other *Set[T]) bool` | 判断是否为子集 |
| `IsSuperset(other *Set[T]) bool` | 判断是否为超集 |
| `Clone() *Set[T]` | 克隆集合 |

## 测试

```bash
cd admin_server/component/marr
go test -v

# 基准测试
go test -bench=.
```

## 注意事项

1. **重复元素**：默认情况下会去除重复元素，统计信息中包含重复元素数量
2. **顺序控制**：大部分函数支持 `keepOrder` 参数控制是否保持原数组顺序
3. **性能**：使用 Set 实现，时间复杂度为 O(n)，适合大数据量
4. **泛型约束**：基础类型需要满足 `comparable` 约束
