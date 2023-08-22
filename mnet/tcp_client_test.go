package mnet

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/quietking0312/component/mnet/pb"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_NEWTCPClient(t *testing.T) {
	cli := NewTCPClient("127.0.0.1:8888")
	conn := cli.Dial()
	tcpConn := newTCPConn("", conn, _log)
	var route = NewRouter()
	c := func(message proto.Message) string {
		msgType := reflect.TypeOf(message)
		return strconv.FormatInt(int64(pb.S2C_value[strings.ToLower(msgType.Elem().Name())]), 10)
	}
	route.Register(c(&pb.Pong{}), func(msg Msg, a AgentIface) {
		var req pb.Pong
		if err := proto.Unmarshal(msg.Data, &req); err != nil {
			t.Fatal(err)
		}
		fmt.Println(req.GetData())
	}, func(msg Msg, a AgentIface) {
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
