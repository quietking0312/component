package mnet

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"strconv"
)

type ProtoParser struct {
	ValueMap map[string]int32
	MsgName  func(data any) string
}

func NewProtoParser(valueMap map[string]int32, msgName func(data any) string) *ProtoParser {
	return &ProtoParser{
		valueMap, msgName,
	}
}

func (p *ProtoParser) Unmarshal(b []byte) (*Msg, error) {
	return &Msg{
		Id:   strconv.Itoa(int(binary.LittleEndian.Uint16(b[0:2]))),
		Data: b[2:],
	}, nil
}

func (p *ProtoParser) Marshal(data any) ([]byte, error) {
	switch data.(type) {
	case proto.Message:
		msgName := p.MsgName(data)
		id, ok := p.ValueMap[p.MsgName(data)]
		if !ok {
			return nil, fmt.Errorf("%s not msgid", msgName)
		}

		b, err := proto.Marshal(data.(proto.Message))
		if err != nil {
			return nil, err
		}
		var msgData = make([]byte, len(b)+2)
		binary.LittleEndian.PutUint16(msgData, uint16(id))
		copy(msgData[2:], b)
		return msgData, err
	}
	return nil, fmt.Errorf("data type not proto.Message")
}
