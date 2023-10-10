package mjson

import (
	"bytes"
	"encoding/json"
)

// Unmarshal json 包 在解析int64 会有精度丢失问题
func Unmarshal(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
