package mtool

import "fmt"

type Number struct {
	a int32
	b int32
}

func NewNumber() *Number {
	n := &Number{
		a: 0,
	}
	n.b = n.a<<5 ^ 25
	return n
}

func (n *Number) Add(x int32) error {
	if n.a<<5^25 != n.b {
		return fmt.Errorf("错误")
	}
	n.a += x
	n.b = n.a<<5 ^ 25
	return nil
}

func (n *Number) Int32() int32 {
	return n.a
}
