# mtimewheel - 时间轮调度组件

高性能延迟任务调度器，基于时间轮算法实现，适用于定时任务、延迟队列等场景。

## 特性

- ⏱️ **精确调度**：支持毫秒级延迟任务
- 🔄 **动态增删**：支持运行时添加和取消任务
- 📊 **任务统计**：实时获取任务数量和状态
- 🧵 **线程安全**：内部使用锁保护，支持并发操作
- 📝 **结构化日志**：基于 zap 记录调度日志

## 快速开始

### 默认时间轮

```go
package main

import (
    "context"
    "fmt"
    "github.com/quietking0312/component/mtimewheel"
    "time"
)

func main() {
    // 使用默认配置初始化（tick 间隔 100ms）
    tw := mtimewheel.InitDefault()

    // 添加延迟任务
    err := tw.AddTask("task1", 2*time.Second, func(ctx context.Context) error {
        fmt.Println("任务执行！")
        return nil
    })
    if err != nil {
        panic(err)
    }

    // 阻塞或执行其他逻辑
    time.Sleep(5 * time.Second)
}
```

### 自定义配置

```go
tw := mtimewheel.NewTimeWheel(
    mtimewheel.WithTick(50 * time.Millisecond), // 每 50ms  tick 一次
)

// 启动时间轮
tw.Start()
```

### 取消任务

```go
tw.AddTask("task2", 10*time.Second, func(ctx context.Context) error {
    fmt.Println("这个任务可能不会被触发")
    return nil
})

// 取消任务
tw.CancelTask("task2")
```

### 获取任务信息

```go
// 获取任务数量
count := tw.TaskCount()
fmt.Printf("当前任务数: %d\n", count)

// 获取指定任务
task := tw.GetTask("task1")
if task != nil {
    fmt.Printf("任务状态: %v\n", task.Status)
}
```

### 便捷函数

```go
// 类似于 time.AfterFunc
mtimewheel.After(3*time.Second, func(ctx context.Context) error {
    fmt.Println("3秒后执行")
    return nil
})
```

## API 文档

### 创建与配置

| 函数 | 说明 |
|------|------|
| `InitDefault() *TimeWheel` | 初始化默认时间轮 |
| `NewTimeWheel(opts ...Option) *TimeWheel` | 创建自定义时间轮 |
| `WithTick(tick time.Duration) Option` | 设置 tick 间隔 |

### 任务操作

| 方法 | 说明 |
|------|------|
| `(tw *TimeWheel) AddTask(id string, delay time.Duration, callback func(ctx context.Context) error) error` | 添加延迟任务 |
| `(tw *TimeWheel) CancelTask(id string) bool` | 取消任务 |
| `(tw *TimeWheel) GetTask(id string) *Task` | 获取任务信息 |
| `(tw *TimeWheel) TaskCount() int64` | 获取当前任务数量 |

### 便捷函数

| 函数 | 说明 |
|------|------|
| `After(delay time.Duration, callback func(ctx context.Context) error) error` | 延迟执行 |

### Task 结构

```go
type Task struct {
    ID       string
    Delay    time.Duration
    Callback func(ctx context.Context) error
    Status   TaskStatus
}
```

## 注意事项

1. **tick 间隔**：tick 间隔越小，调度精度越高，但 CPU 占用也越高
2. **任务唯一性**：任务 ID 必须唯一，重复添加会覆盖旧任务
3. **回调执行**：任务回调在独立 goroutine 中执行，避免阻塞时间轮
4. **停止时间轮**：需要调用 `Stop()` 方法优雅关闭，避免 goroutine 泄漏

## 测试

```bash
cd mtimewheel
go test -v
```
