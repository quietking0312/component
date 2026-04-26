# mrpc - gRPC 封装组件

简化 gRPC 服务端和客户端的创建与连接管理。

## 特性

- 🚀 **服务端快速启动**：一行代码启动 gRPC 服务
- 🔌 **客户端便捷连接**：封装 Dial 逻辑，支持自定义选项
- ⚙️ **灵活配置**：通过 `ServerOption` 自定义服务参数

## 快速开始

### 服务端

```go
package main

import (
    "github.com/quietking0312/component/mrpc"
    "google.golang.org/grpc"
)

func main() {
    opt := &mrpc.ServerOption{
        Network: "tcp",
        Address: ":50051",
    }

    err := mrpc.Serve(opt, func(s *grpc.Server) {
        // 注册你的 gRPC 服务
        // pb.RegisterYourServiceServer(s, &yourServer{})
    })
    if err != nil {
        panic(err)
    }
}
```

### 客户端

```go
conn, err := mrpc.Dial("localhost:50051")
if err != nil {
    panic(err)
}
defer conn.Close()

// 创建 gRPC 客户端
// client := pb.NewYourServiceClient(conn)
```

### 带自定义选项的客户端

```go
conn, err := mrpc.Dial("localhost:50051",
    grpc.WithInsecure(),
    grpc.WithBlock(),
)
```

## API 文档

| 函数 | 说明 |
|------|------|
| `Serve(opt *ServerOption, register func(s *grpc.Server)) error` | 启动 gRPC 服务端 |
| `Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)` | 连接 gRPC 服务端 |

### ServerOption

```go
type ServerOption struct {
    Network string // 网络类型，如 tcp
    Address string // 监听地址，如 :50051
}
```

## 测试

```bash
cd mrpc
go test -v
```
