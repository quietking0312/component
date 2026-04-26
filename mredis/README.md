# mredis - Redis 客户端组件

基于 `go-redis/v9` 封装的 Redis 客户端，支持单机、Sentinel 和集群模式。

## 特性

- 🔌 **多模式支持**：单机、Sentinel、Cluster
- ⚙️ **灵活配置**：通过 Option 模式配置连接池、超时等参数
- 🔐 **TLS 支持**：支持 TLS 加密连接
- 📡 **高可用**：内置 Sentinel 和集群故障转移

## 快速开始

### 单机模式

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mredis"
)

func main() {
    client, err := mredis.NewRedisClient(
        mredis.SetAddrs([]string{"localhost:6379"}),
        mredis.SetAuth("", "password"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()
    err = client.Set(ctx, "key", "value", 0).Err()
    if err != nil {
        panic(err)
    }

    val, err := client.Get(ctx, "key").Result()
    fmt.Println(val)
}
```

### 集群模式

```go
client, err := mredis.NewRedisClusterClient(
    mredis.SetAddrs([]string{"node1:6379", "node2:6379", "node3:6379"}),
    mredis.SetAuth("", "password"),
)
```

## API 文档

### 配置选项

| 函数 | 说明 |
|------|------|
| `SetAddrs(addrs []string) Option` | 设置 Redis 地址列表 |
| `SetAuth(username, password string) Option` | 设置认证信息 |
| `SetReadTimeout(readTimeout time.Duration) Option` | 设置读超时 |
| `SetWriteTimeout(writeTimeout time.Duration) Option` | 设置写超时 |
| `SetPoolSize(poolSize int) Option` | 设置连接池大小 |
| `SetMinIdleConns(minIdle int) Option` | 设置最小空闲连接数 |
| `SetTLSConfig(tlsConfig *tls.Config) Option` | 设置 TLS 配置 |

### 客户端创建

| 函数 | 说明 |
|------|------|
| `NewRedisClient(opts ...RedisOption) (*redis.Client, error)` | 创建单机客户端 |
| `NewRedisClusterClient(opts ...ClusterOption) (*redis.ClusterClient, error)` | 创建集群客户端 |

## 测试

```bash
cd mredis
go test -v
```

## 注意事项

1. 使用集群模式时，请确保所有节点地址都正确配置
2. 生产环境建议开启 TLS 和认证
3. 合理设置连接池大小，避免连接数过多
