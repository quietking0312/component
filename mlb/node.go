package mlb

import (
	"sync/atomic"
)

// Node 表示一个逻辑服务器节点
type Node struct {
	ID                string            // 节点唯一标识
	Addr              string            // 节点地址
	Tags              map[string]string // 节点标签，可用于版本、区域等业务筛选
	weight            int               // 基础权重（私有，防止外部并发写后 rebuildRing 未感知）
	maxLoad           int64             // 最大承载用户数（构造后不可变，无需 atomic）
	overloadThreshold float64           // 过载阈值，默认 0.95
	active            int64             // 当前活跃用户数
	online            atomic.Bool       // 是否在线（atomic，消除裸 bool 竞争）
}

// NodeOption 节点配置选项
type NodeOption func(*Node)

// WithOverloadThreshold 设置节点过载阈值（默认 0.95）
// 取值范围 (0, 1]；超出范围时忽略，保留默认值。
// 示例：maxLoad=100 时可设 0.9，maxLoad=10000 时可设 0.98。
func WithOverloadThreshold(threshold float64) NodeOption {
	return func(n *Node) {
		if threshold > 0 && threshold <= 1 {
			n.overloadThreshold = threshold
		}
	}
}

// WithTags 为节点设置标签，可用于 Pick 时的条件筛选。
// 传入的 tags 会被拷贝到节点中，避免外部后续修改影响节点状态。
func WithTags(tags map[string]string) NodeOption {
	return func(n *Node) {
		if len(tags) == 0 {
			return
		}
		n.Tags = make(map[string]string, len(tags))
		for k, v := range tags {
			n.Tags[k] = v
		}
	}
}

// NewNode 创建一个新节点，初始状态为离线，需调用 Balancer.Online 后才参与路由。
func NewNode(id, addr string, weight int, maxLoad int64, opts ...NodeOption) *Node {
	if weight <= 0 {
		weight = 1
	}
	if maxLoad <= 0 {
		maxLoad = 10000
	}
	n := &Node{
		ID:                id,
		Addr:              addr,
		weight:            weight,
		maxLoad:           maxLoad,
		overloadThreshold: 0.95,
		// online 零值为 false，与 Register 语义一致
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// GetWeight 返回节点基础权重
func (n *Node) GetWeight() int {
	return n.weight
}

// GetMaxLoad 返回节点最大承载用户数
func (n *Node) GetMaxLoad() int64 {
	return n.maxLoad
}

// GetOverloadThreshold 返回节点过载阈值
func (n *Node) GetOverloadThreshold() float64 {
	return n.overloadThreshold
}

// ActiveCount 返回当前活跃用户数
func (n *Node) ActiveCount() int64 {
	return atomic.LoadInt64(&n.active)
}

// LoadRate 返回当前负载率 0.0 ~ 1.0
func (n *Node) LoadRate() float64 {
	if n.maxLoad <= 0 {
		return 0
	}
	active := atomic.LoadInt64(&n.active)
	return float64(active) / float64(n.maxLoad)
}

// IsOverloaded 判断节点是否过载
func (n *Node) IsOverloaded() bool {
	return n.LoadRate() >= n.overloadThreshold
}

// IsHealthy 判断节点是否健康（在线且未过载）
func (n *Node) IsHealthy() bool {
	return n.IsOnline() && !n.IsOverloaded()
}

// IncrActive 增加活跃用户数
func (n *Node) IncrActive() {
	atomic.AddInt64(&n.active, 1)
}

// DecrActive 减少活跃用户数，不会降到 0 以下
func (n *Node) DecrActive() {
	for {
		old := atomic.LoadInt64(&n.active)
		if old <= 0 {
			return
		}
		if atomic.CompareAndSwapInt64(&n.active, old, old-1) {
			return
		}
	}
}

// SetOnline 设置节点在线状态
func (n *Node) SetOnline(online bool) {
	n.online.Store(online)
}

// IsOnline 返回节点是否在线
func (n *Node) IsOnline() bool {
	return n.online.Load()
}

// Tag 返回指定标签的值，不存在时 ok 为 false。
func (n *Node) Tag(key string) (string, bool) {
	if n == nil || n.Tags == nil {
		return "", false
	}
	v, ok := n.Tags[key]
	return v, ok
}

// HasTag 判断节点是否包含指定键值对的标签。
func (n *Node) HasTag(key, value string) bool {
	v, ok := n.Tag(key)
	return ok && v == value
}

// EffectiveWeight 计算有效权重
// 公式: weight * (1 - loadRate²)，负载越低权重越高
func (n *Node) EffectiveWeight() float64 {
	loadRate := n.LoadRate()
	if loadRate >= 1.0 {
		return 0
	}
	return float64(n.weight) * (1 - loadRate*loadRate)
}
