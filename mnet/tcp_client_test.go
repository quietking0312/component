package mnet

import (
	"github.com/quietking0312/component/mnet/pb"
	"testing"
	"time"
)

func Test_NEWTCPClient(t *testing.T) {
	cli := NewTCPClient("127.0.0.1:8888")
	conn := cli.Dial()
	tcpConn := newTCPConn("", conn, _log)
	ag := &agent{
		tcpConn,
		_log,
		&ProtoParser{},
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
