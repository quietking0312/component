package mnet

import (
	"fmt"
	"github.com/quietking0312/component/mnet/pb"
	"google.golang.org/protobuf/proto"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestNewKCPServer(t *testing.T) {

	var route = NewRouter()
	//c := func(message proto.Message) string {
	//	msgType := reflect.TypeOf(message)
	//	return strconv.FormatInt(int64(pb.C2S_value[strings.ToLower(msgType.Elem().Name())]), 10)
	//}
	route.Register(strconv.FormatInt(int64(pb.C2S_value[strings.ToLower(reflect.TypeOf(&pb.Ping{}).Elem().Name())]), 10), func(c Context) {
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
	ser := NewKCPServer(65535, func(conn *KCPConn) AgentIface {
		a := NewAgent(conn, NewProtoParser(pb.S2C_value, func(data any) string {
			return strings.ToLower(reflect.TypeOf(data).Elem().Name())
		}), route)
		return a
	})
	ser.Serve("0.0.0.0:8888")
}
