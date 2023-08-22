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
	route := make(map[string][]HandlerFunc)
	route["world"] = append(route["world"], func(msg Msg, a *Agent) {
		fmt.Println(string(msg.Data))
	})
	ag := Agent{conn: wsConn, log: _log, parser: &JSONParser{}, handler: route}
	go func() {
		ag.Run()
	}()
	for {
		ag.Write(MapMessage{"ping": "helloword"})
		time.Sleep(5 * time.Second)
	}
}
