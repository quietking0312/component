package gogroup

import (
	"fmt"
	"runtime"
	"time"
)

/*
协程池， 用于控制协程数量， 协程启用过多，则阻塞 任务
*/

type Job func(data TaskData)

type TaskData map[interface{}]interface{}

type PanicFunc func(out interface{})

type GoGroup struct {
	num       int
	c         chan struct{}
	panicFunc PanicFunc
}

func NewGoGroup(n int) *GoGroup {
	if n <= 0 {
		return nil
	}
	gg := &GoGroup{
		num: n,
		c:   make(chan struct{}, n),
	}
	return gg
}

func (g *GoGroup) Run(data TaskData, job Job) error {
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case g.c <- struct{}{}:
	case <-timer.C:
		return fmt.Errorf("timeout")
	}
	// fileName, funName, line := runFuncName()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if g.panicFunc == nil {
					fmt.Printf("%v\n", r)
					for i := 1; ; i++ {
						pc, file, line, ok := runtime.Caller(i)
						if !ok {
							break
						}
						f := runtime.FuncForPC(pc)
						fmt.Printf("%s %s:%d(0x%x)\n", file, f.Name(), line, pc)
					}

				} else {
					g.panicFunc(r)
				}
			}
		}()
		job(data)
		<-g.c
	}()
	return nil
}

func (g *GoGroup) SetLogger(fun PanicFunc) {
	g.panicFunc = fun
}
