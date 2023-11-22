package mnet

import "net"

type Conn interface {
	Read() (int, []byte, error)
	Write(b []byte) (int, error)
	LocalAddr() net.Addr
	Close() error
	RemoteAddr() net.Addr
}
