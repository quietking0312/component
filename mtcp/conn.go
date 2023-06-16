package mtcp

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
)

type ConnSettings struct {
	AutoReset     bool
	AutoResetTime time.Duration
}

type Conn struct {
	// 连接
	c net.Conn
	// 工厂
	factory func() (net.Conn, error)
	// 包头长度
	headLen uint8
	sync.Mutex
	isClosed bool
	close    chan bool
	isOpen   bool
	a        string
	// 路由
	router   IRouter
	msgCh    chan []byte
	Settings *ConnSettings
	Writer   *bufio.Writer
	Reader   *bufio.Reader
}

func NewConn(factory func() (net.Conn, error), router IRouter, settings *ConnSettings) (*Conn, error) {
	c := &Conn{
		factory:  factory,
		router:   router,
		isClosed: false,
		isOpen:   true,
	}
	if settings.AutoReset {
		go c.autoReset()
	} else {
		err := c.Reset()
		return nil, err
	}
	return c, nil
}

func (c *Conn) Close() {
	c.Lock()
	defer c.Unlock()
	if !c.isClosed {
		c.close <- true
		_ = c.c.Close()
		c.isClosed = true
	}
}

func (c *Conn) Closed() bool {
	return c.isClosed
}

func (c *Conn) readNext(n int) {
	for c.Reader.Buffered() < n {
	}
	return
}

func (c *Conn) readLoop() {
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			if !c.isOpen {
				continue
			}
			if c.Reader.Buffered() < int(c.headLen) {
				continue
			}
			var head = make([]byte, c.headLen)
			n, err := c.Reader.Read(head)
			if err != nil {
				c.isOpen = false
				fmt.Println(err)
				continue
			}
			if n == 0 {
				continue
			}
			h := &Head{}
			err = h.Unmarshal(head)
			if err != nil {
				c.isOpen = false
				fmt.Println(err)
				continue
			}
			data := make([]byte, h.GetDataLength())
			dn, err := c.Reader.Read(data)
			if err != nil {
				c.isOpen = false
				fmt.Println(err)
				continue
			}
			if dn == 0 {
				c.isOpen = false
				fmt.Println(err)
				continue
			}
			// 执行路由
			c.router.Call(h, data)
		case <-c.close:
			return
		}
	}
}

func (c *Conn) Write(data []byte) error {
	_, err := c.Writer.Write(data)
	if err != nil {
		if err == syscall.EINVAL {
			c.isOpen = false
		}
		return err
	}
	return c.Writer.Flush()
}

// 自动重连
func (c *Conn) autoReset() {
	for {
		select {
		case <-time.After(5 * time.Second):
			if c.isClosed {
				return
			}
			if c.isOpen {
				continue
			}
			_ = c.Reset()
		}
	}
}

func (c *Conn) Reset() error {
	conn, err := c.factory()
	if err != nil {
		c.isOpen = false
		return err
	}
	c.c = conn
	c.Reader = bufio.NewReader(c.c)
	c.Writer = bufio.NewWriter(c.c)
	c.isOpen = true
	return nil
}
