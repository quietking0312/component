package mrpc

import (
	"context"
	"fmt"
	pb "github.com/quietking0312/component/mrpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	pb.UnimplementedServiceServer
}

func (s *Server) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloReply, error) {
	return &pb.HelloReply{
		Message: req.GetName() + "  golang",
	}, nil
}

func Serve() {
	lis, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println("", err)
		return
	}

	s := grpc.NewServer()
	pb.RegisterServiceServer(s, &Server{})
	reflection.Register(s)
	go func() {
		if err := s.Serve(lis); err != nil {
			fmt.Println(err)
		}
	}()
	signChan := make(chan os.Signal, 1)
	signal.Notify(signChan, syscall.SIGINT, syscall.SIGTERM)
	<-signChan
	fmt.Println("shutting down the server...")
	s.GracefulStop()
	fmt.Println("server has been shut down gracefully")
}

func Client() {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.Dial("127.0.0.1:8888", opts)
	if err != nil {
		fmt.Println("", err)
		return
	}

	client := pb.NewServiceClient(conn)
	req := &pb.HelloReq{
		Name: "helloworld",
	}
	resp, err := client.SayHello(context.Background(), req)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp.Message)
}
