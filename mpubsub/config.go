package mpubsub

import "fmt"

type LoggerIface interface {
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
