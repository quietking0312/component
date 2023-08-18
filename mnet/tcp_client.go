package mnet

import (
	"net"
	"time"
)

type TCPClient struct {
	addr      string
	closeFlag bool
}

func NewTCPClient(addr string) *TCPClient {
	return &TCPClient{
		addr:      addr,
		closeFlag: false,
	}
}

func (cli *TCPClient) Dial() net.Conn {
	for {
		conn, err := net.Dial("tcp", cli.addr)
		if err == nil || cli.closeFlag {
			return conn
		}
		time.Sleep(5 * time.Second)
	}

}

func (cli *TCPClient) Close() {

}
