package msock

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// wsClientConn WebSocket客户端连接
type wsClientConn struct {
	*wsConn
	client *Client
}

func (c *wsClientConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.client.logger.Error(fmt.Sprintf("panic in client ws readLoop: %v", r))
		}
		c.Close()
		if c.client.handlers.onDisconnect != nil {
			c.client.handlers.onDisconnect(c)
		}
	}()

	for {
		if c.IsClosed() {
			return
		}

		if c.client.config.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.client.config.ReadTimeout))
		}

		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if !c.IsClosed() && c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			c.handleBinaryMessage(data)
		case websocket.TextMessage:
			c.handleTextMessage(data)
		case websocket.CloseMessage:
			return
		}
	}
}

func (c *wsClientConn) handleBinaryMessage(data []byte) {
	codec := c.client.codec
	headerSize := codec.HeaderSize()
	for len(data) > 0 {
		if len(data) < headerSize {
			if c.client.handlers.onError != nil {
				c.client.handlers.onError(fmt.Errorf("ws client frame too short: %d < %d", len(data), headerSize))
			}
			return
		}
		routeID, bodyLen, err := codec.DecodeHeader(data[:headerSize])
		if err != nil {
			if c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}
		end := headerSize + bodyLen
		if len(data) < end {
			if c.client.handlers.onError != nil {
				c.client.handlers.onError(fmt.Errorf("ws client frame incomplete: need %d, have %d", end, len(data)))
			}
			return
		}
		msg, err := codec.DecodeBody(routeID, data[headerSize:end])
		if err != nil {
			if c.client.handlers.onError != nil {
				c.client.handlers.onError(err)
			}
			return
		}
		c.client.handleMessage(c, msg)
		data = data[end:]
	}
}

func (c *wsClientConn) handleTextMessage(data []byte) {
	msg := NewMessage(0, data)
	c.client.handleMessage(c, msg)
}
