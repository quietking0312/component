package mlb

import (
	"errors"
	"sort"
	"strconv"
	"sync"
)

var (
	ErrNoAvailableNode = errors.New("mlb: no available node")
	ErrNodeNotFound    = errors.New("mlb: node not found")
)

// Balancer 负载均衡器
// 基于一致性哈希 + 负载感知的网关用户分配器
//
// 特性:
//   - 逻辑服上下线时，仅影响哈希环上相邻区域的用户，重新分配数量最少
//   - 虚拟节点保证重新分配的用户均匀分散到剩余节点
//   - 老用户下线后节点负载降低，新用户分配时会倾向选择负载低的节点
//   - 支持权重差异（不同性能的服务器可设置不同权重）
type Balancer struct {
	mu          sync.RWMutex
	nodes       map[string]*Node  // 所有已注册节点
	onlineIDs   map[string]bool   // 在线节点ID集合
	hashRing    []uint64          // 排序后的虚拟节点哈希环
	hashMap     map[uint64]string // 虚拟节点哈希 -> 节点ID
	vnodeFactor int               // 每单位权重的虚拟节点数
	pickFactor  int               // 负载感知候选节点数
}

// Option 配置选项
type Option func(*Balancer)

// FilterFunc 节点过滤函数，返回 true 表示该节点可被选中。
type FilterFunc func(*Node) bool

// PickOption 为 Pick / PickAndBind 提供单次调用的选项。
type PickOption func(*pickConfig)

type pickConfig struct {
	filter FilterFunc
}

// WithFilter 设置 Pick 时的节点过滤条件。
// 只有 filter(n) == true 的节点才会参与本次选择，常用于按版本、区域等标签筛选。
//
// 示例：只选择版本为 "v2" 的节点
//
//	node, err := bl.Pick(userID, mlb.WithFilter(func(n *mlb.Node) bool {
//	    return n.HasTag("version", "v2")
//	}))
func WithFilter(filter FilterFunc) PickOption {
	return func(c *pickConfig) {
		c.filter = filter
	}
}

// WithVnodeFactor 设置每单位权重的虚拟节点数（默认 200）
// 虚拟节点数 = 节点权重 * vnodeFactor
// 值越大分布越均匀，但内存和计算开销越大
func WithVnodeFactor(factor int) Option {
	return func(b *Balancer) {
		if factor > 0 {
			b.vnodeFactor = factor
		}
	}
}

// WithPickFactor 设置负载感知时的候选节点数量（默认 3）
// 分配时会从 hash 落点开始的连续虚拟节点中，提取最多 pickFactor 个不同的物理节点作为候选，
// 然后选择其中负载最低的节点。
// 这样既能保证一致性哈希的稳定性，又能兼顾负载均衡。
func WithPickFactor(factor int) Option {
	return func(b *Balancer) {
		if factor > 0 {
			b.pickFactor = factor
		}
	}
}

// NewBalancer 创建负载均衡器
func NewBalancer(opts ...Option) *Balancer {
	b := &Balancer{
		nodes:       make(map[string]*Node),
		onlineIDs:   make(map[string]bool),
		hashMap:     make(map[uint64]string),
		vnodeFactor: 200,
		pickFactor:  3,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Register 注册节点（仅注册，不上线）
func (b *Balancer) Register(node *Node) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes[node.ID] = node
}

// Unregister 注销节点
func (b *Balancer) Unregister(nodeID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if node, ok := b.nodes[nodeID]; ok {
		node.SetOnline(false)
	}
	delete(b.nodes, nodeID)
	delete(b.onlineIDs, nodeID)
	b.rebuildRing()
}

// Online 节点上线
func (b *Balancer) Online(nodeID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	node, ok := b.nodes[nodeID]
	if !ok {
		return ErrNodeNotFound
	}
	node.SetOnline(true)
	b.onlineIDs[nodeID] = true
	b.rebuildRing()
	return nil
}

// Offline 节点下线
// 下线后该节点上的用户需要重新分配，但其余用户不受影响
func (b *Balancer) Offline(nodeID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	node, ok := b.nodes[nodeID]
	if !ok {
		return ErrNodeNotFound
	}
	node.SetOnline(false)
	delete(b.onlineIDs, nodeID)
	b.rebuildRing()
	return nil
}

// GetNode 获取节点信息
func (b *Balancer) GetNode(nodeID string) (*Node, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	n, ok := b.nodes[nodeID]
	return n, ok
}

// Nodes 返回所有已注册节点（副本）
func (b *Balancer) Nodes() []*Node {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]*Node, 0, len(b.nodes))
	for _, n := range b.nodes {
		result = append(result, n)
	}
	return result
}

// OnlineNodes 返回所有在线节点（副本）
func (b *Balancer) OnlineNodes() []*Node {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]*Node, 0, len(b.onlineIDs))
	for id := range b.onlineIDs {
		if n, ok := b.nodes[id]; ok {
			result = append(result, n)
		}
	}
	return result
}

// Pick 为用户选择一个逻辑服务器节点
// 如果用户已存在活跃连接，建议网关层自行维护映射关系；
// 本方法用于新用户首次分配或用户重连时的节点选择。
//
// 算法:
//  1. 计算 userID 的 hash 值
//  2. 在 hash 环上找到第一个 >= hash 的虚拟节点
//  3. 从该位置开始顺时针遍历，收集最多 pickFactor 个不同物理节点作为候选
//  4. 从候选中按负载加权选择：负载越低，被选中的概率越高
//     这样既保持了一致性哈希的局部稳定性，又能让新用户向负载低的节点倾斜
//  5. 若候选全部过载，扩大范围继续找第一个不过载的在线节点
//  6. 若所有在线节点都过载，返回负载最低的在线节点（兜底）
//
// 可通过 opts 传入 WithFilter 对节点进行业务层筛选，例如按版本标签筛选。
func (b *Balancer) Pick(userID string, opts ...PickOption) (*Node, error) {
	cfg := &pickConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.hashRing) == 0 {
		return nil, ErrNoAvailableNode
	}

	h := Hash(userID)
	idx := b.search(h)

	match := func(node *Node) bool {
		return node.IsOnline() && (cfg.filter == nil || cfg.filter(node))
	}

	// 收集候选节点（不同物理节点），并应用过滤条件
	candidates := make([]*Node, 0, b.pickFactor)
	seen := make(map[string]bool)
	n := len(b.hashRing)

	for i := 0; i < n && len(candidates) < b.pickFactor; i++ {
		vnodeIdx := (idx + i) % n
		nodeID := b.hashMap[b.hashRing[vnodeIdx]]
		if seen[nodeID] {
			continue
		}
		seen[nodeID] = true
		if node, ok := b.nodes[nodeID]; ok && match(node) {
			candidates = append(candidates, node)
		}
	}

	// 筛选未过载的候选
	healthy := make([]*Node, 0, len(candidates))
	for _, node := range candidates {
		if !node.IsOverloaded() {
			healthy = append(healthy, node)
		}
	}

	// 有未过载的候选，按有效权重加权随机选择
	// 负载越低 -> EffectiveWeight 越高 -> 被选中概率越大
	if len(healthy) > 0 {
		return weightedPick(h, healthy), nil
	}

	// 候选全部过载，在哈希环上顺时针找最近的未过载节点（仍受过滤条件约束）
	seen2 := make(map[string]bool)
	for i := 0; i < n; i++ {
		vnodeIdx := (idx + i) % n
		nodeID := b.hashMap[b.hashRing[vnodeIdx]]
		if seen2[nodeID] {
			continue
		}
		seen2[nodeID] = true
		if node, ok := b.nodes[nodeID]; ok && match(node) && !node.IsOverloaded() {
			return node, nil
		}
	}

	// 所有符合条件的在线节点均过载，从中选负载最低的兜底
	var best *Node
	for id := range b.onlineIDs {
		if node, ok := b.nodes[id]; ok && match(node) {
			if best == nil || node.LoadRate() < best.LoadRate() {
				best = node
			}
		}
	}
	if best != nil {
		return best, nil
	}

	return nil, ErrNoAvailableNode
}

// weightedPick 按有效权重加权随机选择一个节点
// hash 为用户的哈希值，用于确定性选择
func weightedPick(hash uint64, nodes []*Node) *Node {
	if len(nodes) == 1 {
		return nodes[0]
	}

	// 计算总权重
	var totalWeight float64
	weights := make([]float64, len(nodes))
	for i, node := range nodes {
		w := node.EffectiveWeight()
		weights[i] = w
		totalWeight += w
	}

	// 使用 userID 相关的方式做确定性加权
	// 为了保持一致性，这里用轮盘赌的方式，但基于 userID 的 hash 做确定性选择
	// 这样同一用户在没有节点变化时总是得到相同结果
	// 先把节点按 ID 排序使结果稳定
	type item struct {
		node   *Node
		weight float64
	}
	items := make([]item, len(nodes))
	for i := range nodes {
		items[i] = item{node: nodes[i], weight: weights[i]}
	}
	// 按节点 ID 排序，保证确定性
	sort.Slice(items, func(i, j int) bool {
		return items[i].node.ID < items[j].node.ID
	})

	// 重新计算排序后的累积权重
	var cum float64
	cumWeights := make([]float64, len(items))
	for i, it := range items {
		cum += it.weight
		cumWeights[i] = cum
	}

	// 用 hash 的低 53 位在 [0, totalWeight) 区间做确定性选择
	// float64 尾数 53 位，取低 53 位可保证精确除法
	pickVal := float64(hash&((1<<53)-1)) / float64(1<<53) * totalWeight
	if pickVal >= totalWeight {
		pickVal = totalWeight - 0.0001
	}

	for i, cw := range cumWeights {
		if pickVal < cw {
			return items[i].node
		}
	}
	return items[len(items)-1].node
}

// PickAndBind 选择节点并自动增加活跃计数。
//
// ⚠️ 注意：这不是原子操作。Pick 返回后 RLock 已释放，到 IncrActive 之间
// 节点可能被 Offline。如果对此敏感，建议先调用 Pick，确认节点仍在线后再调用 IncrActive。
func (b *Balancer) PickAndBind(userID string, opts ...PickOption) (*Node, error) {
	node, err := b.Pick(userID, opts...)
	if err != nil {
		return nil, err
	}
	node.IncrActive()
	return node, nil
}

// Unbind 用户断开连接时调用，减少节点活跃计数
// 这会提高该节点的有效权重，使新用户更倾向于分配到此节点
func (b *Balancer) Unbind(nodeID string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	node, ok := b.nodes[nodeID]
	if !ok {
		return ErrNodeNotFound
	}
	node.DecrActive()
	return nil
}

// search 在 hash 环上二分查找第一个 >= hash 的位置
func (b *Balancer) search(hash uint64) int {
	idx := sort.Search(len(b.hashRing), func(i int) bool {
		return b.hashRing[i] >= hash
	})
	if idx >= len(b.hashRing) {
		idx = 0
	}
	return idx
}

// rebuildRing 重建一致性哈希环
// 仅在写锁保护下调用
func (b *Balancer) rebuildRing() {
	b.hashRing = make([]uint64, 0)
	b.hashMap = make(map[uint64]string)

	for nodeID := range b.onlineIDs {
		node, ok := b.nodes[nodeID]
		if !ok {
			continue
		}
		// 虚拟节点数 = 权重 * 系数
		vnodeCount := node.weight * b.vnodeFactor
		for i := 0; i < vnodeCount; i++ {
			key := vnodeKey(nodeID, i)
			h := Hash(key)
			// 碰撞时换 key 重新 hash，保持分布均匀性（64 位碰撞概率极低）
			const maxCollisionRetries = 100
			for suffix := 1; suffix <= maxCollisionRetries; suffix++ {
				if _, exists := b.hashMap[h]; !exists {
					break
				}
				h = Hash(key + "#c" + strconv.Itoa(suffix))
			}
			b.hashRing = append(b.hashRing, h)
			b.hashMap[h] = nodeID
		}
	}

	sort.Slice(b.hashRing, func(i, j int) bool {
		return b.hashRing[i] < b.hashRing[j]
	})
}

// Stats 返回负载统计信息
type Stats struct {
	TotalNodes  int
	OnlineNodes int
	TotalActive int64
	AvgLoadRate float64
}

// Stats 返回当前负载统计
func (b *Balancer) GetStats() Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var totalActive int64
	var totalLoadRate float64
	onlineCount := 0

	for id := range b.onlineIDs {
		if node, ok := b.nodes[id]; ok {
			onlineCount++
			active := node.ActiveCount()
			totalActive += active
			totalLoadRate += node.LoadRate()
		}
	}

	avgLoadRate := 0.0
	if onlineCount > 0 {
		avgLoadRate = totalLoadRate / float64(onlineCount)
	}

	return Stats{
		TotalNodes:  len(b.nodes),
		OnlineNodes: onlineCount,
		TotalActive: totalActive,
		AvgLoadRate: avgLoadRate,
	}
}
