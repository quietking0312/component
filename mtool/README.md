# mtool - 数据结构工具箱

提供丰富的数据结构和通用工具函数，包括有序映射、跳表、索引映射、协程组等。

## 特性

- 🗺️ **有序映射**：保持插入顺序的泛型 Map
- 🏗️ **跳表**：高性能有序数据结构
- 🔍 **索引映射**：支持按索引和键访问的映射
- 🧵 **协程组**：可控并发数的任务执行组
- 📦 **FIFO 队列**：固定容量的先进先出队列
- 🔤 **前缀树**：敏感词过滤等字符串匹配场景
- 🪞 **结构体工具**：结构体拷贝、Tag 提取等

## 快速开始

### 有序映射

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mtool"
)

func main() {
    m := mtool.NewOrderedMap[string, int]()
    m.Set("a", 1)
    m.Set("b", 2)
    m.Set("c", 3)

    // 保持插入顺序
    fmt.Println(m.Keys())   // [a b c]
    fmt.Println(m.Values()) // [1 2 3]

    v, ok := m.Get("b")
    fmt.Println(v, ok) // 2 true
}
```

### 跳表

```go
sl := mtool.NewSkipList[int, string](func(a, b int) int {
    return a - b
})

sl.Insert(3, "three")
sl.Insert(1, "one")
sl.Insert(2, "two")

val, ok := sl.Search(2)
fmt.Println(val, ok) // two true
```

### 索引映射

```go
im := mtool.NewIndexMap[int, string](func(v string) int {
    return len(v)
})

im.Set("hello")
im.Set("world")

// 按键获取
val, ok := im.Get(5)

// 按索引获取
val, ok = im.GetByIndex(0)
```

### 协程组

```go
group := mtool.NewGoGroup(10) // 最多 10 个并发

group.SetLogger(mtool.DefaultPanicFunc)

for i := 0; i < 100; i++ {
    group.Run(i, func(data mtool.TaskData) error {
        fmt.Printf("处理任务: %d\n", data)
        return nil
    })
}
```

### FIFO 队列

```go
fifo := mtool.NewFIFO[int](100) // 容量 100

fifo.Push(1)
fifo.Push(2)
fifo.Push(3)

val, ok := fifo.Pop()
fmt.Println(val, ok) // 1 true

fmt.Println(fifo.Len()) // 2
```

### 结构体拷贝

```go
type Src struct {
    Name string
    Age  int
}

type Dst struct {
    Name string
    Age  int
}

src := Src{Name: "Alice", Age: 30}
var dst Dst

err := mtool.CopyStruct(&src, &dst)
if err != nil {
    panic(err)
}
fmt.Println(dst) // {Alice 30}
```

### 提取结构体 Tag

```go
type User struct {
    ID   int    `json:"id" db:"user_id"`
    Name string `json:"name" db:"user_name"`
}

// 提取 json tag
jsonMap := mtool.GetFieldsTagValueMap(&User{}, func(opt *mtool.Opt) {
    opt.TagKey = "json"
})
fmt.Println(jsonMap) // map[id:0 name:]

// 获取所有 tag 名称
tags := mtool.GetFieldTags(&User{}, func(opt *mtool.Opt) {
    opt.TagKey = "db"
})
fmt.Println(tags) // [user_id user_name]
```

### 前缀树（敏感词检测）

```go
tree := mtool.NewMapTree()
tree.AddWord("敏感词")
tree.AddWord("违规")

results := tree.Load("这是一条包含敏感词的内容")
for _, r := range results {
    fmt.Printf("发现敏感词: 位置 %d-%d\n", r[0], r[1])
}
```

## API 文档

### 有序映射

| 函数/方法 | 说明 |
|-----------|------|
| `NewOrderedMap[K comparable, V any]() *OrderedMap[K, V]` | 创建有序映射 |
| `Set(k K, v V)` | 设置键值 |
| `Get(k K) (V, bool)` | 获取值 |
| `Delete(k K)` | 删除键 |
| `Keys() []K` | 获取所有键（保持顺序） |
| `Values() []V` | 获取所有值（保持顺序） |
| `Clear()` | 清空 |
| `Update(fn func(dataCopy map[K]V, keysCopy []K) (map[K]V, []K))` | 批量更新 |

### 跳表

| 函数/方法 | 说明 |
|-----------|------|
| `NewSkipList[K comparable, V any](compare func(a, b K) int) *SkipList[K, V]` | 创建跳表 |
| `Insert(key K, value V)` | 插入 |
| `Search(key K) (V, bool)` | 查找 |
| `Remove(key K)` | 删除 |

### 索引映射

| 函数/方法 | 说明 |
|-----------|------|
| `NewIndexMap[K comparable, V any](getKey func(V) K) *IndexMap[K, V]` | 创建索引映射 |
| `Set(value V)` | 设置值 |
| `Get(key K) (V, bool)` | 按键获取 |
| `GetByIndex(idx int) (V, bool)` | 按索引获取 |
| `Delete(key K) bool` | 删除 |

### 协程组

| 函数/方法 | 说明 |
|-----------|------|
| `NewGoGroup(n int) *GoGroup` | 创建协程组，n 为最大并发数 |
| `Run(data TaskData, job Job) error` | 执行任务 |
| `SetLogger(fun PanicFunc)` | 设置 panic 处理函数 |

### FIFO

| 函数/方法 | 说明 |
|-----------|------|
| `NewFIFO[T any](capacity int) *FIFO[T]` | 创建 FIFO 队列 |
| `Push(value T)` | 入队 |
| `Pop() (T, bool)` | 出队 |
| `Len() int` | 获取长度 |

### 结构体工具

| 函数 | 说明 |
|------|------|
| `CopyStruct(src, dst any) error` | 结构体拷贝 |
| `CopyStruct2(src, dst any, opts ...Options) error` | 增强版结构体拷贝 |
| `GetFieldsTagValueMap(v any, opts ...func(opt *Opt)) map[string]any` | 提取结构体 tag 值 |
| `GetFieldTags(v any, opts ...func(opt *Opt)) []string` | 获取 tag 名称列表 |

### 通用函数

| 函数 | 说明 |
|------|------|
| `GetMapKeys[K comparable, V any](m map[K]V) []K` | 获取 Map 所有键 |
| `GetMapValues[K comparable, V any](m map[K]V) []V` | 获取 Map 所有值 |
| `IndexOf[T comparable](list []T, i T) int` | 查找元素索引 |
| `SliceSplit[T comparable](slice []T, n int) [][]T` | 切片分组 |

## 测试

```bash
cd mtool
go test -v
```
