# mcron - 定时任务调度组件

基于 `robfig/cron/v3` 封装的定时任务调度器，支持 Cron 表达式、固定间隔、固定时间点等多种调度方式。

## 特性

- ⏰ **多种调度方式**：Cron 表达式、固定间隔、固定时间点、延迟执行
- 📝 **结构化日志**：基于 zap 记录任务执行日志
- 🛡️ **Panic 恢复**：任务 panic 自动恢复，不影响其他任务
- 📊 **任务统计**：记录执行次数、错误次数、上次错误
- 🌍 **时区支持**：可自定义调度时区
- 🧵 **优雅关闭**：支持上下文取消，等待任务完成

## 快速开始

### 创建调度器

```go
package main

import (
    "context"
    "fmt"
    "github.com/quietking0312/component/mcron"
    "time"
)

func main() {
    // 创建调度器（默认本地时区）
    c := mcron.New()

    // 或自定义时区
    // c := mcron.New(mcron.WithLocation(time.UTC))

    // 启动调度器
    c.Start()
    defer c.Stop()

    // 添加 Cron 表达式任务
    _, err := c.AddFunc("每小时任务", mcron.Schedule{
        Cron: "0 0 * * * *", // 每秒 0 分 0 时执行
    }, func(ctx context.Context) error {
        fmt.Println("每小时执行一次")
        return nil
    })
    if err != nil {
        panic(err)
    }

    // 阻塞
    select {}
}
```

### 固定间隔任务

```go
_, err := c.AddFunc("间隔任务", mcron.Schedule{
    Interval: 5 * time.Minute, // 每 5 分钟执行一次
}, func(ctx context.Context) error {
    fmt.Println("每 5 分钟执行")
    return nil
})
```

### 延迟执行

```go
_, err := c.AddFunc("延迟任务", mcron.Schedule{
    Interval: 1 * time.Hour,
    Delay:    10 * time.Minute, // 首次执行延迟 10 分钟
}, func(ctx context.Context) error {
    fmt.Println("每小时执行，首次延迟 10 分钟")
    return nil
})
```

### 固定时间点

```go
_, err := c.AddFunc("定时任务", mcron.Schedule{
    At: "09:00:00", // 每天上午 9 点执行
}, func(ctx context.Context) error {
    fmt.Println("早上好！")
    return nil
})
```

### 使用 Job 接口

```go
type MyJob struct{}

func (j *MyJob) Name() string {
    return "my-job"
}

func (j *MyJob) Run(ctx context.Context) error {
    fmt.Println("执行任务逻辑")
    return nil
}

_, err := c.AddJob("自定义任务", mcron.Schedule{
    Cron: "*/30 * * * * *", // 每 30 秒
}, &MyJob{})
```

### 移除任务

```go
id, _ := c.AddFunc("临时任务", schedule, fn)

// 稍后移除
c.Remove(id)
```

### 查看任务状态

```go
// 获取单个任务
entry := c.GetEntry(id)
fmt.Printf("任务: %s, 下次执行: %v, 执行次数: %d, 错误次数: %d\n",
    entry.Name, entry.NextRun, entry.RunCount, entry.ErrorCount)

// 列出所有任务
entries := c.ListEntries()
for _, e := range entries {
    fmt.Println(e.Name)
}
```

## API 文档

### 创建调度器

| 函数 | 说明 |
|------|------|
| `New(opts ...Option) *Cron` | 创建调度器 |
| `WithLocation(loc *time.Location) Option` | 设置时区 |
| `WithLogger(logger *zap.Logger) Option` | 设置日志记录器 |

### 添加任务

| 方法 | 说明 |
|------|------|
| `(c *Cron) AddJob(name string, schedule Schedule, job Job) (cron.EntryID, error)` | 添加 Job 接口任务 |
| `(c *Cron) AddFunc(name string, schedule Schedule, fn JobFunc) (cron.EntryID, error)` | 添加函数任务 |

### 任务控制

| 方法 | 说明 |
|------|------|
| `(c *Cron) Start()` | 启动调度器 |
| `(c *Cron) Stop()` | 停止调度器（等待任务完成） |
| `(c *Cron) Remove(id cron.EntryID)` | 移除任务 |
| `(c *Cron) GetEntry(id cron.EntryID) *Entry` | 获取任务信息 |
| `(c *Cron) ListEntries() []*Entry` | 列出所有任务 |

### Schedule 结构

```go
type Schedule struct {
    Cron     string        // Cron 表达式（秒 分 时 日 月 周）
    Interval time.Duration // 固定间隔
    Delay    time.Duration // 首次执行延迟
    At       string        // 固定时间点（如 "09:00:00"）
}
```

### Job 接口

```go
type Job interface {
    Run(ctx context.Context) error
    Name() string
}
```

### Entry 结构

```go
type Entry struct {
    ID         cron.EntryID
    Name       string
    Schedule   Schedule
    Job        Job
    NextRun    time.Time
    PrevRun    time.Time
    RunCount   int64
    ErrorCount int64
    LastError  error
}
```

## Cron 表达式格式

本组件使用 `robfig/cron/v3` 的秒级表达式：

```
秒 分 时 日 月 周
```

| 字段 | 允许值 | 特殊字符 |
|------|--------|----------|
| 秒 | 0-59 | `* , - /` |
| 分 | 0-59 | `* , - /` |
| 时 | 0-23 | `* , - /` |
| 日 | 1-31 | `* , - / ?` |
| 月 | 1-12 | `* , - /` |
| 周 | 0-6（0=周日） | `* , - / ?` |

### 常用示例

| 表达式 | 说明 |
|--------|------|
| `0 */5 * * * *` | 每 5 分钟 |
| `0 0 9 * * 1` | 每周一上午 9 点 |
| `0 0 0 * * *` | 每天凌晨 |
| `0 0 */6 * * *` | 每 6 小时 |
| `0 0 9,18 * * *` | 每天 9 点和 18 点 |

## 测试

```bash
cd mcron
go test -v
```
