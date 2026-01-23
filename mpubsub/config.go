package mpubsub

import (
	"fmt"
	"sync/atomic"
	"time"
)

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

type Metrics struct {
	ActiveWorkers atomic.Int64 // 活跃worker 数
}

type SubGroupOption struct {
	WorkNum    int           // 任务数量 启用的协程分发数量 最低 1
	MinWorkers int           // 最小协程数
	MaxWorkers int           // 最大协程数
	RetryCount int           // 重试次数
	RetryDelay time.Duration // 重试间隔
}

func WithSubGroupConfig[T any](config SubGroupOption) ChannelOption[T] {
	return func(g *SubGroup[T]) {
		g.config = config
	}
}
