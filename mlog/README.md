# mlog - 日志组件

基于 `zap` 和 `lumberjack` 封装的高性能结构化日志库，支持文件切割、日志级别控制和多种输出格式。

## 特性

- 🚀 **高性能**：基于 zap，性能远超标准日志库
- 📁 **自动切割**：按大小自动切割日志文件，支持保留天数和压缩
- 🎨 **多格式**：支持 JSON 和 Console 两种输出格式
- 📺 **双输出**：可同时输出到文件和控制台
- 🏷️ **结构化**：支持字段化日志，便于日志收集和分析

## 快速开始

### 默认配置

```go
package main

import (
    "github.com/quietking0312/component/mlog"
    "go.uber.org/zap"
)

func main() {
    // 使用默认配置初始化
    cfg := mlog.DefaultConfig()
    mlog.Init(cfg)

    mlog.Info("服务启动", zap.String("addr", ":8080"))
    mlog.Error("连接失败", zap.Error(err))
}
```

### 自定义配置

```go
cfg := &mlog.Config{
    Level:      "debug",
    Format:     "json",
    Output:     "./logs/app.log",
    Console:    true,   // 同时输出到控制台
    MaxSize:    100,    // 单个文件最大 100MB
    MaxBackups: 10,     // 保留 10 个备份
    MaxAge:     30,     // 保留 30 天
    Compress:   true,   // 压缩旧文件
}
mlog.Init(cfg)
```

### 使用 Sugar Logger

```go
sugar := mlog.Sugar()
sugar.Infow("用户登录", "user_id", 123, "ip", "192.168.1.1")
```

### 带字段的 Logger

```go
logger := mlog.With(zap.String("service", "order"))
logger.Info("订单创建", zap.Int("order_id", 10001))
```

### 命名 Logger

```go
logger := mlog.Named("payment")
logger.Info("支付成功")
```

## API 文档

### 配置

| 函数 | 说明 |
|------|------|
| `DefaultConfig() *Config` | 返回默认配置 |
| `Init(cfg *Config)` | 初始化日志系统 |

### 日志级别

| 函数 | 说明 |
|------|------|
| `Debug(msg string, fields ...zap.Field)` | 调试日志 |
| `Info(msg string, fields ...zap.Field)` | 信息日志 |
| `Warn(msg string, fields ...zap.Field)` | 警告日志 |
| `Error(msg string, fields ...zap.Field)` | 错误日志 |
| `Fatal(msg string, fields ...zap.Field)` | 致命日志 |
| `Panic(msg string, fields ...zap.Field)` | 恐慌日志 |

### Logger 实例

| 函数 | 说明 |
|------|------|
| `Logger() *zap.Logger` | 获取原始 zap.Logger |
| `Sugar() *zap.SugaredLogger` | 获取 SugaredLogger |
| `With(fields ...zap.Field) *zap.Logger` | 创建带字段的 Logger |
| `WithOptions(opts ...zap.Option) *zap.Logger` | 创建带选项的 Logger |
| `Named(name string) *zap.Logger` | 创建命名 Logger |

### Config 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `Level` | `string` | 日志级别：debug/info/warn/error/fatal |
| `Format` | `string` | 格式：json/console |
| `Output` | `string` | 日志文件路径 |
| `Console` | `bool` | 是否同时输出到控制台 |
| `MaxSize` | `int` | 单个文件最大大小（MB） |
| `MaxBackups` | `int` | 保留旧文件最大个数 |
| `MaxAge` | `int` | 保留旧文件最大天数 |
| `Compress` | `bool` | 是否压缩旧文件 |

## 测试

```bash
cd mlog
go test -v
```
