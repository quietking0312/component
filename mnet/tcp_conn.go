package mnet

import (
	"fmt"
	"net"
)

type TCPConn struct {
	Id        string
	conn      net.Conn
	closeFlag bool
	log       Log
	msgParser *MsgParser
}

func newTCPConn(id string, conn net.Conn, log Log) *TCPConn {
	tcpConn := new(TCPConn)
	tcpConn.Id = id
	tcpConn.conn = conn
	tcpConn.log = log
	tcpConn.closeFlag = false
	tcpConn.msgParser = NewMsgParser(2)
	return tcpConn
}

func (c *TCPConn) SetId(id string) {
	c.Id = id
}

func (c *TCPConn) Read() (int, []byte, error) {
	msg, err := c.msgParser.Read(c.conn)
	return len(msg), msg, err
}

func (c *TCPConn) Write(b []byte) (int, error) {
	if c.closeFlag || b == nil {
		return 0, nil
	}
	i, err := c.msgParser.Write(c.conn, b)
	if err != nil {
		c.log.Error(fmt.Errorf("id:%s write err:%v", c.Id, err))
	}
	return i, err
}

func (c *TCPConn) Close() error {
	if c.closeFlag {
		return nil
	}
	c.closeFlag = true
	return nil
}
