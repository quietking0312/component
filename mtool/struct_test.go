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
