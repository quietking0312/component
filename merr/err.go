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
	Code string
	Msg  error
}

type BaseError func() MErr

func (b BaseError) Code() string {
	return b().Code
}

func (b BaseError) Error() string {
	base := b()
	return fmt.Sprintf("Error %s: %s", base.Code, base.Msg.Error())
}

func (b BaseError) Unwrap() error {
	return b().Msg
}

type MError struct {
	BaseError
}

func NewMErr(code string, format string, a ...any) Error {
	e := fmt.Errorf(format, a...)
	m := MError{}
	m.BaseError = func() MErr {
		return MErr{
			Code: code,
			Msg:  e,
		}
	}
	return m
}
