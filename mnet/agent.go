package mnet

import (
	"fmt"
	"github.com/google/uuid"
	"net"
)

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

type RouterIface interface {
	Route(msg *Msg)
	SetAgent(a AgentIface)
	Next()
	About()
}

type Agent struct {
	conn     Conn
	log      Log
	parser   PackParser
	handler  RouterIface
	AuthFunc AuthFunc // 第一个数据包调用该函数
}

type AuthFunc func(msg *Msg) (string, error)

func NewAgent(conn Conn, parser PackParser, router RouterIface) *Agent {
	return &Agent{
		conn:    conn,
		log:     _log,
		parser:  parser,
		handler: router,
	}
}

func (a *Agent) SetLog(log Log) *Agent {
	a.log = log
	return a
}

func (a *Agent) Auth() (string, error) {
	if a.AuthFunc != nil {
		_, msg, err := a.conn.Read()
		if err != nil {
			a.log.Error(fmt.Errorf("read message, %v", err))
			return "", err
		}
		m, err := a.parser.Unmarshal(msg)
		if err != nil {
			a.log.Error(fmt.Errorf("unmarshal message, %v", err))
			return "", err
		}
		return a.AuthFunc(m)
	}
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
	a.handler.SetAgent(a)
	a.handler.Route(msg)
}

func (a *Agent) Next() {
	a.handler.Next()
}

func (a *Agent) About() {
	a.handler.About()
}

func (a *Agent) Close() {
	a.conn.Close()
}

func (a *Agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}
