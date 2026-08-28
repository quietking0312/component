package msock

import (
	"fmt"
	"io"
	"net"
	"time"
)

// tcpConn TCP连接实现
type tcpConn struct {
	*baseConn
	net.Conn
	server       *Server
	codec        Codec
	writeTimeout time.Duration // 0 表示使用 server.config 或默认值
}

// newTCPConn 创建TCP连接
func newTCPConn(conn net.Conn, server *Server) *tcpConn {
	c := &tcpConn{
		baseConn: newBaseConn(ConnTypeTCP, server.config.IDGenerator()),
		Conn:     conn,
		server:   server,
		codec:    server.codec,
	}
	c.initSendQueue(128)
	c.sendWg.Add(1)
	go c.sendLoop()
	return c
}

// LocalAddr 返回本地地址
func (c *tcpConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

// RemoteAddr 返回远程地址
func (c *tcpConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// Send 发送消息
func (c *tcpConn) Send(msg Message) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}

	return c.SendBytes(data)
}

// sendLoop TCP发送协程
func (c *tcpConn) sendLoop() {
	defer c.sendWg.Done()
	for data := range c.sendCh {
		writeTimeout := c.writeTimeout
		if writeTimeout <= 0 {
			if c.server != nil && c.server.config.WriteTimeout > 0 {
				writeTimeout = c.server.config.WriteTimeout
			} else {
				writeTimeout = 10 * time.Second
			}
		}
		if err := c.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			c.Close()
			return
		}
		if _, err := c.Conn.Write(data); err != nil {
			c.Close()
			return
		}
	}
}

// SendBytes 发送原始字节
func (c *tcpConn) SendBytes(data []byte) error {
	return c.sendAsync(data)
}

// Close 关闭连接
func (c *tcpConn) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.close()
	c.waitSendDone()
	return c.Conn.Close()
}

// SetReadDeadline 设置读取超时
func (c *tcpConn) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *tcpConn) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}

// readLoop 读取循环（两阶段解码）
func (c *tcpConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.server.logger.Error(fmt.Sprintf("panic in readLoop: %v", r))
		}
		c.Close()
	}()

	codec := c.server.codec
	headerSize := codec.HeaderSize()
	header := make([]byte, headerSize)

	for {
		if c.IsClosed() {
			return
		}

		if c.server.config.ReadTimeout > 0 {
			if err := c.SetReadDeadline(time.Now().Add(c.server.config.ReadTimeout)); err != nil {
				c.server.logger.Warn(fmt.Sprintf("set read deadline error: %v", err))
				return
			}
		}

		var msg Message

		if headerSize == 0 {
			lc, ok := codec.(*LineCodec)
			if !ok {
				c.server.logger.Error("codec headerSize=0 but is not LineCodec")
				return
			}
			line, err := scanLine(c.Conn, lc.MaxPacketSize())
			if err != nil {
				if !c.IsClosed() {
					c.server.logger.Debug(fmt.Sprintf("read error: %v", err))
				}
				return
			}
			msg = NewMessage(0, line)
		} else {
			// 第一阶段：读 header
			if _, err := io.ReadFull(c.Conn, header); err != nil {
				if !c.IsClosed() {
					c.server.logger.Debug(fmt.Sprintf("read header error: %v", err))
				}
				return
			}

			routeID, seq, bodyLen, err := codec.DecodeHeader(header)
			if err != nil {
				c.server.logger.Error(fmt.Sprintf("decode header error: %v", err))
				return
			}

			// 第二阶段：读 body
			var body []byte
			if bodyLen > 0 {
				body = acquireBody(bodyLen)
				if _, err = io.ReadFull(c.Conn, body); err != nil {
					releaseBody(body)
					if !c.IsClosed() {
						c.server.logger.Debug(fmt.Sprintf("read body error: %v", err))
					}
					return
				}
			}

			msg, err = codec.DecodeBody(routeID, seq, body)
			if bodyLen > 0 {
				releaseBody(body)
			}
			if err != nil {
				c.server.logger.Error(fmt.Sprintf("decode body error: %v", err))
				return
			}
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					c.server.logger.Error(fmt.Sprintf("panic in handler: %v, conn: %s", r, c.ID()))
				}
			}()
			c.server.handleMessage(c, msg)
		}()
	}
}

// scanLine 从 conn 逐块读取直到遇到换行符，返回不含换行的行内容
func scanLine(conn net.Conn, maxLen int) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 64)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			return nil, err
		}
		for i := 0; i < n; i++ {
			if len(buf) >= maxLen {
				return nil, fmt.Errorf("line too long")
			}
			if tmp[i] == '\n' {
				line := buf
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				return line, nil
			}
			buf = append(buf, tmp[i])
		}
	}
}
