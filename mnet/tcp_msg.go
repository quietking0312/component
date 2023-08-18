package mnet

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type MsgParser struct {
	msgLen       uint8
	littleEndian bool
}

func NewMsgParser(msgLen uint8) *MsgParser {
	p := new(MsgParser)
	p.msgLen = msgLen
	p.littleEndian = false
	return p
}

func (p *MsgParser) Read(conn io.ReadCloser) ([]byte, error) {
	head := make([]byte, p.msgLen)
	if _, err := io.ReadFull(conn, head); err != nil {
		return nil, err
	}
	var msgLength uint16
	if p.littleEndian {
		msgLength = binary.LittleEndian.Uint16(head[0:p.msgLen])
	} else {
		msgLength = binary.BigEndian.Uint16(head[0:p.msgLen])
	}
	msgData := make([]byte, msgLength)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return nil, err
	}
	return msgData, nil
}

func (p *MsgParser) Write(conn io.WriteCloser, data []byte) (int, error) {
	msgLength := len(data)
	if msgLength > math.MaxUint16-int(p.msgLen) {
		return 0, fmt.Errorf("message length > %d", math.MaxUint16-int(p.msgLen))
	}
	var msg = make([]byte, msgLength+int(p.msgLen))
	if p.littleEndian {
		binary.LittleEndian.PutUint16(msg, uint16(msgLength))
	} else {
		binary.BigEndian.PutUint16(msg, uint16(msgLength))
	}
	copy(msg[p.msgLen:], data)

	return conn.Write(msg)
}
