# msock

统一封装 TCP / WebSocket / KCP 的网络通信库，提供一致的服务端与客户端 API、消息路由、中间件链和多种内置编解码器。

## 安装

```bash
go get github.com/quietking0312/component/msock
```

## 快速开始

### 服务端

```go
router := msock.NewRouter()
router.Register(1, func(conn msock.Conn, msg msock.Message) {
    fmt.Println("收到消息:", string(msg.Data()))
    conn.Send(msock.NewMessage(2, []byte("pong")))
})

server, err := msock.NewServer(
    msock.WithAddress(":8080"),
    msock.WithConnType(msock.ConnTypeTCP),
)
if err != nil {
    log.Fatal(err)
}
server.SetRouter(router)
log.Fatal(server.Run())
```

### 客户端

```go
router := msock.NewRouter()
router.Register(2, func(conn msock.Conn, msg msock.Message) {
    fmt.Println("收到回复:", string(msg.Data()))
})

client := msock.NewClient(
    msock.WithConnType(msock.ConnTypeTCP),
)
client.SetRouter(router)

if err := client.Connect(":8080"); err != nil {
    log.Fatal(err)
}
defer client.Close()
client.Send(msock.NewMessage(1, []byte("ping")))
```

## 支持的协议

| 常量 | 协议 | 说明 |
|------|------|------|
| `ConnTypeTCP` | TCP | 原生 TCP 长连接 |
| `ConnTypeWebSocket` | WebSocket | gorilla/websocket |
| `ConnTypeGWS` | WebSocket | lxzan/gws（高性能） |
| `ConnTypeKCP` | KCP | 基于 UDP 的可靠传输 |

协议切换只需修改 `WithConnType`，其余代码不变。WebSocket 服务端默认挂载 `/ws` 路径。

## 消息

`Message` 接口包含两个字段：`RouteID() uint32` 用于路由，`Data() []byte` 为消息体。

```go
msg := msock.NewMessage(1, []byte("hello"))
msg.RouteID() // 1
msg.Data()    // []byte("hello")
```

高并发场景可使用对象池减少 GC 压力：

```go
msg := msock.AcquireMessage()
msg.SetData([]byte("hello"))
// ...
msock.ReleaseMessage(msg)
```

## 编解码器

### 内置编解码器

**SimpleCodec**（默认）— 二进制协议

```
[4字节总长度 BE] [4字节RouteID BE] [Data]
```

```go
msock.NewSimpleCodec()           // 默认 64KB 限制
msock.NewSimpleCodec(128 * 1024) // 自定义最大包大小
```

**TLVCodec** — Type 即 RouteID，适合消息类型 ≤ 255 的场景

```
[1字节Type] [2字节Length BE] [Value]
```

```go
msock.NewTLVCodec()
```

**LineCodec** — 文本协议，RouteID 固定为 0

```
[文本内容]\n
```

```go
msock.NewLineCodec()
```

### 自定义编解码器

实现 `Codec` 接口即可：

```go
type Codec interface {
    Encode(msg Message) ([]byte, error)
    Decode(data []byte) (Message, int, error) // 返回消息和已消费字节数
    MaxPacketSize() int
}
```

## 路由

```go
router := msock.NewRouter()

// 注册处理器
router.Register(1, handleLogin)
router.Register(2, handleLogout)

// 批量注册
router.RegisterMultiple(map[uint32]msock.Handler{
    3: handleA,
    4: handleB,
})

// 移除
router.Remove(1)

// 自定义未匹配处理器
router.SetNotFoundHandler(func(conn msock.Conn, msg msock.Message) {
    fmt.Println("未知路由:", msg.RouteID())
})
```

## 中间件

中间件签名为 `func(next Handler) Handler`，通过 `router.Use()` 全局注册，按注册顺序形成洋葱模型。

```go
router.Use(msock.Recovery(logger), msock.Logging(logger))
```

### 内置中间件

```go
// panic 恢复，记录错误日志后继续运行
msock.Recovery(logger)

// 请求日志，记录 conn/route/size
msock.Logging(logger)

// 认证，返回 false 则中断后续处理
msock.Auth(func(conn msock.Conn) bool {
    _, ok := conn.GetValue("uid")
    return ok
}, logger)

// 消息校验
msock.Validate(func(msg msock.Message) error {
    if len(msg.Data()) == 0 {
        return errors.New("empty data")
    }
    return nil
}, logger)

// 限流（每连接计数器）
msock.RateLimit(100, logger)

// 超时（连接 context 取消时触发）
msock.Timeout(func() { fmt.Println("超时") }, logger)
```

### 自定义中间件

```go
func myMiddleware(next msock.Handler) msock.Handler {
    return func(conn msock.Conn, msg msock.Message) {
        // 前置逻辑
        next(conn, msg)
        // 后置逻辑
    }
}

router.Use(myMiddleware)
```

## 连接事件

```go
// 服务端
server.OnConnect(func(conn msock.Conn) {
    fmt.Println("连接建立:", conn.ID(), conn.Type())
})
server.OnDisconnect(func(conn msock.Conn) {
    fmt.Println("连接断开:", conn.ID())
})
server.OnError(func(conn msock.Conn, err error) {
    fmt.Println("连接错误:", err)
})

// 客户端
client.OnConnect(func(conn msock.Conn) { ... })
client.OnDisconnect(func(conn msock.Conn) { ... })
client.OnError(func(err error) { ... })
```

无论是对端主动断开、网络超时还是服务端调用 `conn.Close()`，`OnDisconnect` 都会触发。

## 连接对象

```go
conn.ID()           // 唯一标识（UUID）
conn.Type()         // ConnTypeTCP / ConnTypeWebSocket / ConnTypeGWS / ConnTypeKCP
conn.RemoteAddr()   // 远端地址
conn.Send(msg)      // 发送消息（异步入队）
conn.SendBytes(b)   // 发送原始字节
conn.Close()        // 关闭连接
conn.IsClosed()     // 是否已关闭
conn.Context()      // 连接的 context，断开时 Done()

// 跨 handler 传递数据
conn.SetValue("uid", 12345)
uid, ok := conn.GetValue("uid")
```

## 连接管理

```go
mgr := server.GetConnManager()
mgr.Count()          // 当前连接数
mgr.Get("conn-id")   // 获取单个连接
mgr.GetAll()         // 获取所有连接

server.SendTo("conn-id", msg) // 发送给指定连接
server.Broadcast(msg)         // 广播（编码一次，批量发送）
server.ConnCount()            // 当前连接数
```

## 服务端配置

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithAddress` | `:8080` | 监听地址 |
| `WithConnType` | `ConnTypeTCP` | 协议类型 |
| `WithCodec` | `SimpleCodec` | 编解码器 |
| `WithLogger` | 空日志 | 日志实现 |
| `WithMaxConnections` | `10000` | 最大连接数 |
| `WithReadBufferSize` | `4096` | 读缓冲区字节数 |
| `WithWriteBufferSize` | `4096` | 写缓冲区字节数 |
| `WithReadTimeout` | `60s` | 读超时 |
| `WithWriteTimeout` | `10s` | 写超时 |
| `WithHeartbeat` | `30s / 90s` | 心跳间隔 / 超时 |

## 日志

实现 `Logger` 接口对接任意日志库：

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}
```

内置提供基于标准库 `log` 的 `StdLogger`：

```go
msock.WithLogger(msock.NewStdLogger())
```
