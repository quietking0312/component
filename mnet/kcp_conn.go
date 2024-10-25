package mnet

import (
	"net"
)

type KCPConn struct {
	*TCPConn
}

func newKCPConn(id string, conn net.Conn, log Log) *KCPConn {
	kcpConn := new(KCPConn)
	kcpConn.TCPConn = newTCPConn(id, conn, log)
	return kcpConn
}
