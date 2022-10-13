package mjson

import (
	"encoding/json"
	"strings"
)

// Unmarshal json 包 在解析int64 会有精度丢失问题
func Unmarshal(data []byte, v interface{}) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	return dec.Decode(v)
}

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
