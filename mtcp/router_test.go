package mtcp

import (
	"fmt"
	"testing"
	"time"
)

func TestRouter_Call(t *testing.T) {
	r := Router{
		r: make(map[string]func(msg Msg)),
	}
	r.r["hello"] = func(msg Msg) {
		fmt.Println(msg)
	}
	pack := JsonPack{}
	head, _ := pack.UnmarshalHead(nil)
	r.Call(head, "hello")
	time.Sleep(5)
}
