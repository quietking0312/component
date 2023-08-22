package mnet

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/quietking0312/component/mnet/pb"
	"net"
	"testing"
)

func Test_NewTCPServer(t *testing.T) {
	ln, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		t.Fatal(err)
	}
	var route = make(map[string][]HandlerFunc)
	route["0"] = append(route["0"], func(msg Msg, a *Agent) {
		var req pb.Ping
		if err := proto.Unmarshal(msg.Data, &req); err != nil {
			t.Fatal(err)
		}
		fmt.Println(req.GetArgs())
		a.Write(&pb.Pong{
			Data: "go 1.20",
		})
	}, func(msg Msg, a *Agent) {
		fmt.Println("2222")
	})
	ser := NewTCPServer(65535, func(conn *TCPConn) AgentIface {
		a := &Agent{conn: conn, log: _log, parser: &ProtoParser{}, handler: route}
		return a
	})
	ser.Serve(ln)
}
