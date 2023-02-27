package mtypes

import (
	"fmt"
	"testing"
)

func TestNewPtr(t *testing.T) {
	a := struct {
		A *string
		B *int
	}{
		A: NewPtr[string]("hello"),
		B: NewPtr[int](55),
	}
	fmt.Println(*a.A)
	fmt.Println(*a.B)

	c := NewPtr("hello")
	fmt.Println(Value(c))
}
