package mnet

import (
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
		&JSONParser{},
	}
	go func() {
		ag.Run()
	}()
	for {
		ag.Write(MapMessage{
			"ping": "world",
		})
		time.Sleep(5 * time.Second)
	}
}
