package mnet

import (
	"net"
	"testing"
)

func Test_NewTCPServer(t *testing.T) {
	ln, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		t.Fatal(err)
	}
	ser := NewTCPServer(65535, func(conn *TCPConn) Agent {
		a := &agent{conn: conn, log: _log, parser: &JSONParser{}}
		return a
	})
	ser.Serve(ln)
}
