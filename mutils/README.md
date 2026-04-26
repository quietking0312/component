# mutils - 实用工具组件

提供各种实用的小工具，目前包含高性能位集（BitSet）实现。

## 特性

- 🚀 **高性能位集**：使用 `uint64` 数组实现，适合大规模布尔值存储
- 💾 **内存高效**：1 亿个布尔值仅需约 12MB 内存
- ⚡ **快速查找**：支持快速查找第一个 false 位置

## 快速开始

### 位集使用

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mutils"
)

func main() {
    // 创建大小为 1000 的位集
    bs := mutils.NewBitSet64(1000)

    // 设置位
    bs.Set(10)
    bs.Set(20)
    bs.Set(100)

    // 获取位
    fmt.Println(bs.Get(10))  // true
    fmt.Println(bs.Get(11))  // false

    // 清除位
    bs.Clear(10)
    fmt.Println(bs.Get(10))  // false

    // 翻转位
    bs.Toggle(10)
    fmt.Println(bs.Get(10))  // true

    // 快速查找第一个 false
    idx := bs.FindFirstFalseFast()
    fmt.Printf("第一个未设置的位: %d\n", idx)
}
```

## API 文档

### BitSet64

| 函数/方法 | 说明 |
|-----------|------|
| `NewBitSet64(size int) *BitSet64` | 创建位集 |
| `(b *BitSet64) Set(n int)` | 将第 n 位设为 true |
| `(b *BitSet64) Clear(n int)` | 将第 n 位设为 false |
| `(b *BitSet64) Get(n int) bool` | 获取第 n 位的值 |
| `(b *BitSet64) Toggle(n int)` | 翻转第 n 位的值 |
| `(b *BitSet64) FindFirstFalseFast() int` | 快速查找第一个 false 的位置 |

## 适用场景

- 用户签到记录（按天索引）
- 权限位标记
- 大规模去重判断
- 布隆过滤器的底层存储

## 测试

```bash
cd mutils
go test -v
```
