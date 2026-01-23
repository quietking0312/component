package mpubsub

import (
	"bytes"
	"encoding/gob"
	"sync"
)

type GobParser struct {
	bufferPool sync.Pool
}

func NewGobPaser() *GobParser {
	return &GobParser{
		bufferPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
	}
}

func (g *GobParser) Decoder(msgBytes []byte, message any) error {
	buf := g.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(msgBytes)
	defer func() {
		g.bufferPool.Put(buf)
	}()
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(message)
}

func (g *GobParser) Encoder(message any) ([]byte, error) {
	buf := g.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		g.bufferPool.Put(buf)
	}()
	encoder := gob.NewEncoder(buf)
	if err := encoder.Encode(message); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}
