package gorpc

import (
	"context"
	"net/rpc"
	"sync"

	"github.com/quietking0312/component/mrpc/balancer"
	"github.com/quietking0312/component/mrpc/middleware"
	"github.com/quietking0312/component/mrpc/registry"
)

// ========== 带中间件的 Client ==========

// MiddlewareClient 封装 net/rpc Client，支持服务发现、负载均衡、熔断、限流。
type MiddlewareClient struct {
	cfg      *ClientConfig
	cb       *middleware.CircuitBreaker
	rl       *middleware.RateLimiter
	reg      registry.Registry
	svcName  string
	balancer balancer.Balancer

	mu     sync.Mutex
	client *rpc.Client
	addr   string // 当前连接的地址
}

// MiddlewareClientConfig 中间件客户端配置
type MiddlewareClientConfig struct {
	// 基础连接配置
	ClientConfig
	// 服务发现（可选，不设则直连 Target）
	Registry    registry.Registry
	ServiceName string
	Balancer    balancer.Balancer
	// 熔断配置（可选，nil 表示不启用）
	CircuitBreaker *middleware.CircuitConfig
	// 限流配置（可选，nil 表示不启用）
	RateLimiter *middleware.RateLimiterConfig
}

// NewMiddlewareClient 创建带中间件的客户端。
func NewMiddlewareClient(ctx context.Context, cfg *MiddlewareClientConfig) (*MiddlewareClient, error) {
	c := &MiddlewareClient{
		cfg:      &cfg.ClientConfig,
		reg:      cfg.Registry,
		svcName:  cfg.ServiceName,
		balancer: cfg.Balancer,
	}
	if cfg.CircuitBreaker != nil {
		c.cb = middleware.NewCircuitBreaker(*cfg.CircuitBreaker)
	}
	if cfg.RateLimiter != nil {
		c.rl = middleware.NewRateLimiter(*cfg.RateLimiter)
	}
	if c.balancer == nil {
		c.balancer = balancer.NewRoundRobin()
	}

	if err := c.connect(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// connect 建立连接（服务发现选址 or 直连）
func (c *MiddlewareClient) connect(ctx context.Context) error {
	addr := c.cfg.Target
	if c.reg != nil && c.svcName != "" {
		instances, err := c.reg.Discover(ctx, c.svcName)
		if err != nil {
			return err
		}
		info, err := c.balancer.Pick(instances)
		if err != nil {
			return err
		}
		addr = info.Addr
	}

	nc, err := NewClient(&ClientConfig{
		Target:      addr,
		Codec:       c.cfg.Codec,
		DialTimeout: c.cfg.DialTimeout,
	})
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.client != nil {
		_ = c.client.Close()
	}
	c.client = nc.client
	c.addr = addr
	c.mu.Unlock()
	return nil
}

// Call 同步调用，经过熔断和限流检查。
func (c *MiddlewareClient) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	// 限流
	if c.rl != nil && !c.rl.Allow() {
		return middleware.ErrRateLimited
	}

	// 熔断
	var cbDone func(error)
	if c.cb != nil {
		done, ok := c.cb.Allow()
		if !ok {
			return middleware.ErrCircuitOpen
		}
		cbDone = done
	}

	c.mu.Lock()
	cli := c.client
	c.mu.Unlock()

	// 支持 context 取消
	type result struct {
		err error
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{err: cli.Call(serviceMethod, args, reply)}
	}()

	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case r := <-ch:
		err = r.err
	}

	if cbDone != nil {
		cbDone(err)
	}

	// 连接失败时尝试重连（换节点）
	if err != nil && c.reg != nil {
		_ = c.connect(context.Background())
	}
	return err
}

// Go 异步调用（不经过熔断/限流，适合低延迟场景）。
func (c *MiddlewareClient) Go(serviceMethod string, args interface{}, reply interface{}, done chan *rpc.Call) *rpc.Call {
	c.mu.Lock()
	cli := c.client
	c.mu.Unlock()
	return cli.Go(serviceMethod, args, reply, done)
}

// Close 关闭连接。
func (c *MiddlewareClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
