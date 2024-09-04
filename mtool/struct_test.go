package mtool

import (
	"fmt"
	"reflect"
	"testing"
)

type A struct {
	Name string `json:"name"`
	Age  int
}

type B struct {
	Name  string `json:"name"`
	Age   int
	Phone int
}

func TestCopyStruct(t *testing.T) {
	a := A{
		Age:  5,
		Name: "aaa",
	}
	b := B{}

	fmt.Println(CopyStruct(a, &b))
	fmt.Println(a)
	fmt.Println(b)
}

func TestCopyStruct2(t *testing.T) {
	a := A{
		Age:  5,
		Name: "aaa",
	}
	b := B{}

	fmt.Println(CopyStruct2(a, &b, func(srcFiled, dstFiled reflect.StructField) bool {
		if srcFiled.Tag.Get("json") == dstFiled.Tag.Get("json") && srcFiled.Tag.Get("json") != "" {
			return true
		}
		return false
	}))
	fmt.Println(a)
	fmt.Println(b)
}

type C struct {
	B
}

func TestCopyStruct3(t *testing.T) {
	a := A{
		Age:  5,
		Name: "aaa",
	}
	b := C{}

	fmt.Println(CopyStruct(a, &b))
	fmt.Println(a)
	fmt.Println(b)
}

type D struct {
	Age int64
}

func TestCopyStruct4(t *testing.T) {
	a := A{
		Age:  5,
		Name: "aaa",
	}
	b := D{}

	fmt.Println(CopyStruct(a, &b))
	fmt.Println(a)
	fmt.Println(b)
}

type E struct {
	A int
	B string
}

func (c *E) SetB(b string) {
	c.B = b
}

type F struct {
	E
}

func TestCopyStruct5(t *testing.T) {
	f := func(r any) {
		r = &r
		switch r.(type) {
		case interface{ SetB(s string) }:
			r.(interface{ SetB(s string) }).SetB("3")
			fmt.Println(111)
			fmt.Println(r)
		}
	}
	args := F{}
	args.A = 1
	args.B = "2"
	f(args)
	fmt.Println(args)
}

func TestC(t *testing.T) {
	a := make(map[int32][5]int)
	a[1] = [5]int{}
	v := a[1]
	v[2] = 3

	fmt.Println(a)
}
