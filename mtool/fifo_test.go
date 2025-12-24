package mtool

import (
	"fmt"
	"testing"
)

func TestNewFIFO(t *testing.T) {
	a := NewFIFO[int](13)
	for i := 0; i < 100; i++ {
		a.Push(i)
	}
	fmt.Println(a.Data())
}
