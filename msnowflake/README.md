# msnowflake - 雪花算法ID生成器

高性能分布式唯一ID生成组件，基于雪花算法（Snowflake）实现。

## 特性

- **高性能**：单机每秒可生成数百万个唯一ID
- **分布式**：支持最多 1024 个机器节点
- **趋势递增**：ID按时间趋势递增，有利于数据库索引
- **高可用**：不依赖外部服务，本地生成
- **时钟回拨处理**：自动处理时钟回拨问题
- **线程安全**：支持高并发场景

## ID结构

生成的64位长整型ID结构如下：

```
| 1 bit |  41 bits   |  10 bits  |  12 bits  |
|-------|------------|-----------|-----------|
| sign  | timestamp  |  worker   | sequence  |
|  (0)  | (毫秒级)    |  (机器ID)  |  (序列号)  |
```

- **符号位**：1位，始终为0（保证为正数）
- **时间戳**：41位，可支持约69年（默认从2024-01-01开始）
- **机器ID**：10位，支持 0-1023 共1024个节点
- **序列号**：12位，每毫秒每节点可生成 4096 个ID

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "go_admin_element/admin_server/component/msnowflake"
)

func main() {
    // 创建生成器，指定机器ID
    g, err := msnowflake.NewGenerator(1)
    if err != nil {
        panic(err)
    }

    // 生成ID
    id := g.NextID()
    fmt.Println(id) // 例如：1234567890123456789
}
```

### 全局默认生成器

```go
// 初始化（程序启动时执行一次）
err := msnowflake.InitDefault(1)
if err != nil {
    panic(err)
}

// 之后可在任何地方使用
id := msnowflake.Generate()
```

### 批量生成

```go
g, _ := msnowflake.NewGenerator(1)

// 批量生成100个ID
ids := g.NextIDs(100)
// ids = []int64{...}
```

### 解析ID

```go
id := g.NextID()

// 解析ID获取组成部分
timestamp, workerID, sequence := msnowflake.ParseIDWithDefaultEpoch(id)

fmt.Printf("时间戳: %d\n", timestamp)
fmt.Printf("机器ID: %d\n", workerID)
fmt.Printf("序列号: %d\n", sequence)
```

### 自定义起始时间

```go
import "time"

// 自定义起始时间
customEpoch := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

g, err := msnowflake.NewGeneratorWithEpoch(1, customEpoch)
if err != nil {
    panic(err)
}
```

## 配置建议

### 机器ID分配

在分布式环境中，需要为每个服务实例分配唯一的机器ID：

```
服务器1: workerID = 1
服务器2: workerID = 2
服务器3: workerID = 3
...
```

机器ID可以通过以下方式分配：
- 环境变量
- 配置文件
- 服务注册中心
- 容器编排平台（如K8s的Pod ID）

### 时钟同步

为确保ID的唯一性和有序性，建议：
- 启用NTP时间同步
- 定期校准服务器时间
- 避免手动修改系统时间

## 性能

```
BenchmarkGenerator_NextID           100000000    10.2 ns/op    0 B/op    0 allocs/op
BenchmarkGenerator_NextID_Parallel   50000000    24.6 ns/op    0 B/op    0 allocs/op
```

- 单线程：约 9800万 ID/秒
- 多线程：约 4000万 ID/秒

## 注意事项

1. **机器ID唯一性**：不同实例必须使用不同的机器ID，否则可能生成重复ID
2. **时钟回拨**：组件会自动处理小幅度时钟回拨，但大幅度回拨可能导致等待
3. **起始时间**：一旦确定不要随意更改，否则可能生成重复ID

## API文档

### 类型

- `type Generator struct` - ID生成器

### 函数

- `NewGenerator(workerID int64) (*Generator, error)` - 创建生成器
- `NewGeneratorWithEpoch(workerID int64, epoch int64) (*Generator, error)` - 创建自定义起始时间的生成器
- `InitDefault(workerID int64) error` - 初始化全局默认生成器
- `InitDefaultWithEpoch(workerID int64, epoch int64) error` - 初始化自定义起始时间的全局生成器
- `Generate() int64` - 使用全局生成器生成ID
- `Generates(count int) []int64` - 使用全局生成器批量生成ID
- `ParseID(id int64, epoch int64) (timestamp, workerID, sequence int64)` - 解析ID
- `ParseIDWithDefaultEpoch(id int64) (timestamp, workerID, sequence int64)` - 使用默认起始时间解析ID

### 方法

- `(g *Generator) NextID() int64` - 生成下一个ID
- `(g *Generator) NextIDs(count int) []int64` - 批量生成ID
- `(g *Generator) GetWorkerID() int64` - 获取机器ID
- `(g *Generator) GetEpoch() int64` - 获取起始时间
