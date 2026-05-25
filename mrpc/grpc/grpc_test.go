package grpc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/quietking0312/component/mrpc/grpc"
	pb "github.com/quietking0312/component/mrpc/proto"
	g "google.golang.org/grpc"
)

// helloImpl 业务实现
type helloImpl struct {
	pb.UnimplementedServiceServer
}

func (h *helloImpl) SayHello(_ context.Context, req *pb.HelloReq) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "hello, " + req.Name}, nil
}

func TestGRPC(t *testing.T) {
	// 启动服务端
	srv := grpc.NewServer(&grpc.ServerConfig{Address: ":19000"})
	srv.Register(func(s *g.Server) {
		pb.RegisterServiceServer(s, &helloImpl{})
	})

	go func() {
		if err := srv.Serve(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	// 客户端连接
	cli, err := grpc.NewClient(&grpc.ClientConfig{Target: "127.0.0.1:19000"})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	pbCli := pb.NewServiceClient(cli.ClientConn)
	resp, err := pbCli.SayHello(context.Background(), &pb.HelloReq{Name: "world"})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("grpc response:", resp.Message)

	srv.Stop(3 * time.Second)
}
