package main

import (
	"context"
	"golang.org/x/time/rate"
	"time"
)

func A() {
	//限速器
	// 每 100 秒产生1个令牌， 最多存储1个令牌
	limiter := rate.NewLimiter(rate.Every(time.Duration(100)*time.Second), 1)
	// 判断是否有可用的令牌
	limiter.Allow()
	// 阻塞 等待， 直到有可用的令牌
	limiter.Wait(context.Background())
	// 没有可用事件时 返回 reservation
	limiter.Reserve()
}

func main() {
}
