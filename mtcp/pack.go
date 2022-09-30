package mtcp

type JsonHead struct {
	method string
}

func (head *JsonHead) Unmarshal([]byte) error {
	return nil
}

func (head *JsonHead) GetDataLength() int64 {
	return 0
}

func (head *JsonHead) GetMethod() string {
	return head.method
}

type JsonPack struct {
}

func (pack *JsonPack) GetHeadLength() uint8 {
	return 6
}

func (pack *JsonPack) Marshal(msg Msg) ([]byte, error) {
	return nil, nil
}

func (pack *JsonPack) UnmarshalMsg([]byte) (Msg, error) {
	return nil, nil
}

func (pack *JsonPack) UnmarshalHead([]byte) (IHead, error) {
	//h := &JsonHead{
	//	method: "hello",
	//}
	return nil, nil
}
