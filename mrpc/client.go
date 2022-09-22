package mrpc

import (
	"net"
	"sync"
	"time"
)

const (
	HeadLength = 6
)

type Head interface {
	GetDataLength() int64
}

type Pack interface {
	UnmarshalHead([]byte) (Head, error)
	UnmarshalData([]byte) error
}

type Client struct {
	factory func() (*net.TCPConn, error)
	conn    *net.TCPConn
	pack    Pack
	HeadLen int
	sync.Mutex
	isClosed bool
}

func NewClient(headLen int, pack Pack, factory func() (*net.TCPConn, error)) (*Client, error) {
	conn, err := factory()
	if err != nil {
		return nil, err
	}
	if headLen == 0 {
		headLen = HeadLength
	}
	cli := &Client{
		factory:  factory,
		conn:     conn,
		pack:     pack,
		isClosed: false,
		HeadLen:  headLen,
	}
	go cli.readLoop()
	return cli, nil
}

func (cli *Client) Close() {
	cli.Lock()
	defer cli.Unlock()
	if !cli.isClosed {
		cli.conn.Close()
		cli.isClosed = true
	}
}

func (cli *Client) readLoop() {
	defer cli.Close()
	for {
		select {
		case <-time.After(5 * time.Second):
			if cli.isClosed {
				return
			}
		default:
			var head = make([]byte, cli.HeadLen)
			n, err := cli.conn.Read(head)
			if err != nil {
				return
			}
			if n == 0 {
				continue
			}
			h, err := cli.pack.UnmarshalHead(head)
			if err != nil {
				return
			}
			data := make([]byte, h.GetDataLength())
			dn, err := cli.conn.Read(data)
			if err != nil {
				return
			}
			if dn == 0 {
				return
			}
			if err := cli.pack.UnmarshalData(data); err != nil {
				return
			}
		}
	}
}

func (cli *Client) Write(data []byte) error {
	_, err := cli.conn.Write(data)
	if err != nil {
		return err
	}
	return nil
}
