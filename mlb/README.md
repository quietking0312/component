# mlb - 网关负载均衡组件

基于一致性哈希 + 负载感知的逻辑服务器负载均衡器，专为网关场景设计。

## 特性

- 🎯 **一致性哈希**：逻辑服上下线时，仅影响哈希环上相邻区域的用户，重新分配数量最少
- 🔄 **虚拟节点**：保证重新分配的用户均匀分散到剩余节点
- ⚖️ **负载感知**：老用户下线后节点负载降低，新用户分配时自动向负载低的节点倾斜
- 📊 **权重支持**：不同性能的服务器可设置不同权重，高性能节点承接更多用户
- 🛡️ **过载保护**：节点接近满载时自动将新用户导向其他节点

## 核心设计

### 一致性哈希 + 虚拟节点

每个物理节点根据权重生成大量虚拟节点，均匀分布在哈希环上：

```
虚拟节点数 = 节点权重 × vnodeFactor（默认 200）
```

用户通过 `userID` 的哈希值定位到环上的虚拟节点，再映射到物理节点。这样：
- 节点上线：新增虚拟节点，只有环上相邻区域的用户会迁移过来
- 节点下线：移除虚拟节点，只有该节点上的用户需要重新分配

### 负载感知分配

分配算法不是简单的一致性哈希，而是结合了负载感知：

1. 计算 `userID` 的哈希值，在环上找到落点
2. 从落点开始收集最多 `pickFactor` 个不同物理节点作为候选
3. 在候选中按**有效权重**做确定性加权选择：
   - 有效权重 = 基础权重 × (1 - 负载率²)
   - 负载越低，被选中的概率越高
4. 如果候选全部过载，扩大范围寻找健康节点

### 老用户下线的影响

当老用户从某逻辑服断开连接时：

```go
balancer.Unbind("node-1")  // 减少 node-1 的活跃计数
```

这会立即降低该节点的负载率，提升其有效权重。后续新用户分配时，此节点会有更高概率被选中，从而自然达到负载均衡。

## 快速开始

### 基础用法

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mlb"
)

func main() {
    // 创建负载均衡器
    balancer := mlb.NewBalancer()

    // 注册逻辑服务器节点
    // 参数: ID, 地址, 权重(性能系数), 最大承载用户数
    n1 := mlb.NewNode("logic-1", "10.0.0.1:8001", 1, 10000)
    n2 := mlb.NewNode("logic-2", "10.0.0.2:8002", 1, 10000)
    n3 := mlb.NewNode("logic-3", "10.0.0.3:8003", 2, 20000) // 高性能节点，权重为2

    balancer.Register(n1)
    balancer.Register(n2)
    balancer.Register(n3)

    // 节点上线
    balancer.Online("logic-1")
    balancer.Online("logic-2")
    balancer.Online("logic-3")

    // 为新用户分配逻辑服
    node, err := balancer.PickAndBind("user-10086")
    if err != nil {
        panic(err)
    }
    fmt.Println("用户分配到:", node.Addr)
}
```

### 用户断开连接

```go
// 用户下线时，释放节点上的活跃计数
balancer.Unbind("logic-1")
```

### 节点上下线

```go
// 逻辑服维护/重启时下线
balancer.Offline("logic-2")

// 维护完成后重新上线
balancer.Online("logic-2")
```

### 获取负载统计

```go
stats := balancer.GetStats()
fmt.Printf("总节点: %d, 在线: %d, 总活跃用户: %d, 平均负载: %.2f%%\n",
    stats.TotalNodes, stats.OnlineNodes, stats.TotalActive, stats.AvgLoadRate*100)
```

## 进阶配置

```go
// 自定义虚拟节点因子和候选节点数
balancer := mlb.NewBalancer(
    mlb.WithVnodeFactor(500),  // 每单位权重 500 个虚拟节点，分布更均匀
    mlb.WithPickFactor(5),     // 负载感知时考察 5 个候选节点，负载更均衡
)
```

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `vnodeFactor` | 200 | 每单位权重的虚拟节点数。越大分布越均匀，但内存开销越大 |
| `pickFactor` | 3 | 负载感知候选节点数。越大负载越均衡，但一致性越弱 |

## 使用建议

1. **权重设置**：建议根据服务器性能（CPU/内存）设置权重，如普通机器=1，高配机器=2~4
2. **MaxLoad 设置**：根据实际压测结果设置，建议预留 5% 缓冲（设置值为实际承载的 95%）
3. **网关层配合**：
   - 首次分配用户时调用 `PickAndBind`
   - 在网关层维护 `userID -> nodeID` 的映射关系
   - 用户断连时调用 `Unbind`
   - 节点下线时，网关层将受影响用户批量重连并重新 `Pick`
4. **pickFactor 调整**：
   - 追求强一致性（同一用户尽量固定节点）：保持默认 3
   - 追求负载均衡：可增大到 5~10

## API 文档

### 函数

| 函数 | 说明 |
|------|------|
| `NewBalancer(opts ...Option) *Balancer` | 创建负载均衡器 |
| `NewNode(id, addr string, weight int, maxLoad int64) *Node` | 创建节点 |

### Balancer 方法

| 方法 | 说明 |
|------|------|
| `Register(node *Node)` | 注册节点 |
| `Unregister(nodeID string)` | 注销节点 |
| `Online(nodeID string) error` | 节点上线 |
| `Offline(nodeID string) error` | 节点下线 |
| `Pick(userID string) (*Node, error)` | 为用户选择节点 |
| `PickAndBind(userID string) (*Node, error)` | 选择节点并增加活跃计数 |
| `Unbind(nodeID string) error` | 减少节点活跃计数 |
| `GetNode(nodeID string) (*Node, bool)` | 获取节点信息 |
| `Nodes() []*Node` | 获取所有注册节点 |
| `OnlineNodes() []*Node` | 获取所有在线节点 |
| `GetStats() Stats` | 获取负载统计 |

### Node 方法

| 方法 | 说明 |
|------|------|
| `ActiveCount() int64` | 当前活跃用户数 |
| `LoadRate() float64` | 当前负载率 0.0~1.0 |
| `IsOverloaded() bool` | 是否过载（>=95%） |
| `IsHealthy() bool` | 是否健康（在线且未过载） |
| `EffectiveWeight() float64` | 有效权重（考虑负载后的权重） |

## 测试

```bash
cd mlb
go test -v

# 基准测试
go test -bench=.
```
