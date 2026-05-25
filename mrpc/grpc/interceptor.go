package grpc

import (
	"context"
	"sync"

	"github.com/quietking0312/component/mrpc/balancer"
	"github.com/quietking0312/component/mrpc/middleware"
	"github.com/quietking0312/component/mrpc/registry"
	"google.golang.org/grpc"
)

// ========== 服务发现 Resolver ==========

// resolver 监听注册中心变更，维护最新实例列表
type resolver struct {
	mu        sync.RWMutex
	instances []registry.ServiceInfo
}

func newResolver(ctx context.Context, reg registry.Registry, serviceName string) (*resolver, error) {
	ch, err := reg.Watch(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	r := &resolver{}
	// 等第一批数据到位
	if list, ok := <-ch; ok {
		r.instances = list
	}
	go func() {
		for list := range ch {
			r.mu.Lock()
			r.instances = list
			r.mu.Unlock()
		}
	}()
	return r, nil
}

func (r *resolver) pick(b balancer.Balancer) (registry.ServiceInfo, error) {
	r.mu.RLock()
	instances := make([]registry.ServiceInfo, len(r.instances))
	copy(instances, r.instances)
	r.mu.RUnlock()
	return b.Pick(instances)
}

// DialWithDiscovery 使用服务发现创建 gRPC 客户端。
// 每次调用前通过 balancer 选出一个实例地址再 Dial。
func DialWithDiscovery(
	ctx context.Context,
	reg registry.Registry,
	serviceName string,
	b balancer.Balancer,
	cfg *ClientConfig,
) (*Client, error) {
	r, err := newResolver(ctx, reg, serviceName)
	if err != nil {
		return nil, err
	}
	info, err := r.pick(b)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &ClientConfig{}
	}
	cfg.Target = info.Addr
	return NewClient(cfg)
}

// ========== gRPC 拦截器 ==========

// UnaryCircuitBreaker 客户端一元 RPC 熔断拦截器。
func UnaryCircuitBreaker(cb *middleware.CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		done, ok := cb.Allow()
		if !ok {
			return middleware.ErrCircuitOpen
		}
		err := invoker(ctx, method, req, reply, cc, opts...)
		done(err)
		return err
	}
}

// UnaryRateLimiter 客户端一元 RPC 限流拦截器。
func UnaryRateLimiter(rl *middleware.RateLimiter) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !rl.Allow() {
			return middleware.ErrRateLimited
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamCircuitBreaker 客户端流式 RPC 熔断拦截器。
func StreamCircuitBreaker(cb *middleware.CircuitBreaker) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		done, ok := cb.Allow()
		if !ok {
			return nil, middleware.ErrCircuitOpen
		}
		stream, err := streamer(ctx, desc, cc, method, opts...)
		done(err)
		return stream, err
	}
}

// StreamRateLimiter 客户端流式 RPC 限流拦截器。
func StreamRateLimiter(rl *middleware.RateLimiter) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if !rl.Allow() {
			return nil, middleware.ErrRateLimited
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// ---- 服务端拦截器 ----

// UnaryServerCircuitBreaker 服务端一元 RPC 熔断拦截器（防止后端过载）。
func UnaryServerCircuitBreaker(cb *middleware.CircuitBreaker) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		done, ok := cb.Allow()
		if !ok {
			return nil, middleware.ErrCircuitOpen
		}
		resp, err := handler(ctx, req)
		done(err)
		return resp, err
	}
}

// UnaryServerRateLimiter 服务端一元 RPC 限流拦截器。
func UnaryServerRateLimiter(rl *middleware.RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		if !rl.Allow() {
			return nil, middleware.ErrRateLimited
		}
		return handler(ctx, req)
	}
}

// ChainUnaryClient 组合多个客户端一元拦截器（执行顺序：参数顺序）。
func ChainUnaryClient(interceptors ...grpc.UnaryClientInterceptor) grpc.DialOption {
	return grpc.WithChainUnaryInterceptor(interceptors...)
}

// ChainUnaryServer 组合多个服务端一元拦截器（执行顺序：参数顺序）。
func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(interceptors...)
}
