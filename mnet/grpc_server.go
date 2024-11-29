package mnet

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

type GRPCServer struct {
	ln     net.Listener
	logger Log
	desc   grpc.ServiceDesc
	srv    any //继承该结构体的指针用来注册服务方法
}

func NewGRPCServer() *GRPCServer {
	srv := &GRPCServer{}
	srv.srv = srv
	return srv
}

func (g *GRPCServer) SetLogger(log Log) {
	g.logger = log
}

func (g *GRPCServer) SerSrv(srv any) {
	g.srv = srv
}

func (g *GRPCServer) Server(ln net.Listener) {
	s := grpc.NewServer()
	s.RegisterService(&g.desc, g.srv)
	reflection.Register(s)

}
