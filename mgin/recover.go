package mgin

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/quietking0312/component/mlog"
	"go.uber.org/zap"
)

type Middleware struct {
}

func Recover(recoveryFunc gin.RecoveryFunc) gin.HandlerFunc {
	DefaultErrorWriter := &PanicExceptionRecord{}
	return gin.RecoveryWithWriter(DefaultErrorWriter, recoveryFunc)
}

type PanicExceptionRecord struct {
}

func (p *PanicExceptionRecord) Write(b []byte) (n int, err error) {
	errStr := string(b)
	err = errors.New(errStr)
	mlog.Error("panic", zap.Error(err))
	return len(errStr), err
}
