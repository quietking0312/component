package mtool

import (
	"fmt"
	"reflect"
	"testing"
)

func TestCopy(t *testing.T) {
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
