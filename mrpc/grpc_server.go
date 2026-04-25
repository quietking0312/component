package mrpc

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServerOption gRPC 服务端配置选项
type ServerOption struct {
	Address string              // 监听地址，如 ":8888"
	Options []grpc.ServerOption // 自定义 gRPC 选项
}

// DefaultServerOption 返回默认服务端配置
func DefaultServerOption() *ServerOption {
	return &ServerOption{
		Address: ":8888",
	}
}

// Serve 启动 gRPC 服务端
//   - opt: 服务端配置，为 nil 时使用默认配置
//   - register: 服务注册函数，用于注册业务 Service
//
// 使用示例：
//
//	mrpc.Serve(nil, func(s *grpc.Server) {
//	    pb.RegisterMyServiceServer(s, &myServiceImpl{})
//	})
func Serve(opt *ServerOption, register func(s *grpc.Server)) error {
	if opt == nil {
		opt = DefaultServerOption()
	}

	lis, err := net.Listen("tcp", opt.Address)
	if err != nil {
		return fmt.Errorf("rpc listen failed: %w", err)
	}

	s := grpc.NewServer(opt.Options...)
	if register != nil {
		register(s)
	}

	return s.Serve(lis)
}

// Dial 创建 gRPC 客户端连接
//   - target: 服务端地址，如 "127.0.0.1:8888"
//   - opts: 可选的 DialOption
func Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	defaultOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	defaultOpts = append(defaultOpts, opts...)
	return grpc.Dial(target, defaultOpts...)
}

// DialContext 带上下文的 gRPC 客户端连接创建
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	defaultOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	defaultOpts = append(defaultOpts, opts...)
	return grpc.DialContext(ctx, target, defaultOpts...)
}
