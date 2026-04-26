# mpubsub - 发布订阅组件

通用的消息发布订阅框架，支持多通道、分组订阅、自动扩缩容和消息序列化。

## 特性

- 📡 **多通道支持**：可同时监听多个消息通道
- 👥 **分组订阅**：支持按分组管理订阅者
- 🔄 **自动扩缩容**：根据消息负载自动调整工作者数量
- 💾 **多种序列化**：内置 Gob 序列化，可扩展其他格式
- 🛡️ **错误重试**：发布失败自动重试

## 快速开始

### 基础发布订阅

```go
package main

import (
    "context"
    "fmt"
    "github.com/quietking0312/component/mpubsub"
)

func main() {
    // 创建发布订阅实例
    // subFunc: 从消息中间件订阅的函数（如 Redis PubSub、NATS 等）
    ps := mpubsub.NewMPubSub[string](
        []string{"channel1", "channel2"},
        func(ctx context.Context, k string) (<-chan []byte, error) {
            // 返回消息通道
            return make(<-chan []byte), nil
        },
    )

    ps.Start()
}
```

### 使用 Gob 序列化

```go
parser := mpubsub.NewGobParser()

// 编码
msgBytes, err := parser.Encoder(message)

// 解码
err = parser.Decoder(msgBytes, &message)
```

### 分组订阅

```go
// 创建子组配置
sg := mpubsub.NewSubGroup("group1", group,
    mpubsub.WithSubGroupLogger[string](logger),
)
```

## API 文档

### 核心类型

| 类型 | 说明 |
|------|------|
| `MPubSub[T any]` | 发布订阅核心结构 |
| `SubGroup[T any]` | 订阅子组，支持自动扩缩容 |
| `GroupIface[T any]` | 订阅分组接口 |

### 核心函数

| 函数 | 说明 |
|------|------|
| `NewMPubSub[T any](channel []string, subFunc func(ctx context.Context, k string) (<-chan []byte, error), opts ...Option[T]) *MPubSub[T]` | 创建发布订阅实例 |
| `NewGobParser() *GobParser` | 创建 Gob 序列化器 |
| `NewSubGroup[T any](id string, group GroupIface[T], opts ...ChannelOption[T]) *SubGroup[T]` | 创建订阅子组 |

### 方法

| 方法 | 说明 |
|------|------|
| `(m *MPubSub[T]) Start()` | 启动订阅监听 |
| `(m *MPubSub[T]) Publish(msg Message[T]) error` | 发布消息 |
| `(m *MPubSub[T]) UnRegister(channelId string)` | 注销通道 |

## 测试

```bash
cd mpubsub
go test -v
```
