package mnet

import (
	"fmt"
	"github.com/google/uuid"
)

type Gate struct {
}

func (gate *Gate) Run(closeFlag chan bool) {

}

type Agent interface {
	Run()
	Auth() (string, error)
	Close()
}

type agent struct {
	conn   Conn
	log    Log
	parser PackParser
}

type PackParser interface {
	Unmarshal([]byte) (err error)
	Marshal(data any) ([]byte, error)
}

func (a *agent) Auth() (string, error) {
	return uuid.New().String(), nil
}

func (a *agent) Run() {
	for {
		_, msg, err := a.conn.Read()
		if err != nil {
			a.log.Error(fmt.Errorf("read message, %v", err))
			break
		}
		if err := a.parser.Unmarshal(msg); err != nil {
			a.log.Error(fmt.Errorf("unmarshal message, %v", err))
			break
		}
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
}
