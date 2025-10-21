package mtool

import (
	"fmt"
	"testing"
)

func TestNewNumber(t *testing.T) {
	a := NewNumber()
	fmt.Println(a)
	for i := 0; i < 10; i++ {
		a.Add(1000)
	}
	fmt.Println(a)
}
