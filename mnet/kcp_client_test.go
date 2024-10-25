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

func TestNewKCPClient(t *testing.T) {
	cli := NewKCPClient("127.0.0.1:8888")
	conn := cli.Dial()
	var route = NewRouter()
	kcpConn := newKCPConn("1", conn, _log)
	route.Register(strconv.FormatInt(int64(pb.S2C_value[strings.ToLower(reflect.TypeOf(&pb.Pong{}).Elem().Name())]), 10), func(c Context) {
		var req pb.Pong
		if err := proto.Unmarshal(c.GetMsg().Data, &req); err != nil {
			t.Fatal(err)
		}
		fmt.Println(req.GetData())
	}, func(c Context) {
		fmt.Println("3333")
	})
	ag := NewAgent(kcpConn, NewProtoParser(pb.C2S_value, func(data any) string {
		return strings.ToLower(reflect.TypeOf(data).Elem().Name())
	}), route)
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
