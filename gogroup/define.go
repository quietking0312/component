package gogroup

import (
	"runtime"
)

// 获取协程调用函数
func runFuncName() (string, string, int) {
	pc := make([]uintptr, 1)
	runtime.Callers(3, pc)
	f := runtime.FuncForPC(pc[0])
	fileName, line := f.FileLine(pc[0])
	return fileName, f.Name(), line
}
