package merr

import (
	"errors"
	"fmt"
	"testing"
)

func TestMErr_GetMsg(t *testing.T) {
	a := NewMErr("1", "hello")
	fmt.Println(a.Code())
	fmt.Println(a.Error())
	fmt.Println(errors.Unwrap(a))
}
