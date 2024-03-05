package mtool

import (
	"fmt"
	"testing"
)

func TestNewPtr(t *testing.T) {
	a := struct {
		A *string
		B *int
	}{
		A: Ptr[string]("hello"),
		B: Ptr[int](55),
	}
	fmt.Println(*a.A)
	fmt.Println(*a.B)

	c := Ptr(a)
	fmt.Println(Value(c))

}
