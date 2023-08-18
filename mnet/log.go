package mnet

import "fmt"

type Log interface {
	Error(err error)
	Info(msg string)
}

var _log = &defaultLog{}

type defaultLog struct{}

func (log *defaultLog) Error(err error) {
	fmt.Println(err)
}

func (log *defaultLog) Info(msg string) {
	fmt.Println(msg)
}
