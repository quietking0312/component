package mnet

import (
	"fmt"
	"github.com/google/uuid"
	"net"
)

const (
	MsgType404 = "404"
)

var _defaultMsg map[string]any

func init() {
	_defaultMsg = make(map[string]any)
}

func SetDefault404Msg(msg any) {
	_defaultMsg[MsgType404] = msg
}

type PackParser interface {
	Unmarshal([]byte) (msg *Msg, err error)
	Marshal(data any) ([]byte, error)
}

type AgentIface interface {
	Run()
	Auth() (string, error)
	Write(any)
	LocalAddr() net.Addr
	Close()
}

type Agent struct {
	conn   Conn
	log    Log
	parser PackParser

	routeFlag bool
	handler   map[string][]HandlerFunc
}

type HandlerFunc func(msg Msg, a *Agent)

func (a *Agent) Auth() (string, error) {
	// TODO 获取第一个数据包进行校验
	return uuid.New().String(), nil
}

func (a *Agent) Run() {
	for {
		_, msg, err := a.conn.Read()
		if err != nil {
			a.log.Error(fmt.Errorf("read message, %v", err))
			break
		}
		m, err := a.parser.Unmarshal(msg)
		if err != nil {
			a.log.Error(fmt.Errorf("unmarshal message, %v", err))
			break
		}
		a.Route(m)
	}
}

func (a *Agent) Write(msg any) {
	data, err := a.parser.Marshal(msg)
	if err != nil {
		a.log.Error(fmt.Errorf("parser.Marshal, %v", err))
	}
	_, err = a.conn.Write(data)
	if err != nil {
		a.log.Error(fmt.Errorf("write message, %v", err))
	}
}

func (a *Agent) Route(msg *Msg) {
	if a.handler == nil {
		m, o := _defaultMsg[MsgType404]
		if o {
			a.Write(m)
		}
		return
	}
	handles, ok := a.handler[msg.Id]
	if !ok {
		m, o := _defaultMsg[MsgType404]
		if o {
			a.Write(m)
		}
		return
	}
	a.routeFlag = true
	for _, fc := range handles {
		fc(*msg, a)
		if !a.routeFlag {
			break
		}
	}
}

func (a *Agent) Next() {
	a.routeFlag = true
}

func (a *Agent) About() {
	a.routeFlag = false
}

func (a *Agent) Close() {
	a.conn.Close()
}

func (a *Agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}
