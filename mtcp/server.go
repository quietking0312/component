package mtcp

import (
	"fmt"
	"net"
	"sync"
)

type Token interface {
}

type Server struct {
	lin           net.Listener
	MaxConnNumber uint16
	ConnNumber    uint16
	conns         map[Token]*Conn
	sync.Mutex
}

func NewServer(lin net.Listener) *Server {
	return &Server{}
}

func (ser *Server) Accept() error {

	for {
		conn, err := ser.lin.Accept()
		if err != nil {
			return err
		}
		c, _ := NewConn(func() (net.Conn, error) {
			return conn, nil
		}, nil, nil)
		_ = ser.register("hello", c)
	}
}

func (ser *Server) register(token Token, conn *Conn) error {
	ser.Lock()
	defer ser.Unlock()
	if ser.ConnNumber >= ser.MaxConnNumber {
		return fmt.Errorf("conn max")
	}
	if _, ok := ser.conns[token]; ok {
		return fmt.Errorf("%v is exists", token)
	}
	ser.ConnNumber += 1
	ser.conns[token] = conn
	return nil
}

func (ser *Server) Send(token Token, msg Msg) error {
	c, ok := ser.conns[token]
	if !ok {
		return fmt.Errorf("%v not is exists", token)
	}
	err := c.Write([]byte(""))
	return err
}

func (ser *Server) SendAll(msg Msg) error {
	for token, _ := range ser.conns {
		go func(token Token) {
			ser.Send(token, msg)
		}(token)
	}
	return nil
}
