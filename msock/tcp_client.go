package msock

import (
	"fmt"
	"io"
	"time"
)

// tcpClientConn TCP客户端连接
type tcpClientConn struct {
	*tcpConn
	client *Client
}

func (c *tcpClientConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.client.logger.Error(fmt.Sprintf("panic in client readLoop: %v", r))
		}
		c.Close()
		c.client.notifyDisconnect(c)
	}()

	codec := c.client.codec
	headerSize := codec.HeaderSize()
	header := make([]byte, headerSize)

	for {
		if c.IsClosed() {
			return
		}

		if c.client.config.ReadTimeout > 0 {
			if err := c.SetReadDeadline(time.Now().Add(c.client.config.ReadTimeout)); err != nil {
				return
			}
		}

		var (
			msg Message
			err error
		)

		if headerSize == 0 {
			lc, ok := codec.(*LineCodec)
			if !ok {
				return
			}
			line, e := scanLine(c.Conn, lc.MaxPacketSize())
			if e != nil {
				if !c.IsClosed() && c.client.handlers.onError != nil {
					c.client.handlers.onError(e)
				}
				return
			}
			msg = NewMessage(0, line)
		} else {
			if _, e := io.ReadFull(c.Conn, header); e != nil {
				if !c.IsClosed() && c.client.handlers.onError != nil {
					c.client.handlers.onError(e)
				}
				return
			}

			routeID, seq, bodyLen, e := codec.DecodeHeader(header)
			if e != nil {
				if c.client.handlers.onError != nil {
					c.client.handlers.onError(e)
				}
				return
			}

			var body []byte
			if bodyLen > 0 {
				body = acquireBody(bodyLen)
				if _, e = io.ReadFull(c.Conn, body); e != nil {
					releaseBody(body)
					if !c.IsClosed() && c.client.handlers.onError != nil {
						c.client.handlers.onError(e)
					}
					return
				}
			}

			msg, err = codec.DecodeBody(routeID, seq, body)
			if bodyLen > 0 {
				releaseBody(body)
			}
			if err != nil {
				if c.client.handlers.onError != nil {
					c.client.handlers.onError(err)
				}
				return
			}
		}

		c.client.handleMessage(c, msg)
	}
}
