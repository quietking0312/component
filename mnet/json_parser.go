package mnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type JSONParser struct {
}

type Msg struct {
	Id   []byte
	Data []byte
}

func (p *JSONParser) Unmarshal(b []byte) error {
	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		fmt.Println(err)
		return err
	}
	for msgId, message := range m {
		msg := &Msg{
			Id:   []byte(msgId),
			Data: message,
		}
		p.Route(msg)
	}
	return nil
}

func (p *JSONParser) Marshal(data any) ([]byte, error) {
	var msgId string
	switch d := data.(type) {
	case interface{ GetMsgId() string }:
		msgId = d.GetMsgId()
	default:
		msgType := reflect.TypeOf(data)
		if msgType == nil || msgType.Kind() != reflect.Ptr {
			return nil, errors.New("json message pointer required")
		}
		msgId = msgType.Elem().Name()
	}

	m := map[string]interface{}{msgId: data}
	return json.Marshal(m)
}

func (p *JSONParser) Route(msg *Msg) {
	fmt.Println(string(msg.Id), string(msg.Data))
}
