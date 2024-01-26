package mnet

import (
	"fmt"
	"github.com/quietking0312/component/mnet/pb"
	"google.golang.org/protobuf/proto"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func Test_NewTCPServer(t *testing.T) {
	ln, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		t.Fatal(err)
	}
	var route = NewRouter()
	c := func(message proto.Message) string {
		msgType := reflect.TypeOf(message)
		return strconv.FormatInt(int64(pb.C2S_value[strings.ToLower(msgType.Elem().Name())]), 10)
	}
	route.Register(c(&pb.Ping{}), func(c Context) {
		var req pb.Ping
		if err := proto.Unmarshal(c.GetMsg().Data, &req); err != nil {
			t.Fatal(err)
		}
		fmt.Println(req.GetArgs())
		c.Write(&pb.Pong{
			Data: "go 1.20",
		})
	}, func(c Context) {
		fmt.Println("2222")
	})
	ser := NewTCPServer(65535, func(conn *TCPConn) AgentIface {
		a := NewAgent(conn, &ProtoParser{}, route)
		return a
	})
	ser.Serve(ln)
}
