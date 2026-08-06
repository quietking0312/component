package msock

import (
	"fmt"
	"net"
	"time"

	"github.com/lxzan/gws"
)

// gwsConn gws WebSocket连接实现
type gwsConn struct {
	*baseConn
	conn   *gws.Conn
	server *Server
	codec  Codec
}

// newGWSConn 创建gws连接
func newGWSConn(conn *gws.Conn, server *Server, codec Codec, id string) *gwsConn {
	return &gwsConn{
		baseConn: newBaseConn(ConnTypeGWS, id),
		conn:     conn,
		server:   server,
		codec:    codec,
	}
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

// SendBytes 异步发送二进制消息
func (c *gwsConn) SendBytes(data []byte) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	buf := make([]byte, len(data))
	copy(buf, data)
	c.conn.WriteAsync(gws.OpcodeBinary, buf, func(err error) {
		if err != nil && !c.IsClosed() {
			if c.server != nil {
				c.server.logger.Error(fmt.Sprintf("gws async write error: %v, conn: %s", err, c.ID()))
			}
			c.Close()
		}
	})
	return nil
}

// SendText 异步发送文本消息
func (c *gwsConn) SendText(text string) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	c.conn.WriteAsync(gws.OpcodeText, []byte(text), func(err error) {
		if err != nil && !c.IsClosed() {
			if c.server != nil {
				c.server.logger.Error(fmt.Sprintf("gws async write text error: %v, conn: %s", err, c.ID()))
			}
			c.Close()
		}
	})
	return nil
}

// Close 关闭连接
func (c *gwsConn) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.close()
	_ = c.conn.WriteClose(1000, nil)
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

// handleBinaryMessage 处理二进制消息（帧内两阶段解码）
func (c *gwsConn) handleBinaryMessage(data []byte) {
	codec := c.codec
	headerSize := codec.HeaderSize()
	for len(data) > 0 {
		if len(data) < headerSize {
			if c.server != nil {
				c.server.logger.Error(fmt.Sprintf("gws frame too short: %d < %d", len(data), headerSize))
			}
			return
		}
		routeID, bodyLen, err := codec.DecodeHeader(data[:headerSize])
		if err != nil {
			if c.server != nil {
				c.server.logger.Error(fmt.Sprintf("gws decode header error: %v", err))
			}
			return
		}
		end := headerSize + bodyLen
		if len(data) < end {
			if c.server != nil {
				c.server.logger.Error(fmt.Sprintf("gws frame incomplete: need %d, have %d", end, len(data)))
			}
			return
		}
		msg, err := codec.DecodeBody(routeID, data[headerSize:end])
		if err != nil {
			if c.server != nil {
				c.server.logger.Error(fmt.Sprintf("gws decode body error: %v", err))
			}
			return
		}
		if c.server != nil {
			c.server.handleMessage(c, msg)
		}
		data = data[end:]
	}
}

// handleTextMessage 处理文本消息
func (c *gwsConn) handleTextMessage(data []byte) {
	msg := NewMessage(0, data)
	if c.server != nil {
		c.server.handleMessage(c, msg)
	}
}
