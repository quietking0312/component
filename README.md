# Component

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue)](https://golang.org/doc/devel/release.html)
[![License](https://img.shields.io/badge/license-Apache%202.0-green)](LICENSE)

> 个人用 Go 工具组件库，对常用第三方包进行简易封装，追求**零内部依赖、高内聚、可插拔**的模块化设计。

## ✨ 设计哲学

- **完全独立**：31 个子模块零内部依赖，按需引入，不强迫消费不需要的代码
- **接口解耦**：日志、存储、编解码等关键能力均通过接口注入，不绑定具体实现
- **泛型优先**：充分利用 Go 1.18+ 泛型，在类型安全与性能之间取得平衡
- **开箱即用**：每个模块都有完整的 README、示例代码和测试用例

## 📦 模块一览

### 基础工具

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`marr`](./marr) | 数组/集合操作：交集、差集、并集、泛型集合 | 无 |
| [`mtool`](./mtool) | 数据结构工具箱：有序Map、跳表、FIFO、协程组、前缀树 | 无 |
| [`mutils`](./mutils) | 实用工具：BitSet 位集 | 无 |
| [`mmath`](./mmath) | 数学工具：快速逆平方根 | 无 |

### 日志与错误

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`mlog`](./mlog) | 结构化日志：基于 zap，支持文件切割与压缩 | `zap`, `lumberjack` |
| [`merr`](./merr) | 带错误码的错误体系：支持链式 Unwrap 和 errors.Is/As | 无 |

### 数据持久化

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`msql`](./msql) | 数据库连接与事务管理：支持 MySQL、SQLite | `sqlx`, `mysql`, `sqlite3` |
| [`mredis`](./mredis) | Redis 客户端：单机、Sentinel、Cluster 模式 | `go-redis/v9` |
| [`mcachedb`](./mcachedb) | 多级缓存 + 分布式事务：L1/L2/L3 三级架构 | `sqlx` |

### 网络与通信

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`mhttp`](./mhttp) | HTTP 客户端：链式 API、JSON 自动绑定、超时控制 | `x/net` |
| [`msock`](./msock) | 网络通信框架：TCP / WebSocket / KCP 三协议统一 | `uuid`, `websocket`, `kcp-go` |
| [`mrpc`](./mrpc) | gRPC 封装：服务端快速启动、客户端便捷连接 | `grpc`, `protobuf` |
| [`mssh`](./mssh) | SSH/SFTP 客户端：远程命令、文件传输、进度展示 | `sftp`, `x/crypto` |
| [`mpubsub`](./mpubsub) | 发布订阅框架：多通道、分组订阅、自动扩缩容 | 无 |

### 安全与认证

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`mjwt`](./mjwt) | JWT 生成与解析：支持自定义 Claims | 无 |
| [`mcyptos`](./mcyptos) | 加密哈希工具：AES、Base64、MD5/SHA/HMAC 系列 | 无 |
| [`m2fa`](./m2fa) | 双因素认证：TOTP 实现，兼容 Google Authenticator | 无 |

### 云存储

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`mstorage`](./mstorage) | 多云存储统一接口：阿里云 OSS、腾讯云 COS、AWS S3、百度 BOS、UFile | 各厂商 SDK |

### 系统与运维

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`mcron`](./mcron) | 定时任务调度：Cron 表达式、固定间隔、固定时间点、延迟执行 | `robfig/cron/v3` |
| [`msnowflake`](./msnowflake) | 雪花算法 ID 生成器：支持时钟回拨处理 | 无 |
| [`mtimewheel`](./mtimewheel) | 时间轮调度器：毫秒级延迟任务、分层时间轮 | 无 |
| [`mbar`](./mbar) | 命令行进度条：实时进度、耗时估算 | 无 |
| [`mconf`](./mconf) | 配置管理：基于 Viper，支持热更新、环境变量覆盖 | `viper`, `fsnotify` |

### 其他工具

| 模块 | 说明 | 外部依赖 |
|------|------|----------|
| [`memail`](./memail) | SMTP 邮件发送：连接池、异步发送 | 无 |
| [`mencoding`](./mencoding) | 编码检测与转换：自动识别 UTF-8/GBK 并转码 | 无 |
| [`mmen`](./mmen) | 系统内存与进程信息：支持 Windows/Linux | 无 |
| [`mstore`](./mstore) | 通用键值存储：线程安全、支持变更监听 Watch | 无 |
| [`mtime`](./mtime) | 时间处理：时间戳转换、时间冻结（测试利器） | 无 |
| [`mrand`](./mrand) | 随机数生成：自定义随机源、概率布尔保底机制 | 无 |
| [`middleware`](./middleware) | Gin/HTTP 中间件：CORS、Panic 恢复 | `gin` |
| [`mexcel`](./mexcel) | Excel 读写：结构体映射、流式处理 | `excelize/v2` |

## 🚀 快速开始

### 安装单个模块

```bash
go get github.com/quietking0312/component/mhttp
go get github.com/quietking0312/component/merr
go get github.com/quietking0312/component/msnowflake
```

### HTTP 请求示例

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mhttp"
)

func main() {
    resp, err := mhttp.Get("https://api.example.com/users").
        Query("page", "1").
        SetBearerToken("your-token").
        DoAndBindBytes()
    if err != nil {
        panic(err)
    }
    fmt.Println(string(resp))
}
```

### 带错误码的错误处理

```go
import "github.com/quietking0312/component/merr"

// 创建带码错误
err := merr.New("ERR_USER_NOT_FOUND", "用户 %d 不存在", 10086)

// 预定义错误
err := merr.InvalidParam("参数 id 不能为空")
err := merr.Timeout("请求超时")

// 判断错误码
if merr.IsCode(err, "ERR_USER_NOT_FOUND") {
    // 处理...
}
```

### 雪花 ID 生成

```go
import "github.com/quietking0312/component/msnowflake"

// 初始化全局生成器
msnowflake.InitDefault(1)

// 生成 ID
id := msnowflake.Generate()
```

### 接口注入日志（解耦示例）

```go
import (
    "github.com/quietking0312/component/mcron"
    "go.uber.org/zap"
)

// 使用 zap
logger := zap.L()
scheduler := mcron.New(mcron.WithLogger(logger))

// 或使用自定义 Logger
scheduler := mcron.New(mcron.WithLogger(myCustomLogger))
```

## 🏗️ 项目架构

```
component/
├── go.work              # Go 工作区，管理 31 个模块
├── marr/                # 数组/集合（泛型）
├── mbar/                # 进度条
├── mcachedb/            # 多级缓存 + 分布式事务
├── mconf/               # 配置管理
├── mcron/               # 定时任务
├── mcyptos/             # 加密哈希
├── memail/              # 邮件发送
├── mencoding/           # 编码转换
├── merr/                # 错误处理
├── mexcel/              # Excel 工具
├── mhttp/               # HTTP 客户端
├── middleware/           # Gin/HTTP 中间件
├── mjwt/                # JWT
├── mlog/                # 结构化日志
├── mmath/               # 数学工具
├── mmen/                # 内存/进程信息
├── mpubsub/             # 发布订阅
├── mrand/               # 随机数
├── mredis/              # Redis 客户端
├── mrpc/                # gRPC 封装
├── msnowflake/          # 雪花算法
├── msock/               # 网络通信框架
├── msql/                # SQL 数据库
├── mssh/                # SSH/SFTP
├── mstorage/            # 云存储统一接口
├── mstore/              # 通用键值存储
├── mtime/               # 时间处理
├── mtimewheel/          # 时间轮调度
├── mtool/               # 数据结构工具箱
└── mutils/              # 实用工具
```

### 架构特点

1. **完全独立**：每个目录都是独立 Go Module，拥有独立的 `go.mod`
2. **零内部依赖**：模块间不直接引用，需要协作时通过接口注入
3. **通用 Logger 接口**：`mhttp`、`mcron`、`middleware`、`msock`、`mtimewheel` 均定义了相同的 `Logger` 接口，使用者可统一适配

## 📊 项目统计

| 指标 | 数据 |
|------|------|
| 模块数量 | **31** |
| 总代码行数 | ~15,900 行 |
| 测试文件 | 41 个 |
| README 覆盖率 | **100%** |
| 零外部依赖模块 | 18 个（58%） |
| 定义接口的模块 | 16 个 |
| 使用泛型的模块 | 4 个 |

## 🧪 测试

```bash
# 测试单个模块
cd marr
go test -v

# 测试所有模块
go work sync
go test ./...
```

## 📄 许可证

[Apache License 2.0](LICENSE)

---

> 个人维护项目，欢迎学习参考。如有问题，欢迎提交 Issue 或 PR。
