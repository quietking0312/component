package mnet

import (
	"github.com/xtaci/kcp-go"
	"net"
	"time"
)

type KCPClient struct {
	addr      string
	block     kcp.BlockCrypt
	closeFlag bool
}

func NewKCPClient(addr string) *KCPClient {
	return &KCPClient{
		addr:      addr,
		closeFlag: false,
	}
}

func (cli *KCPClient) Dial() net.Conn {
	for {
		conn, err := kcp.DialWithOptions(cli.addr, cli.block, 10, 3)
		if err == nil || cli.closeFlag {
			return conn
		}
		time.Sleep(5 * time.Second)
	}

}

func (cli *KCPClient) Close() {

}
