package msock

import (
	"net"
	"sync"
	"time"

	"github.com/lxzan/gws"
)

// gwsConn gws WebSocket连接实现
type gwsConn struct {
	*baseConn
	conn    *gws.Conn
	server  *Server
	codec   Codec
	writeMu sync.Mutex
}

// newGWSConn 创建gws连接
func newGWSConn(conn *gws.Conn, server *Server, codec Codec) *gwsConn {
	c := &gwsConn{
		baseConn: newBaseConn(ConnTypeGWS),
		conn:     conn,
		server:   server,
		codec:    codec,
	}
	c.initSendQueue(128)
	c.sendWg.Add(1)
	go c.sendLoop()
	return c
}

// LocalAddr 返回本地地址
func (c *gwsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr 返回远程地址
func (c *gwsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Send 发送消息
func (c *gwsConn) Send(msg Message) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}

	return c.SendBytes(data)
}

// SendBytes 发送二进制消息
func (c *gwsConn) SendBytes(data []byte) error {
	return c.sendAsync(data)
}

// sendLoop gws发送协程
func (c *gwsConn) sendLoop() {
	defer c.sendWg.Done()
	for data := range c.sendCh {
		writeTimeout := 10 * time.Second
		if c.server != nil && c.server.config.WriteTimeout > 0 {
			writeTimeout = c.server.config.WriteTimeout
		}
		c.writeMu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		_ = c.conn.WriteMessage(gws.OpcodeBinary, data)
		c.writeMu.Unlock()
	}
}

// SendText 发送文本消息
func (c *gwsConn) SendText(text string) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	writeTimeout := 10 * time.Second
	if c.server != nil && c.server.config.WriteTimeout > 0 {
		writeTimeout = c.server.config.WriteTimeout
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	return c.conn.WriteMessage(gws.OpcodeText, []byte(text))
}

// Close 关闭连接
func (c *gwsConn) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.close()
	c.waitSendDone()

	c.writeMu.Lock()
	_ = c.conn.WriteClose(1000, nil)
	c.writeMu.Unlock()

	return c.conn.NetConn().Close()
}

// SetReadDeadline 设置读取超时
func (c *gwsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *gwsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// handleBinaryMessage 处理二进制消息
func (c *gwsConn) handleBinaryMessage(data []byte) {
	for len(data) > 0 {
		msg, n, err := c.codec.Decode(data)
		if err != nil {
			if c.server != nil {
				c.server.logger.Errorf("decode error: %v", err)
			}
			return
		}
		if n == 0 {
			break
		}
		if c.server != nil {
			c.server.handleMessage(c, msg)
		}
		data = data[n:]
	}
}

// handleTextMessage 处理文本消息
func (c *gwsConn) handleTextMessage(data []byte) {
	msg := NewMessage(0, data)
	if c.server != nil {
		c.server.handleMessage(c, msg)
	}
}

// gwsClientConn gws客户端连接
type gwsClientConn struct {
	*gwsConn
	client *Client
}

// handleBinaryMessage 处理客户端二进制消息
func (c *gwsClientConn) handleBinaryMessage(data []byte) {
	for len(data) > 0 {
		msg, n, err := c.client.codec.Decode(data)
		if err != nil {
			if c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}
		if n == 0 {
			break
		}
		c.client.handleMessage(c, msg)
		data = data[n:]
	}
}

// handleTextMessage 处理客户端文本消息
func (c *gwsClientConn) handleTextMessage(data []byte) {
	msg := NewMessage(0, data)
	c.client.handleMessage(c, msg)
}
