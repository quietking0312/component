# mbar - 命令行进度条组件

终端进度条渲染工具，支持文件上传/下载等长时间任务的进度展示。

## 特性

- 📊 **实时进度显示**：百分比、已传输大小、总大小
- ⏱️ **耗时估算**：显示已用时间和预计剩余时间
- 🎨 **自定义宽度**：支持调整进度条显示宽度
- 🔄 **动态更新**：通过 `Add()` 方法增量更新进度

## 快速开始

### 基本用法

```go
package main

import (
    "github.com/quietking0312/component/mbar"
    "time"
)

func main() {
    total := 100
    bar := mbar.NewBar(total)

    for i := 0; i < total; i++ {
        // 模拟工作
        time.Sleep(50 * time.Millisecond)
        // 更新进度
        bar.Add(1)
    }
}
```

### 批量更新

```go
bar := mbar.NewBar(1000)

// 每次处理一批数据后更新
bar.Add(batchSize)
```

### 检查进度条状态

```go
bar := mbar.NewBar(100)
bar.Add(50)

// 判断进度条是否已完成
if bar.Closed() {
    fmt.Println("已完成")
}
```

## API 文档

### 类型

- `type Bar struct` - 进度条实例

### 函数

- `NewBar(total int) *Bar` - 创建进度条，total 为总任务量

### 方法

- `(b *Bar) Add(n ...int)` - 增加进度，默认增加 1
- `(b *Bar) Closed() bool` - 判断进度条是否已完成

## 测试

```bash
cd mbar
go test -v
```
