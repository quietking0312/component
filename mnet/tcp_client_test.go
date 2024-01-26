package mnet

import (
	"fmt"
	"github.com/quietking0312/component/mnet/pb"
	"google.golang.org/protobuf/proto"
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
	route.Register(c(&pb.Pong{}), func(c Context) {
		var req pb.Pong
		if err := proto.Unmarshal(c.GetMsg().Data, &req); err != nil {
			t.Fatal(err)
		}
		fmt.Println(req.GetData())
	}, func(c Context) {
		fmt.Println("3333")
	})
	ag := NewAgent(tcpConn, &ProtoParser{}, route)
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
