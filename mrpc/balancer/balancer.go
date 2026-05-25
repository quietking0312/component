// Package balancer 提供负载均衡接口及轮询、随机、加权随机、最少连接四种实现。
package balancer

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/quietking0312/component/mrpc/registry"
)

// ErrNoAvailable 没有可用节点
var ErrNoAvailable = errors.New("balancer: no available instance")

// Balancer 负载均衡接口
type Balancer interface {
	// Pick 从节点列表中选出一个实例
	Pick(instances []registry.ServiceInfo) (registry.ServiceInfo, error)
}

// ========== 轮询 ==========

// RoundRobin 轮询负载均衡
type RoundRobin struct {
	counter atomic.Uint64
}

func NewRoundRobin() *RoundRobin { return &RoundRobin{} }

func (r *RoundRobin) Pick(instances []registry.ServiceInfo) (registry.ServiceInfo, error) {
	if len(instances) == 0 {
		return registry.ServiceInfo{}, ErrNoAvailable
	}
	idx := r.counter.Add(1) - 1
	return instances[idx%uint64(len(instances))], nil
}

// ========== 随机 ==========

// Random 随机负载均衡
type Random struct{}

func NewRandom() *Random { return &Random{} }

func (r *Random) Pick(instances []registry.ServiceInfo) (registry.ServiceInfo, error) {
	if len(instances) == 0 {
		return registry.ServiceInfo{}, ErrNoAvailable
	}
	return instances[rand.Intn(len(instances))], nil
}

// ========== 加权随机 ==========

// WeightedRandom 加权随机负载均衡，权重由 ServiceInfo.Weight 决定
type WeightedRandom struct{}

func NewWeightedRandom() *WeightedRandom { return &WeightedRandom{} }

func (w *WeightedRandom) Pick(instances []registry.ServiceInfo) (registry.ServiceInfo, error) {
	if len(instances) == 0 {
		return registry.ServiceInfo{}, ErrNoAvailable
	}
	total := 0
	for _, s := range instances {
		wt := s.Weight
		if wt <= 0 {
			wt = 1
		}
		total += wt
	}
	n := rand.Intn(total)
	for _, s := range instances {
		wt := s.Weight
		if wt <= 0 {
			wt = 1
		}
		n -= wt
		if n < 0 {
			return s, nil
		}
	}
	return instances[len(instances)-1], nil
}

// ========== 最少连接 ==========

// LeastConn 最少活跃连接数负载均衡
type LeastConn struct {
	mu      sync.Mutex
	counter map[string]int // addr -> active conns
}

func NewLeastConn() *LeastConn {
	return &LeastConn{counter: make(map[string]int)}
}

func (l *LeastConn) Pick(instances []registry.ServiceInfo) (registry.ServiceInfo, error) {
	if len(instances) == 0 {
		return registry.ServiceInfo{}, ErrNoAvailable
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	best := instances[0]
	bestCnt := l.counter[best.Addr]
	for _, s := range instances[1:] {
		if c := l.counter[s.Addr]; c < bestCnt {
			best = s
			bestCnt = c
		}
	}
	l.counter[best.Addr]++
	return best, nil
}

// Done 请求结束时调用，减少活跃计数。
func (l *LeastConn) Done(addr string) {
	l.mu.Lock()
	if l.counter[addr] > 0 {
		l.counter[addr]--
	}
	l.mu.Unlock()
}
