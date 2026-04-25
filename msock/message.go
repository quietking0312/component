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

// ========== 内置编解码器 ==========

// SimpleCodec 简单编解码器实现
// 包格式: [4字节包头长度(大端)] + [4字节路由ID(大端)] + [数据]
type SimpleCodec struct {
	maxPacketSize int
}

// NewSimpleCodec 创建简单编解码器
// maxPacketSize: 最大包大小，默认 64KB
func NewSimpleCodec(maxPacketSize ...int) *SimpleCodec {
	maxSize := 64 * 1024 // 64KB
	if len(maxPacketSize) > 0 && maxPacketSize[0] > 0 {
		maxSize = maxPacketSize[0]
	}
	return &SimpleCodec{maxPacketSize: maxSize}
}

// Encode 编码消息
func (c *SimpleCodec) Encode(msg Message) ([]byte, error) {
	data := msg.Data()
	routeID := msg.RouteID()

	// 总长度 = 4(长度) + 4(路由ID) + len(数据)
	totalLen := 8 + len(data)
	if totalLen > c.maxPacketSize {
		return nil, fmt.Errorf("packet too large: %d > %d", totalLen, c.maxPacketSize)
	}

	buf := make([]byte, totalLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(totalLen))
	binary.BigEndian.PutUint32(buf[4:8], routeID)
	copy(buf[8:], data)

	return buf, nil
}

// Decode 解码消息
// 返回消息和已解码的字节数
func (c *SimpleCodec) Decode(data []byte) (Message, int, error) {
	if len(data) < 8 {
		return nil, 0, nil // 数据不足，等待更多数据
	}

	packetLen := int(binary.BigEndian.Uint32(data[0:4]))
	if packetLen > c.maxPacketSize {
		return nil, 0, fmt.Errorf("packet too large: %d > %d", packetLen, c.maxPacketSize)
	}

	if packetLen < 8 {
		return nil, 0, fmt.Errorf("invalid packet length: %d", packetLen)
	}

	if len(data) < packetLen {
		return nil, 0, nil // 数据不足，等待更多数据
	}

	routeID := binary.BigEndian.Uint32(data[4:8])
	msgData := make([]byte, packetLen-8)
	copy(msgData, data[8:packetLen])

	msg := NewMessage(routeID, msgData)
	return msg, packetLen, nil
}

// MaxPacketSize 返回最大包大小
func (c *SimpleCodec) MaxPacketSize() int {
	return c.maxPacketSize
}

// TLVCodec TLV格式编解码器
// 包格式: [1字节Type] + [2字节Length(大端)] + [Value]
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
	tlvMsg, ok := msg.(*TLVMessage)

	var msgType byte
	if ok {
		msgType = tlvMsg.msgType
	}

	if len(data) > 0xFFFF {
		return nil, fmt.Errorf("data too large for TLV: %d", len(data))
	}

	buf := make([]byte, 3+len(data))
	buf[0] = msgType
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(data)))
	copy(buf[3:], data)

	return buf, nil
}

// Decode 解码TLV消息
func (c *TLVCodec) Decode(data []byte) (Message, int, error) {
	if len(data) < 3 {
		return nil, 0, nil
	}

	msgType := data[0]
	length := int(binary.BigEndian.Uint16(data[1:3]))

	if length > c.maxPacketSize {
		return nil, 0, fmt.Errorf("TLV packet too large: %d", length)
	}

	if len(data) < 3+length {
		return nil, 0, nil
	}

	msgData := make([]byte, length)
	copy(msgData, data[3:3+length])

	msg := &TLVMessage{
		msgType: msgType,
		data:    msgData,
	}
	return msg, 3 + length, nil
}

// MaxPacketSize 返回最大包大小
func (c *TLVCodec) MaxPacketSize() int {
	return c.maxPacketSize
}

// RouteID TLVMessage 的路由ID就是 msgType
func (m *TLVMessage) RouteID() uint32 {
	return uint32(m.msgType)
}

// Data 返回数据
func (m *TLVMessage) Data() []byte {
	return m.data
}

// SetData 设置数据
func (m *TLVMessage) SetData(data []byte) {
	m.data = data
}

// MsgType 返回消息类型
func (m *TLVMessage) MsgType() byte {
	return m.msgType
}

// ========== 文本行编解码器 ==========

// LineCodec 文本行编解码器，适用于文本协议（如 Telnet）
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

// Encode 编码文本行（添加换行符）
func (c *LineCodec) Encode(msg Message) ([]byte, error) {
	data := msg.Data()
	if len(data) > c.maxLineLength {
		return nil, fmt.Errorf("line too long: %d", len(data))
	}
	// 确保以换行符结尾
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return data, nil
}

// Decode 解码文本行
func (c *LineCodec) Decode(data []byte) (Message, int, error) {
	// 查找换行符
	for i, b := range data {
		if b == '\n' {
			line := make([]byte, i)
			copy(line, data[:i])
			// 移除可能的 \r
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			return NewMessage(0, line), i + 1, nil
		}
		if i >= c.maxLineLength {
			return nil, 0, fmt.Errorf("line too long")
		}
	}
	return nil, 0, nil // 等待更多数据
}

// MaxPacketSize 返回最大行长度
func (c *LineCodec) MaxPacketSize() int {
	return c.maxLineLength
}

// ========== JSON 编解码器 ==========

// JSONCodec JSON 格式编解码器
// 包格式: [4字节长度(大端)] + [JSON数据]
type JSONCodec struct {
	maxPacketSize int
	routeKey      string // JSON中用于路由的字段名
}

// NewJSONCodec 创建JSON编解码器
// routeKey: JSON中用于路由的字段名，如 "cmd" 或 "type"
func NewJSONCodec(routeKey string, maxPacketSize ...int) *JSONCodec {
	maxSize := 64 * 1024
	if len(maxPacketSize) > 0 && maxPacketSize[0] > 0 {
		maxSize = maxPacketSize[0]
	}
	return &JSONCodec{
		maxPacketSize: maxSize,
		routeKey:      routeKey,
	}
}

// Encode 编码JSON消息
func (c *JSONCodec) Encode(msg Message) ([]byte, error) {
	data := msg.Data()
	totalLen := 4 + len(data)
	if totalLen > c.maxPacketSize {
		return nil, fmt.Errorf("packet too large: %d", totalLen)
	}

	buf := make([]byte, totalLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(totalLen))
	copy(buf[4:], data)
	return buf, nil
}

// Decode 解码JSON消息
func (c *JSONCodec) Decode(data []byte) (Message, int, error) {
	if len(data) < 4 {
		return nil, 0, nil
	}

	packetLen := int(binary.BigEndian.Uint32(data[0:4]))
	if packetLen > c.maxPacketSize {
		return nil, 0, fmt.Errorf("packet too large: %d", packetLen)
	}

	if len(data) < packetLen {
		return nil, 0, nil
	}

	jsonData := make([]byte, packetLen-4)
	copy(jsonData, data[4:packetLen])

	// 简单解析获取路由ID（这里只是示例，实际需要解析JSON）
	// 生产环境建议使用 jsonparser 或类似库
	routeID := c.extractRouteID(jsonData)

	return NewMessage(routeID, jsonData), packetLen, nil
}

// extractRouteID 从JSON数据中提取路由ID（简化实现）
func (c *JSONCodec) extractRouteID(data []byte) uint32 {
	// 这是一个简化实现，生产环境应该使用标准JSON库
	// 这里仅作为示例
	return 0
}

// MaxPacketSize 返回最大包大小
func (c *JSONCodec) MaxPacketSize() int {
	return c.maxPacketSize
}
