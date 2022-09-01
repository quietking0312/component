package merr

import (
	"fmt"
)

type MErr[I ~int | ~int8 | ~int32 | ~int64 | ~string] struct {
	code I
	msg  error
}

func NewMErr[I ~int | ~int8 | ~int32 | ~int64 | ~string](code I, format string, a ...any) MErr[I] {
	e := fmt.Errorf(format, a...)
	return MErr[I]{
		code: code,
		msg:  e,
	}
}

func (e MErr[I]) Code() I {
	return e.code
}

func (e MErr[I]) Error() string {
	return fmt.Sprintf("Error %v: %s", e.code, e.msg.Error())
}

func (e MErr[I]) Unwrap() error {
	return e.msg
}
