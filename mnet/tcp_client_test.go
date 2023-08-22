package mnet

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/quietking0312/component/mnet/pb"
	"testing"
	"time"
)

func Test_NEWTCPClient(t *testing.T) {
	cli := NewTCPClient("127.0.0.1:8888")
	conn := cli.Dial()
	tcpConn := newTCPConn("", conn, _log)
	var route = make(map[string][]HandlerFunc)
	route["0"] = append(route["0"], func(msg Msg, a *Agent) {
		var req pb.Pong
		if err := proto.Unmarshal(msg.Data, &req); err != nil {
			t.Fatal(err)
		}
		fmt.Println(req.GetData())
	}, func(msg Msg, a *Agent) {
		fmt.Println("3333")
	})
	ag := &Agent{
		conn:    tcpConn,
		log:     _log,
		parser:  &ProtoParser{},
		handler: route,
	}
	go func() {
		ag.Run()
	}()
	for {
		ag.Write(&pb.Ping{
			Args: "hello",
		})
		time.Sleep(5 * time.Second)
	}
}
