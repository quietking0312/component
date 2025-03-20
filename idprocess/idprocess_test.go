package idprocess

import (
	"fmt"
	"testing"
)

func TestNewIdProcess(t *testing.T) {
	w := NewIdProcess(1000009)
	for i := 0; i < 100000; i++ {
		fmt.Println(w.GetId())
	}
}
