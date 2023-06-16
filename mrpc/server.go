package mrpc

import (
	"bufio"
	"io"
	"net"
	"sync"
)

type Server struct {
}

func (server *Server) ServeCodec(codec ServerCodec) {
	wg := new(sync.WaitGroup)
	for {
		wg.Add(1)
	}
	wg.Wait()
	codec.Close()
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	buf := bufio.NewWriterSize(conn, 4096)
	_ = buf
}

func (server Server) Run() {
	lis, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		return
	}
	for {
		conn, err := lis.Accept()
		if err != nil {
			panic(err)
		}
		server.ServeConn(conn)
	}
}

type Request struct {
	ServiceMethod string
	Seq           uint64
}

type Response struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

type ServerCodec interface {
	ReadRequestHeader(*Request) error
	ReadRequestBody(any) error
	WriteResponse(*Response, any) error
	Close() error
}
