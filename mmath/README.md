# mmath - 数学工具组件

提供高性能数学运算工具，包括快速逆平方根等算法。

## 特性

- ⚡ **快速逆平方根**：基于经典算法的 `1/sqrt(x)` 快速计算
- 📐 **牛顿迭代平方根**：使用牛顿法求解平方根

## 快速开始

### 快速逆平方根

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mmath"
)

func main() {
    result := mmath.RSqrt(4.0)
    fmt.Println(result) // 约 0.5
}
```

### 牛顿法平方根

```go
result := mmath.Sqrt(2.0)
fmt.Println(result) // 约 1.4142135623730951
```

## API 文档

| 函数 | 说明 |
|------|------|
| `RSqrt(x float32) float32` | 快速计算 `1 / sqrt(x)` |
| `Sqrt(x float64) float64` | 牛顿迭代法计算平方根 |

## 注意事项

- `RSqrt` 使用 `float32`，精度有限，适合图形学等对性能敏感的场景
- `Sqrt` 使用 `float64`，精度更高，适合一般数学计算

## 测试

```bash
cd mmath
go test -v
```
