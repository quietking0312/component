package gorpc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/quietking0312/component/mrpc/balancer"
	"github.com/quietking0312/component/mrpc/gorpc"
	"github.com/quietking0312/component/mrpc/middleware"
	"github.com/quietking0312/component/mrpc/registry"
)

// ========== Gob 编码测试 ==========

type MathArgs struct {
	A, B int
}

type MathService struct{}

func (m *MathService) Add(args *MathArgs, reply *int) error {
	*reply = args.A + args.B
	return nil
}

func (m *MathService) Mul(args *MathArgs, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func TestGorpcGob(t *testing.T) {
	srv := gorpc.NewServer(&gorpc.ServerConfig{Address: ":19001", Codec: gorpc.CodecGob})
	if err := srv.Register(&MathService{}); err != nil {
		t.Fatal(err)
	}

	go func() { _ = srv.Serve() }()
	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	cli, err := gorpc.NewClient(&gorpc.ClientConfig{Target: "127.0.0.1:19001", Codec: gorpc.CodecGob})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	var result int
	if err := cli.Call("MathService.Add", &MathArgs{A: 10, B: 20}, &result); err != nil {
		t.Fatal(err)
	}
	fmt.Println("gob Add(10,20) =", result)

	if err := cli.Call("MathService.Mul", &MathArgs{A: 6, B: 7}, &result); err != nil {
		t.Fatal(err)
	}
	fmt.Println("gob Mul(6,7) =", result)
}

// ========== JSON 编码测试 ==========

func TestGorpcJSON(t *testing.T) {
	srv := gorpc.NewServer(&gorpc.ServerConfig{Address: ":19002", Codec: gorpc.CodecJSON})
	if err := srv.Register(&MathService{}); err != nil {
		t.Fatal(err)
	}

	go func() { _ = srv.Serve() }()
	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	cli, err := gorpc.NewClient(&gorpc.ClientConfig{Target: "127.0.0.1:19002", Codec: gorpc.CodecJSON})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	var result int
	if err := cli.Call("MathService.Add", &MathArgs{A: 3, B: 4}, &result); err != nil {
		t.Fatal(err)
	}
	fmt.Println("json Add(3,4) =", result)
}

// ========== 异步调用测试 ==========

func TestGorpcAsync(t *testing.T) {
	srv := gorpc.NewServer(&gorpc.ServerConfig{Address: ":19003"})
	if err := srv.Register(&MathService{}); err != nil {
		t.Fatal(err)
	}

	go func() { _ = srv.Serve() }()
	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	cli, err := gorpc.NewClient(&gorpc.ClientConfig{Target: "127.0.0.1:19003"})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	var r1, r2 int
	call1 := cli.Go("MathService.Add", &MathArgs{A: 1, B: 2}, &r1, nil)
	call2 := cli.Go("MathService.Mul", &MathArgs{A: 5, B: 5}, &r2, nil)

	<-call1.Done
	<-call2.Done

	if call1.Error != nil {
		t.Fatal(call1.Error)
	}
	if call2.Error != nil {
		t.Fatal(call2.Error)
	}
	fmt.Printf("async Add(1,2)=%d  Mul(5,5)=%d\n", r1, r2)
}

// ========== 中间件：服务发现 + 熔断 + 限流 ==========

func TestMiddlewareClient(t *testing.T) {
	// 启动两个服务实例
	for _, addr := range []string{":19010", ":19011"} {
		srv := gorpc.NewServer(&gorpc.ServerConfig{Address: addr})
		_ = srv.Register(&MathService{})
		go srv.Serve()
	}
	time.Sleep(100 * time.Millisecond)

	// 内存注册中心
	reg := registry.NewMemRegistry()
	ctx := context.Background()
	_ = reg.Register(ctx, registry.ServiceInfo{Name: "math", Addr: "127.0.0.1:19010"})
	_ = reg.Register(ctx, registry.ServiceInfo{Name: "math", Addr: "127.0.0.1:19011"})

	cli, err := gorpc.NewMiddlewareClient(ctx, &gorpc.MiddlewareClientConfig{
		ClientConfig: gorpc.ClientConfig{Codec: gorpc.CodecGob},
		Registry:     reg,
		ServiceName:  "math",
		Balancer:     balancer.NewRoundRobin(),
		CircuitBreaker: &middleware.CircuitConfig{
			WindowSize:       10,
			FailureRatio:     0.5,
			OpenTimeout:      2 * time.Second,
			HalfOpenRequests: 2,
		},
		RateLimiter: &middleware.RateLimiterConfig{
			Rate:  100,
			Burst: 10,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	for i := 0; i < 5; i++ {
		var result int
		if err := cli.Call(ctx, "MathService.Add", &MathArgs{A: i, B: i}, &result); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		fmt.Printf("middleware Add(%d,%d) = %d\n", i, i, result)
	}
}

// ========== 限流测试 ==========

func TestRateLimiter(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{Rate: 5, Burst: 2})
	allowed, denied := 0, 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			allowed++
		} else {
			denied++
		}
	}
	fmt.Printf("rate limiter: allowed=%d denied=%d\n", allowed, denied)
	if allowed > 3 {
		t.Errorf("expected at most 3 allowed (burst=2 + ~1 replenished), got %d", allowed)
	}
}

// ========== 熔断测试 ==========

func TestCircuitBreaker(t *testing.T) {
	cb := middleware.NewCircuitBreaker(middleware.CircuitConfig{
		WindowSize:       4,
		FailureRatio:     0.5,
		OpenTimeout:      500 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	// 模拟 4 次失败，填满窗口触发熔断
	for i := 0; i < 4; i++ {
		done, ok := cb.Allow()
		if ok {
			done(fmt.Errorf("fake error"))
		}
	}
	fmt.Println("state after failures:", cb.State())

	// 应被拒绝
	_, ok := cb.Allow()
	if ok {
		t.Error("expected circuit open")
	}
	fmt.Println("request blocked:", !ok)

	// 等待半开
	time.Sleep(600 * time.Millisecond)
	done, ok := cb.Allow()
	if !ok {
		t.Error("expected half-open allow")
	}
	done(nil) // 试探成功
	fmt.Println("half-open probe passed, state:", cb.State())
}
