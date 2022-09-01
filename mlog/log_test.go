package mlog

import (
	"fmt"
	"go.uber.org/zap"
	"testing"
)

func TestInitLog(t *testing.T) {
	_ = InitLog()
	Debug("hello, world")
	Error("world", zap.Error(fmt.Errorf("sss")))
}
