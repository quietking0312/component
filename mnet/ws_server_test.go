package mnet

import (
	"fmt"
	"net"
	"testing"
)

type ServerMessage map[string]any

func (m ServerMessage) GetMsgId() string {
	return "world"
}

func Test_NewWSServer(t *testing.T) {
	ln, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		t.Fatal(err)
	}
	var route = NewRouter()
	route.Register("hello", func(c Context) {
		fmt.Println(string(c.GetMsg().Data))
		c.Write(ServerMessage{
			"go": "1.20",
		})
	})
	ser := NewWSServer(65535, func(conn *WSConn) AgentIface {
		return NewAgent(conn, &JSONParser{}, route)
	})
	ser.Serve(ln)
}
