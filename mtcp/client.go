package mtcp

import (
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
)

const (
	HeadLength uint8 = 6
)

// IHead  头
type IHead interface {
	GetDataLength() int16
	GetMethod() string
}

// Msg 消息
type Msg interface {
}

// IPack 包
type IPack interface {
	GetHeadLength() uint8
	UnmarshalHead([]byte) (Head, error)
	UnmarshalMsg([]byte) (Msg, error)
	Marshal(Msg) ([]byte, error)
}

// IRouter  路由
type IRouter interface {
	Call(head IHead, msg []byte)
}

type Client struct {
	factory func() (*net.TCPConn, error)
	conn    *net.TCPConn
	pack    IPack
	headLen uint8
	sync.Mutex
	isClosed bool
	close    chan bool
	isOpen   bool
	router   IRouter
	msgCh    chan []byte
}

func NewClient(pack IPack, router IRouter, factory func() (*net.TCPConn, error)) (*Client, error) {
	conn, err := factory()
	if err != nil {
		return nil, err
	}
	headLen := pack.GetHeadLength()
	if headLen == 0 {
		headLen = HeadLength
	}
	cli := &Client{
		factory:  factory,
		conn:     conn,
		pack:     pack,
		isClosed: false,
		headLen:  headLen,
		close:    make(chan bool),
		router:   router,
		msgCh:    make(chan []byte, 1),
	}
	go cli.autoReset()
	go cli.readLoop()
	return cli, nil
}

func (cli *Client) Close() {
	cli.Lock()
	defer cli.Unlock()
	if !cli.isClosed {
		cli.close <- true
		_ = cli.conn.Close()
		cli.isClosed = true
	}
}

func (cli *Client) Closed() bool {
	return cli.isClosed
}

func (cli *Client) readLoop() {
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			if !cli.isOpen {
				continue
			}
			var head = make([]byte, cli.headLen)
			n, err := cli.conn.Read(head)
			if err != nil {
				cli.isOpen = false
				fmt.Println(err)
				continue
			}
			if n == 0 {
				continue
			}
			h := &Head{}
			err = h.Unmarshal(head)
			if err != nil {
				cli.isOpen = false
				fmt.Println(err)
				continue
			}
			data := make([]byte, h.GetDataLength())
			dn, err := cli.conn.Read(data)
			if err != nil {
				cli.isOpen = false
				fmt.Println(err)
				continue
			}
			if dn == 0 {
				cli.isOpen = false
				fmt.Println(err)
				continue
			}
			// 执行路由
			cli.router.Call(h, data)
		case <-cli.close:
			return
		}
	}
}

func (cli *Client) Write(msg Msg) error {

	data, err := cli.pack.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = cli.conn.Write(data)
	if err != nil {
		if err == syscall.EINVAL {
			cli.isOpen = false
		}
		return err
	}
	return nil
}

// 自动重连
func (cli *Client) autoReset() {
	for {
		select {
		case <-time.After(5 * time.Second):
			if cli.isClosed {
				return
			}
			if cli.isOpen {
				continue
			}
			conn, err := cli.factory()
			if err != nil {
				cli.isOpen = false
			} else {
				cli.conn = conn
				cli.isOpen = true
			}
		}
	}
}
