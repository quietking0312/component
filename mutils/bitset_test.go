package utils

import (
	"fmt"
	"testing"
)

func TestNewBitSet64(t *testing.T) {
	b := NewBitSet64(1024)
	fmt.Println(5, b.Get(5))
	b.Set(5)
	fmt.Println(5, b.Get(5))
	b.Set(5)
	fmt.Println(5, b.Get(5))
	b.Clear(5)
	fmt.Println(5, b.Get(5))
	b.Clear(5)
	fmt.Println(5, b.Get(5))
	fmt.Println(4, b.Get(4))
	b.Toggle(4)
	fmt.Println(4, b.Get(4))
	b.Toggle(4)
	fmt.Println(4, b.Get(4))
	b.Set(1023)
}
