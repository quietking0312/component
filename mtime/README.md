# mtime - 时间处理组件

提供丰富的时间格式转换、时间戳操作和时间偏移工具，支持测试时的时间冻结功能。

## 特性

- 🔄 **时间戳转换**：秒/毫秒/微秒时间戳与字符串互转
- 📅 **时间计算**：周初计算、日期加减
- 🧊 **时间冻结**：测试时可冻结时间，所有 `Now()` 返回固定值
- ⏱️ **时间偏移**：设置全局时间偏移量

## 快速开始

### 时间戳与字符串转换

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mtime"
)

func main() {
    // 当前时间戳
    ts := mtime.NowTimestamp()
    fmt.Println(ts)

    // 时间戳转字符串
    str := mtime.TimestampToStr(ts, "2006-01-02 15:04:05")
    fmt.Println(str)

    // 字符串转时间戳
    ts2, err := mtime.StrToTimestamp("2024-01-01 00:00:00", "2006-01-02 15:04:05")
    if err != nil {
        panic(err)
    }
    fmt.Println(ts2)
}
```

### 毫秒/微秒时间戳

```go
// 毫秒时间戳
ms := mtime.NowTimestampMs()
msStr := mtime.TimestampMsToStr(ms, "2006-01-02 15:04:05")

// 微秒时间戳
us := mtime.NowTimestampUs()
```

### 时间计算

```go
// 获取当周开始时间（周一）
begin := mtime.BeginOfWeek(time.Now())

// 添加天数
future := mtime.AddDays(time.Now(), 7)

// 添加月数
nextMonth := mtime.AddMonths(time.Now(), 1)

// 添加年数
nextYear := mtime.AddYears(time.Now(), 1)

// 获取星期几（0=周日, 1=周一...）
weekday := mtime.GetWeek(time.Now())
```

### 测试时冻结时间

```go
// 冻结时间到指定时刻
mtime.Freeze(time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local))

// 之后所有 Now 相关调用都返回冻结的时间
fmt.Println(mtime.NowStr()) // 2024-01-01 00:00:00

// 设置时间偏移（在真实时间基础上偏移）
mtime.SetOffset(1 * time.Hour)
```

## API 文档

### 当前时间

| 函数 | 说明 |
|------|------|
| `NowTimestamp() int64` | 当前时间戳（秒） |
| `NowTimestampMs() int64` | 当前时间戳（毫秒） |
| `NowTimestampUs() int64` | 当前时间戳（微秒） |
| `NowTimestampNs() int64` | 当前时间戳（纳秒） |
| `NowStr() string` | 当前时间格式化字符串 |
| `NowDateStr() string` | 当前日期字符串（2006-01-02） |
| `NowDateTimeStr() string` | 当前日期时间字符串（2006-01-02 15:04:05） |

### 转换函数

| 函数 | 说明 |
|------|------|
| `TimestampToStr(timestamp int64, layout string) string` | 秒时间戳转字符串 |
| `TimestampMsToStr(timestampMs int64, layout string) string` | 毫秒时间戳转字符串 |
| `TimestampUsToStr(timestampUs int64, layout string) string` | 微秒时间戳转字符串 |
| `StrToTimestamp(timeStr string, layout string) (int64, error)` | 字符串转秒时间戳 |
| `StrToTimestampMs(timeStr string, layout string) (int64, error)` | 字符串转毫秒时间戳 |
| `StrToTimestampUs(timeStr string, layout string) (int64, error)` | 字符串转微秒时间戳 |
| `TimeToStr(t time.Time, layout string) string` | Time 转字符串 |
| `StrToTime(timeStr string, layout string) (time.Time, error)` | 字符串转本地时间 |
| `StrToTimeUTC(timeStr string, layout string) (time.Time, error)` | 字符串转 UTC 时间 |

### 时间计算

| 函数 | 说明 |
|------|------|
| `BeginOfWeek(t time.Time) time.Time` | 获取当周开始时间（周一） |
| `AddDays(t time.Time, days int) time.Time` | 添加天数 |
| `AddMonths(t time.Time, months int) time.Time` | 添加月数 |
| `AddYears(t time.Time, years int) time.Time` | 添加年数 |
| `GetWeek(t time.Time) int` | 获取星期几（0=周日） |

### 测试工具

| 函数 | 说明 |
|------|------|
| `Freeze(t time.Time)` | 冻结时间到指定时刻 |
| `SetOffset(offset time.Duration)` | 设置全局时间偏移量 |

## 测试

```bash
cd mtime
go test -v
```
