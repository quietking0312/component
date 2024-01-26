package mnet

import (
	"fmt"
	"testing"
	"time"
)

type MapMessage map[string]any

func (m MapMessage) GetMsgId() string {
	return "hello"
}

func Test_WSClient(t *testing.T) {
	cli := NewWSClient("ws://127.0.0.1:8888")
	conn := cli.Dial()
	wsConn := newWSConn("1", conn, _log)
	route := NewRouter()
	route.Register("world", func(c Context) {
		fmt.Println(string(c.GetMsg().Data))
	})
	ag := NewAgent(wsConn, &JSONParser{}, route)
	go func() {
		ag.Run()
	}()
	for {
		ag.Write(MapMessage{"ping": "helloword"})
		time.Sleep(5 * time.Second)
	}
}
