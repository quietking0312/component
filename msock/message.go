package msock

import (
	"encoding/binary"
	"fmt"
	"sync"
)

// DefaultMessage 默认消息实现
type DefaultMessage struct {
	routeID uint32
	data    []byte
}

// NewMessage 创建新消息
func NewMessage(routeID uint32, data []byte) *DefaultMessage {
	return &DefaultMessage{
		routeID: routeID,
		data:    data,
	}
}

// RouteID 返回路由ID
func (m *DefaultMessage) RouteID() uint32 {
	return m.routeID
}

// Data 返回消息数据
func (m *DefaultMessage) Data() []byte {
	return m.data
}

// SetData 设置消息数据
func (m *DefaultMessage) SetData(data []byte) {
	m.data = data
}

// messagePool 消息对象池
var messagePool = sync.Pool{
	New: func() interface{} {
		return &DefaultMessage{}
	},
}

// AcquireMessage 从池中获取消息对象
func AcquireMessage() *DefaultMessage {
	return messagePool.Get().(*DefaultMessage)
}

// ReleaseMessage 将消息对象放回池中
func ReleaseMessage(msg *DefaultMessage) {
	msg.routeID = 0
	msg.data = msg.data[:0]
	messagePool.Put(msg)
}

// ========== SimpleCodec ==========

// SimpleCodec 简单编解码器
// 包格式: [4字节totalLen(大端)] + [4字节routeID(大端)] + [body]
// header = 8字节，totalLen 包含自身
type SimpleCodec struct {
	maxPacketSize int
}

// NewSimpleCodec 创建简单编解码器，maxPacketSize 默认 64KB
func NewSimpleCodec(maxPacketSize ...int) *SimpleCodec {
	maxSize := 64 * 1024
	if len(maxPacketSize) > 0 && maxPacketSize[0] > 0 {
		maxSize = maxPacketSize[0]
	}
	return &SimpleCodec{maxPacketSize: maxSize}
}

// Encode 编码消息
func (c *SimpleCodec) Encode(msg Message) ([]byte, error) {
	body := msg.Data()
	totalLen := 8 + len(body)
	if totalLen > c.maxPacketSize {
		return nil, fmt.Errorf("packet too large: %d > %d", totalLen, c.maxPacketSize)
	}

	buf := make([]byte, totalLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(totalLen))
	binary.BigEndian.PutUint32(buf[4:8], msg.RouteID())
	copy(buf[8:], body)
	return buf, nil
}

// HeaderSize 返回固定 header 大小
func (c *SimpleCodec) HeaderSize() int { return 8 }

// DecodeHeader 解析 header，返回 routeID 和 body 长度
func (c *SimpleCodec) DecodeHeader(header []byte) (routeID uint32, bodyLen int, err error) {
	totalLen := int(binary.BigEndian.Uint32(header[0:4]))
	if totalLen < 8 {
		return 0, 0, fmt.Errorf("invalid packet length: %d", totalLen)
	}
	if totalLen > c.maxPacketSize {
		return 0, 0, fmt.Errorf("packet too large: %d > %d", totalLen, c.maxPacketSize)
	}
	routeID = binary.BigEndian.Uint32(header[4:8])
	bodyLen = totalLen - 8
	return routeID, bodyLen, nil
}

// DecodeBody 将 body 解析为消息
func (c *SimpleCodec) DecodeBody(routeID uint32, body []byte) (Message, error) {
	data := make([]byte, len(body))
	copy(data, body)
	return NewMessage(routeID, data), nil
}

// MaxPacketSize 返回最大包大小
func (c *SimpleCodec) MaxPacketSize() int { return c.maxPacketSize }

// ========== TLVCodec ==========

// TLVCodec TLV格式编解码器
// 包格式: [1字节Type] + [2字节Length(大端)] + [Value]
// header = 3字节，Length 为 body 长度
type TLVCodec struct {
	maxPacketSize int
}

// TLVMessage TLV消息
type TLVMessage struct {
	msgType byte
	data    []byte
}

// NewTLVCodec 创建TLV编解码器
func NewTLVCodec(maxPacketSize ...int) *TLVCodec {
	maxSize := 64 * 1024
	if len(maxPacketSize) > 0 && maxPacketSize[0] > 0 {
		maxSize = maxPacketSize[0]
	}
	return &TLVCodec{maxPacketSize: maxSize}
}

// Encode 编码TLV消息
func (c *TLVCodec) Encode(msg Message) ([]byte, error) {
	data := msg.Data()
	if len(data) > 0xFFFF {
		return nil, fmt.Errorf("data too large for TLV: %d", len(data))
	}

	var msgType byte
	if tlvMsg, ok := msg.(*TLVMessage); ok {
		msgType = tlvMsg.msgType
	}

	buf := make([]byte, 3+len(data))
	buf[0] = msgType
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(data)))
	copy(buf[3:], data)
	return buf, nil
}

// HeaderSize 返回固定 header 大小
func (c *TLVCodec) HeaderSize() int { return 3 }

// DecodeHeader 解析 header，返回 routeID 和 body 长度
func (c *TLVCodec) DecodeHeader(header []byte) (routeID uint32, bodyLen int, err error) {
	length := int(binary.BigEndian.Uint16(header[1:3]))
	if length > c.maxPacketSize {
		return 0, 0, fmt.Errorf("TLV packet too large: %d", length)
	}
	return uint32(header[0]), length, nil
}

// DecodeBody 将 body 解析为消息
func (c *TLVCodec) DecodeBody(routeID uint32, body []byte) (Message, error) {
	data := make([]byte, len(body))
	copy(data, body)
	return &TLVMessage{msgType: byte(routeID), data: data}, nil
}

// MaxPacketSize 返回最大包大小
func (c *TLVCodec) MaxPacketSize() int { return c.maxPacketSize }

// RouteID TLVMessage 的路由ID就是 msgType
func (m *TLVMessage) RouteID() uint32 { return uint32(m.msgType) }

// Data 返回数据
func (m *TLVMessage) Data() []byte { return m.data }

// SetData 设置数据
func (m *TLVMessage) SetData(data []byte) { m.data = data }

// MsgType 返回消息类型
func (m *TLVMessage) MsgType() byte { return m.msgType }

// ========== LineCodec ==========

// LineCodec 文本行编解码器，适用于文本协议（如 Telnet）
// 因行长度不固定，HeaderSize 返回 0，bufferedReader 走流式扫描路径。
type LineCodec struct {
	maxLineLength int
}

// NewLineCodec 创建文本行编解码器
func NewLineCodec(maxLineLength ...int) *LineCodec {
	maxLen := 4096
	if len(maxLineLength) > 0 && maxLineLength[0] > 0 {
		maxLen = maxLineLength[0]
	}
	return &LineCodec{maxLineLength: maxLen}
}

// Encode 编码文本行（末尾添加换行符）
func (c *LineCodec) Encode(msg Message) ([]byte, error) {
	data := msg.Data()
	if len(data) > c.maxLineLength {
		return nil, fmt.Errorf("line too long: %d", len(data))
	}
	if len(data) > 0 && data[len(data)-1] == '\n' {
		return data, nil
	}
	buf := make([]byte, len(data)+1)
	copy(buf, data)
	buf[len(data)] = '\n'
	return buf, nil
}

// HeaderSize 返回 0，表示使用流式扫描而非固定 header
func (c *LineCodec) HeaderSize() int { return 0 }

// DecodeHeader 不适用于 LineCodec，始终返回错误
func (c *LineCodec) DecodeHeader(header []byte) (uint32, int, error) {
	return 0, 0, fmt.Errorf("LineCodec does not support two-phase decoding")
}

// DecodeBody 不适用于 LineCodec
func (c *LineCodec) DecodeBody(routeID uint32, body []byte) (Message, error) {
	return nil, fmt.Errorf("LineCodec does not support two-phase decoding")
}

// ScanLine 扫描缓冲区中的一行，返回消息和已消费字节数；数据不足时返回 nil,0,nil
func (c *LineCodec) ScanLine(data []byte) (Message, int, error) {
	for i, b := range data {
		if i >= c.maxLineLength {
			return nil, 0, fmt.Errorf("line too long")
		}
		if b == '\n' {
			line := make([]byte, i)
			copy(line, data[:i])
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			return NewMessage(0, line), i + 1, nil
		}
	}
	return nil, 0, nil
}

// MaxPacketSize 返回最大行长度
func (c *LineCodec) MaxPacketSize() int { return c.maxLineLength }
