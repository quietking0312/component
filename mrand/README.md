# mrand - 随机数生成组件

提供自定义随机源和概率随机布尔生成器，适用于游戏、抽奖等需要概率控制的场景。

## 特性

- 🎲 **自定义随机源**：支持指定种子的随机数生成器
- 🎯 **概率布尔**：为运气特差的情况提供保底机制
- 📊 **缓存优化**：随机源预缓存，减少锁竞争

## 快速开始

### 自定义随机源

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mrand"
)

func main() {
    source := mrand.NewSource(time.Now().UnixNano(), 1024)
    r := rand.New(source)

    fmt.Println(r.Intn(100))
}
```

### 概率布尔（保底机制）

```go
// 初始概率 10%，目标概率 50%，最大保底值 100
rb := mrand.NewRandBoolean(10, 50, 100)

// 每次返回 false 时，基础概率会增加，促使更快达成 true
if rb.attack() {
    fmt.Println("命中！")
} else {
    fmt.Println("未命中，概率已提升")
}
```

## API 文档

### 随机源

| 函数/方法 | 说明 |
|-----------|------|
| `NewSource(seed int64, cacheSize uint32) *MSource` | 创建自定义随机源 |
| `(m *MSource) Int63() int64` | 生成 int64 随机数 |
| `(m *MSource) Seed(seed int64)` | 设置种子 |

### 概率布尔

| 函数/方法 | 说明 |
|-----------|------|
| `NewRandBoolean(initial, target, max int64) *RandBoolean` | 创建概率布尔生成器 |
| `(r *RandBoolean) attack() bool` | 按概率返回布尔值，失败时增加概率 |

## 注意事项

- `MSource` 使用缓存机制减少锁竞争，适合高并发场景
- `RandBoolean` 的保底机制适合游戏抽卡等场景，避免玩家长期不中奖
- `MSource` 实现了 `rand.Source64` 接口，可直接用于 `rand.New()`

## 测试

```bash
cd mrand
go test -v
```
