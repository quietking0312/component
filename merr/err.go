package merr

import (
	"fmt"
)

type Error interface {
	Code() string
	Error() string
	Unwrap() error
}

type MErr struct {
	code string
	msg  error
}

type BaseError func() MErr

func (b BaseError) Code() string {
	return b().code
}

func (b BaseError) Error() string {
	base := b()
	return fmt.Sprintf("Error %s: %s", base.code, base.msg.Error())
}

func (b BaseError) Unwrap() error {
	return b().msg
}

type MError struct {
	BaseError
}

func NewMErr(code string, format string, a ...any) Error {
	e := fmt.Errorf(format, a...)
	m := MError{}
	m.BaseError = func() MErr {
		return MErr{
			code: code,
			msg:  e,
		}
	}
	return m
}
