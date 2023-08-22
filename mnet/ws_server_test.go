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
	route.Register("hello", func(msg Msg, a AgentIface) {
		fmt.Println(string(msg.Data))
		a.Write(ServerMessage{
			"go": "1.20",
		})
	})
	ser := NewWSServer(65535, func(conn *WSConn) AgentIface {
		a := &Agent{conn: conn, log: _log, parser: &JSONParser{}, handler: route}
		return a
	})
	ser.Serve(ln)
}
