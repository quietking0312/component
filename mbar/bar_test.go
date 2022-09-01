package mbar

import (
	"testing"
	"time"
)

func TestNewBar(t *testing.T) {
	b := NewBar(100)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				b.Add(1)
			}
		}
	}()
	b.run()
}
