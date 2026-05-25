// Package registry 定义服务注册与发现接口，并提供 etcd 实现。
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceInfo 服务实例信息
type ServiceInfo struct {
	// 服务名称
	Name string `json:"name"`
	// 实例地址，格式 host:port
	Addr string `json:"addr"`
	// 权重，用于加权负载均衡，默认 1
	Weight int `json:"weight"`
	// 扩展元数据
	Meta map[string]string `json:"meta,omitempty"`
}

// Registry 服务注册与发现接口
type Registry interface {
	// Register 注册服务实例，ctx 取消时自动注销
	Register(ctx context.Context, info ServiceInfo) error
	// Deregister 主动注销服务实例
	Deregister(ctx context.Context, info ServiceInfo) error
	// Discover 获取指定服务的所有实例快照
	Discover(ctx context.Context, name string) ([]ServiceInfo, error)
	// Watch 监听服务变更，返回的 channel 持续推送最新实例列表，ctx 取消时关闭
	Watch(ctx context.Context, name string) (<-chan []ServiceInfo, error)
	// Close 关闭注册中心连接
	Close() error
}

// ========== etcd 实现 ==========

// EtcdConfig etcd 注册中心配置
type EtcdConfig struct {
	// etcd 端点列表
	Endpoints []string
	// 连接超时，默认 5s
	DialTimeout time.Duration
	// 租约 TTL（秒），服务实例心跳周期，默认 10s
	LeaseTTL int64
	// 服务键前缀，默认 /mrpc/services
	Prefix string
}

func (c *EtcdConfig) fill() {
	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}
	if c.LeaseTTL <= 0 {
		c.LeaseTTL = 10
	}
	if c.Prefix == "" {
		c.Prefix = "/mrpc/services"
	}
}

// EtcdRegistry etcd 注册中心
type EtcdRegistry struct {
	cfg    EtcdConfig
	client *clientv3.Client
}

// NewEtcdRegistry 创建 etcd 注册中心。
func NewEtcdRegistry(cfg EtcdConfig) (*EtcdRegistry, error) {
	cfg.fill()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("registry: connect etcd: %w", err)
	}
	return &EtcdRegistry{cfg: cfg, client: cli}, nil
}

func (r *EtcdRegistry) key(info ServiceInfo) string {
	return fmt.Sprintf("%s/%s/%s", r.cfg.Prefix, info.Name, info.Addr)
}

func (r *EtcdRegistry) prefix(name string) string {
	return fmt.Sprintf("%s/%s/", r.cfg.Prefix, name)
}

// Register 注册服务实例，ctx 取消时自动续租停止并注销。
func (r *EtcdRegistry) Register(ctx context.Context, info ServiceInfo) error {
	if info.Weight <= 0 {
		info.Weight = 1
	}
	val, err := json.Marshal(info)
	if err != nil {
		return err
	}

	// 申请租约
	lease, err := r.client.Grant(ctx, r.cfg.LeaseTTL)
	if err != nil {
		return fmt.Errorf("registry: grant lease: %w", err)
	}

	// 写入键值
	if _, err = r.client.Put(ctx, r.key(info), string(val), clientv3.WithLease(lease.ID)); err != nil {
		return fmt.Errorf("registry: put key: %w", err)
	}

	// 自动续租，ctx 取消时停止
	keepCh, err := r.client.KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("registry: keepalive: %w", err)
	}

	go func() {
		for {
			select {
			case _, ok := <-keepCh:
				if !ok {
					return
				}
			case <-ctx.Done():
				// ctx 取消：主动注销
				dCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, _ = r.client.Revoke(dCtx, lease.ID)
				return
			}
		}
	}()

	return nil
}

// Deregister 主动注销服务实例。
func (r *EtcdRegistry) Deregister(ctx context.Context, info ServiceInfo) error {
	_, err := r.client.Delete(ctx, r.key(info))
	return err
}

// Discover 获取指定服务的所有实例快照。
func (r *EtcdRegistry) Discover(ctx context.Context, name string) ([]ServiceInfo, error) {
	resp, err := r.client.Get(ctx, r.prefix(name), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	return parseKVs(resp.Kvs), nil
}

// Watch 监听服务变更，推送最新实例列表。
func (r *EtcdRegistry) Watch(ctx context.Context, name string) (<-chan []ServiceInfo, error) {
	// 先拉一次快照
	snap, err := r.Discover(ctx, name)
	if err != nil {
		return nil, err
	}

	ch := make(chan []ServiceInfo, 1)
	ch <- snap

	go func() {
		defer close(ch)
		prefix := r.prefix(name)
		wch := r.client.Watch(ctx, prefix, clientv3.WithPrefix())
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-wch:
				if !ok {
					return
				}
				// 变更后重新拉取全量，避免处理增量复杂逻辑
				dCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				list, e := r.Discover(dCtx, name)
				cancel()
				if e == nil {
					select {
					case ch <- list:
					default:
						// 消费慢时丢弃旧值，写入新值
						select {
						case <-ch:
						default:
						}
						ch <- list
					}
				}
			}
		}
	}()

	return ch, nil
}

// Close 关闭 etcd 连接。
func (r *EtcdRegistry) Close() error {
	return r.client.Close()
}

// ========== 内存实现（测试 / 单机用）==========

// MemRegistry 基于内存的注册中心，适合单元测试和单机场景。
type MemRegistry struct {
	mu       sync.RWMutex
	services map[string][]ServiceInfo
	watchers map[string][]chan []ServiceInfo
}

// NewMemRegistry 创建内存注册中心。
func NewMemRegistry() *MemRegistry {
	return &MemRegistry{
		services: make(map[string][]ServiceInfo),
		watchers: make(map[string][]chan []ServiceInfo),
	}
}

func (m *MemRegistry) Register(_ context.Context, info ServiceInfo) error {
	if info.Weight <= 0 {
		info.Weight = 1
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.services[info.Name]
	for _, s := range list {
		if s.Addr == info.Addr {
			return nil
		}
	}
	m.services[info.Name] = append(list, info)
	m.notify(info.Name)
	return nil
}

func (m *MemRegistry) Deregister(_ context.Context, info ServiceInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.services[info.Name]
	out := list[:0]
	for _, s := range list {
		if s.Addr != info.Addr {
			out = append(out, s)
		}
	}
	m.services[info.Name] = out
	m.notify(info.Name)
	return nil
}

func (m *MemRegistry) Discover(_ context.Context, name string) ([]ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := m.services[name]
	cp := make([]ServiceInfo, len(list))
	copy(cp, list)
	return cp, nil
}

func (m *MemRegistry) Watch(ctx context.Context, name string) (<-chan []ServiceInfo, error) {
	ch := make(chan []ServiceInfo, 4)
	m.mu.Lock()
	// 推送初始快照
	list := m.services[name]
	cp := make([]ServiceInfo, len(list))
	copy(cp, list)
	ch <- cp
	m.watchers[name] = append(m.watchers[name], ch)
	m.mu.Unlock()

	go func() {
		<-ctx.Done()
		m.mu.Lock()
		ws := m.watchers[name]
		out := ws[:0]
		for _, w := range ws {
			if w != ch {
				out = append(out, w)
			}
		}
		m.watchers[name] = out
		m.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

func (m *MemRegistry) Close() error { return nil }

// notify 调用方持有写锁
func (m *MemRegistry) notify(name string) {
	list := m.services[name]
	cp := make([]ServiceInfo, len(list))
	copy(cp, list)
	for _, ch := range m.watchers[name] {
		select {
		case ch <- cp:
		default:
		}
	}
}

// ========== 工具 ==========

func parseKVs(kvs []*mvccpb.KeyValue) []ServiceInfo {
	out := make([]ServiceInfo, 0, len(kvs))
	for _, kv := range kvs {
		var info ServiceInfo
		if err := json.Unmarshal(kv.Value, &info); err == nil {
			out = append(out, info)
		}
	}
	return out
}
