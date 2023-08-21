package mnet

import (
	"fmt"
	"github.com/google/uuid"
	"net"
)

type Gate struct {
}

func (gate *Gate) Run(closeFlag chan bool) {

}

type Agent interface {
	Run()
	Auth() (string, error)
	Write(any)
	LocalAddr() net.Addr
	Close()
}

type agent struct {
	conn   Conn
	log    Log
	parser PackParser
}

type PackParser interface {
	Unmarshal([]byte) (msg *Msg, err error)
	Marshal(data any) ([]byte, error)
	Route(msg *Msg, a Agent)
}

func (a *agent) Auth() (string, error) {
	// TODO 获取第一个数据包进行校验
	return uuid.New().String(), nil
}

func (a *agent) Run() {
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
		a.parser.Route(m, a)
	}
}

func (a *agent) Write(msg any) {
	data, err := a.parser.Marshal(msg)
	if err != nil {
		a.log.Error(fmt.Errorf("parser.Marshal, %v", err))
	}
	_, err = a.conn.Write(data)
	if err != nil {
		a.log.Error(fmt.Errorf("write message, %v", err))
	}
}

func (a *agent) Close() {
	a.conn.Close()
}

func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}
