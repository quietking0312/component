package mnet

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/quietking0312/component/mnet/pb"
	"reflect"
	"strconv"
	"strings"
)

type ProtoParser struct {
}

func (p *ProtoParser) Unmarshal(b []byte) (*Msg, error) {
	msgId := binary.LittleEndian.Uint16(b[0:2])

	return &Msg{
		Id:   []byte(strconv.FormatInt(int64(msgId), 10)),
		Data: b[2:],
	}, nil
}

func (p *ProtoParser) Marshal(data any) ([]byte, error) {
	switch data.(type) {
	case proto.Message:
		msgType := reflect.TypeOf(data)
		id, ok := pb.S2C_value[strings.ToLower(msgType.Elem().Name())]
		if !ok {
			id, ok = pb.C2S_value[strings.ToLower(msgType.Elem().Name())]
		}
		if !ok {
			return nil, fmt.Errorf("%s not msgid", msgType.Elem().Name())
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

func (p *ProtoParser) Route(msg *Msg, a Agent) {
	var req pb.Ping
	err := proto.Unmarshal(msg.Data, &req)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(strconv.ParseInt(string(msg.Id), 10, 64))
	fmt.Println(string(req.GetArgs()), a.LocalAddr().String())
}
