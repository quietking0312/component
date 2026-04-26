# msock - 统一网络通信组件

`msock` 是一个统一封装了 TCP、WebSocket、KCP 三种协议的高性能网络通信组件，支持自定义编解码、消息路由和日志输出。

## 特性

- **多协议支持**：TCP、WebSocket、KCP 统一接口
- **高性能**：基于事件驱动，支持高并发连接
- **消息路由**：灵活的路由机制，支持路由组和中间件
- **自定义编解码**：支持多种内置编解码器，可扩展自定义协议
- **连接管理**：内置连接管理器，支持广播和单播
- **中间件支持**：Recovery、Logging、Auth、RateLimit 等中间件
- **自定义日志**：可接入项目统一的日志系统

## 安装

```bash
# 添加依赖到 go.mod
go get github.com/gorilla/websocket
go get github.com/xtaci/kcp-go/v5
```

## 快速开始

### TCP 服务器

```go
package main

import (
    "log"
    " github.com/quietking0312/component/msock"
)

func main() {
    // 创建路由器
    router := msock.NewRouter()
    
    // 注册消息处理器
    router.Register(1, func(conn msock.Conn, msg msock.Message) {
        log.Printf("收到消息: %s", string(msg.Data()))
        
        // 回复消息
        reply := msock.NewMessage(2, []byte("收到"))
        conn.Send(reply)
    })
    
    // 创建服务器
    server, err := msock.NewServer(
        msock.WithAddress(":8080"),
        msock.WithConnType(msock.ConnTypeTCP),
        msock.WithCodec(msock.NewSimpleCodec()),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    server.SetRouter(router)
    
    // 启动服务器
    if err := server.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### TCP 客户端

```go
package main

import (
    "log"
    "github.com/quietking0312/component/msock"
)

func main() {
    // 创建路由器处理服务器消息
    router := msock.NewRouter()
    router.Register(2, func(conn msock.Conn, msg msock.Message) {
        log.Printf("服务器回复: %s", string(msg.Data()))
    })
    
    // 创建客户端
    client := msock.NewClient(
        msock.WithConnType(msock.ConnTypeTCP),
        msock.WithCodec(msock.NewSimpleCodec()),
    )
    client.SetRouter(router)
    
    // 连接服务器
    if err := client.Connect("localhost:8080"); err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 发送消息
    msg := msock.NewMessage(1, []byte("Hello"))
    client.Send(msg)
    
    // 阻塞等待
    select {}
}
```

## 协议选择

```go
// TCP
msock.WithConnType(msock.ConnTypeTCP)

// WebSocket
msock.WithConnType(msock.ConnTypeWebSocket)

// KCP (UDP可靠传输)
msock.WithConnType(msock.ConnTypeKCP)
```

## 编解码器

### 内置编解码器

| 编解码器 | 格式 | 适用场景 |
|---------|------|---------|
| SimpleCodec | [4字节长度][4字节路由ID][数据] | 二进制协议 |
| TLVCodec | [1字节类型][2字节长度][数据] | TLV协议 |
| LineCodec | 文本行（以\n分隔） | 文本协议 |
| JSONCodec | [4字节长度][JSON数据] | JSON协议 |

### 自定义编解码器

```go
type MyCodec struct{}

func (c *MyCodec) Encode(msg msock.Message) ([]byte, error) {
    // 实现编码逻辑
}

func (c *MyCodec) Decode(data []byte) (msock.Message, int, error) {
    // 实现解码逻辑，返回消息和已解码字节数
}

func (c *MyCodec) MaxPacketSize() int {
    return 1024 * 1024 // 1MB
}

// 使用自定义编解码器
server, _ := msock.NewServer(
    msock.WithCodec(&MyCodec{}),
)
```

## 路由系统

### 基础路由

```go
router := msock.NewRouter()

// 注册处理器
router.Register(1, handleLogin)
router.Register(2, handleLogout)

// 处理器函数
func handleLogin(conn msock.Conn, msg msock.Message) {
    // 处理登录逻辑
}
```

### 路由组

```go
// 用户模块 (前缀1)
userGroup := router.Group(1)
userGroup.Register(1, handleLogin)    // 路由ID = 0x10001
userGroup.Register(2, handleRegister) // 路由ID = 0x10002

// 聊天模块 (前缀2)
chatGroup := router.Group(2)
chatGroup.Register(1, handleSendMsg)  // 路由ID = 0x20001
```

### 中间件

```go
// 使用中间件
router.Use(msock.Recovery(logger))
router.Use(msock.Logging(logger))

// 自定义中间件
func MyMiddleware() msock.Middleware {
    return func(next msock.Handler) msock.Handler {
        return func(conn msock.Conn, msg msock.Message) {
            // 前置处理
            log.Println("before")
            
            next(conn, msg) // 调用下一个处理器
            
            // 后置处理
            log.Println("after")
        }
    }
}

router.Use(MyMiddleware())
```

### 内置中间件

| 中间件 | 功能 |
|-------|------|
| Recovery | 捕获 panic，防止服务崩溃 |
| Logging | 记录请求日志 |
| Auth | 认证检查 |
| Validate | 消息校验 |
| RateLimit | 限流控制 |
| Timeout | 超时控制 |

## 自定义日志

```go
// 实现 Logger 接口
type MyLogger struct{}

func (l *MyLogger) Debugf(format string, args ...interface{}) {
    log.Printf("[DEBUG] "+format, args...)
}
func (l *MyLogger) Infof(format string, args ...interface{}) {
    log.Printf("[INFO] "+format, args...)
}
func (l *MyLogger) Warnf(format string, args ...interface{}) {
    log.Printf("[WARN] "+format, args...)
}
func (l *MyLogger) Errorf(format string, args ...interface{}) {
    log.Printf("[ERROR] "+format, args...)
}

// 使用自定义日志
server, _ := msock.NewServer(
    msock.WithLogger(&MyLogger{}),
)
```

## 连接管理

```go
// 获取连接管理器
manager := server.GetConnManager()

// 获取连接数
count := manager.Count()

// 广播消息
msg := msock.NewMessage(100, []byte("广播消息"))
manager.Broadcast(msg)

// 获取指定连接
conn, ok := manager.Get(connID)
if ok {
    conn.Send(msg)
}

// 遍历所有连接
for _, conn := range manager.GetAll() {
    // ...
}
```

## 完整示例

### 游戏服务器示例

```go
package main

import (
    "log"
    "github.com/quietking0312/component/msock"
)

func main() {
    // 创建路由器
    router := msock.NewRouter()
    
    // 添加中间件
    router.Use(msock.Recovery(nil))
    router.Use(msock.Logging(nil))
    
    // 注册处理器
    gameGroup := router.Group(1)
    gameGroup.Register(1, handleMove)
    gameGroup.Register(2, handleAttack)
    gameGroup.Register(3, handleChat)
    
    // 创建服务器
    server, _ := msock.NewServer(
        msock.WithAddress(":8888"),
        msock.WithConnType(msock.ConnTypeTCP),
        msock.WithCodec(msock.NewSimpleCodec(1024*1024)),
        msock.WithMaxConnections(10000),
        msock.WithHeartbeat(30*time.Second, 90*time.Second),
    )
    
    server.SetRouter(router)
    
    // 连接事件
    server.OnConnect(func(conn msock.Conn) {
        log.Printf("玩家连接: %s", conn.ID())
        conn.SetValue("playerID", generatePlayerID())
    })
    
    server.OnDisconnect(func(conn msock.Conn) {
        log.Printf("玩家断开: %s", conn.ID())
    })
    
    // 启动
    log.Fatal(server.Run())
}

func handleMove(conn msock.Conn, msg msock.Message) {
    // 处理移动
}

func handleAttack(conn msock.Conn, msg msock.Message) {
    // 处理攻击
}

func handleChat(conn msock.Conn, msg msock.Message) {
    // 处理聊天
}
```

## 配置选项

| 选项 | 默认值 | 说明 |
|-----|-------|------|
| WithAddress | :8080 | 监听地址 |
| WithConnType | TCP | 连接类型 |
| WithCodec | SimpleCodec | 编解码器 |
| WithLogger | 空日志 | 日志器 |
| WithReadBufferSize | 4096 | 读缓冲区大小 |
| WithWriteBufferSize | 4096 | 写缓冲区大小 |
| WithMaxConnections | 10000 | 最大连接数 |
| WithReadTimeout | 60s | 读超时 |
| WithWriteTimeout | 10s | 写超时 |
| WithHeartbeat | 30s/90s | 心跳间隔/超时 |

## API 文档

### Server 方法

- `Run()` - 启动服务器
- `Stop()` - 停止服务器
- `SetRouter(router)` - 设置路由器
- `SetCodec(codec)` - 设置编解码器
- `SetLogger(logger)` - 设置日志器
- `OnConnect(fn)` - 设置连接回调
- `OnDisconnect(fn)` - 设置断开回调
- `OnError(fn)` - 设置错误回调
- `SendTo(connID, msg)` - 发送给指定连接
- `Broadcast(msg)` - 广播消息
- `ConnCount()` - 获取连接数

### Client 方法

- `Connect(addr)` - 连接服务器
- `Send(msg)` - 发送消息
- `SendBytes(data)` - 发送字节
- `Close()` - 关闭连接
- `SetRouter(router)` - 设置路由器
- `OnConnect(fn)` - 设置连接回调
- `OnDisconnect(fn)` - 设置断开回调

## 注意事项

1. **WebSocket 路径**：默认处理 `/ws` 路径，可通过 HTTP 路由扩展
2. **KCP 加密**：如需使用加密，需要配置密钥
3. **消息大小**：默认最大包大小 64KB，可通过编解码器配置
4. **连接ID**：使用 UUID 生成唯一标识
5. **并发安全**：所有公开方法都是线程安全的
