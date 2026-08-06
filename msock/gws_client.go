package msock

import (
	"fmt"

	"github.com/lxzan/gws"
)

// gwsClientConn gws客户端连接
type gwsClientConn struct {
	*gwsConn
	client *Client
}

// handleBinaryMessage 处理客户端二进制消息（帧内两阶段解码）
func (c *gwsClientConn) handleBinaryMessage(data []byte) {
	codec := c.client.codec
	headerSize := codec.HeaderSize()
	for len(data) > 0 {
		if len(data) < headerSize {
			if c.client.handlers.onError != nil {
				c.client.handlers.onError(fmt.Errorf("gws client frame too short: %d < %d", len(data), headerSize))
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
				c.client.handlers.onError(fmt.Errorf("gws client frame incomplete: need %d, have %d", end, len(data)))
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

// handleTextMessage 处理客户端文本消息
func (c *gwsClientConn) handleTextMessage(data []byte) {
	msg := NewMessage(0, data)
	c.client.handleMessage(c, msg)
}

// gwsClientEventHandler gws客户端事件处理器
type gwsClientEventHandler struct {
	client *Client
}

func (h *gwsClientEventHandler) OnOpen(socket *gws.Conn) {}

func (h *gwsClientEventHandler) OnClose(socket *gws.Conn, err error) {
	v, ok := socket.Session().Load("msock_conn")
	if !ok {
		return
	}
	conn := v.(*gwsClientConn)
	_ = conn.Close()
	if h.client.handlers.onDisconnect != nil {
		h.client.handlers.onDisconnect(conn)
	}
}

func (h *gwsClientEventHandler) OnPing(socket *gws.Conn, payload []byte) {
	if cfg := h.client.config.GWSConfig; cfg != nil && cfg.OnPing != nil {
		_ = socket.WritePong(cfg.OnPing(payload))
		return
	}
	_ = socket.WritePong(nil)
}

func (h *gwsClientEventHandler) OnPong(socket *gws.Conn, payload []byte) {
	if cfg := h.client.config.GWSConfig; cfg != nil && cfg.OnPong != nil {
		cfg.OnPong(payload)
	}
}

func (h *gwsClientEventHandler) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()

	v, ok := socket.Session().Load("msock_conn")
	if !ok {
		return
	}
	conn := v.(*gwsClientConn)

	switch message.Opcode {
	case gws.OpcodeBinary:
		func() {
			defer func() {
				if r := recover(); r != nil {
					h.client.logger.Error(fmt.Sprintf("panic in handler: %v", r))
				}
			}()
			conn.handleBinaryMessage(message.Bytes())
		}()
	case gws.OpcodeText:
		func() {
			defer func() {
				if r := recover(); r != nil {
					h.client.logger.Error(fmt.Sprintf("panic in handler: %v", r))
				}
			}()
			conn.handleTextMessage(message.Bytes())
		}()
	}
}
