package mbar

import (
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

// https://github.com/qianlnk/pgbar/blob/master/bar.go

type Bar struct {
	mu        sync.Mutex
	prefix    string         // 前置描述
	total     int            // 总量
	width     int            // 宽度
	advance   chan bool      // 是否刷新进度条
	done      chan bool      // 是否完成
	currents  map[string]int // 一段时间内每个时间点完成量
	current   int            // 当前完成量
	rate      int            // 进度百分比
	speed     int            // 速度
	cost      int            // 耗时
	estimate  int            // 预计剩余完成时间
	closed    bool           // 是否结束
	before    int
	BarRender *BarRender // 进度条渲染方式
	speedUnit func(speed float64) (float64, string)
	closeOnce sync.Once
}

type BarRender struct {
	bar1 string // 已完成
	bar2 string // 未完成
	fast int    // 速度快的阈值
	slow int    // 速度慢的阈值
}

func (b *BarRender) initBarRender(width int) {
	for i := 0; i < width; i++ {
		b.bar1 += "="
		b.bar2 += "-"
	}
}

func NewBar(total int) *Bar {
	if total <= 0 {
		panic("mbar: total must be greater than 0")
	}
	bar := &Bar{
		prefix:   "",
		total:    total,
		width:    100,
		advance:  make(chan bool),
		done:     make(chan bool),
		currents: make(map[string]int),
		current:  0,
		BarRender: &BarRender{
			fast: 20,
			slow: 5,
		},
		speedUnit: func(speed float64) (float64, string) {
			const baseKB = 1024
			const baseMB = 1024 * baseKB
			const baseGB = 1024 * baseMB
			if baseMB > speed && speed >= baseKB {
				return speed / baseKB, "KB/s"
			} else if baseGB > speed && speed >= baseMB {
				return speed / baseMB, "MB/s"
			} else if speed >= baseGB {
				return speed / baseGB, "GB/s"
			} else {
				return speed, "B/s"
			}
		},
	}
	bar.BarRender.initBarRender(bar.width)
	go bar.updateCost()
	go bar.run()
	return bar
}

func (b *Bar) Add(n ...int) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	step := 1
	if len(n) > 0 {
		step = n[0]
	}
	b.current += step
	lastRate := b.rate
	lastSpeed := b.speed
	b.count()
	needAdvance := lastRate != b.rate || lastSpeed != b.speed
	isDone := b.total-b.current <= 0
	b.mu.Unlock()

	if needAdvance {
		select {
		case b.advance <- true:
		default:
		}
	}
	if isDone {
		b.closeOnce.Do(func() {
			b.mu.Lock()
			b.closed = true
			b.mu.Unlock()
			close(b.done)
			select {
			case b.advance <- false:
			default:
			}
			close(b.advance)
		})
	}
}

func (b *Bar) Closed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

func (b *Bar) count() {
	now := time.Now()
	nowKey := now.Format("20060102150405")
	befKey := now.Add(time.Minute * -1).Format("20060102150405")
	b.currents[nowKey] = b.current
	if v, ok := b.currents[befKey]; ok {
		b.before = v
	}
	delete(b.currents, befKey)
	// 保留2位小数 所以需要 * 100
	b.rate = b.current * 100 / b.total
	if b.cost == 0 {
		b.speed = b.current * 100
	} else if b.before == 0 {
		b.speed = b.current * 100 / b.cost
	} else {
		b.speed = (b.current - b.before) * 100 / 60
	}
	if b.speed != 0 {
		b.estimate = (b.total - b.current) * 100 / b.speed
	}
}

func (b *Bar) updateCost() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.mu.Lock()
			if b.closed {
				b.mu.Unlock()
				return
			}
			b.cost++
			b.count()
			b.mu.Unlock()
			select {
			case b.advance <- true:
			default:
			}
		case <-b.done:
			return
		}
	}
}

func (b *Bar) run() {
	for {
		select {
		case isClose := <-b.advance:
			if isClose {
				fmt.Printf("\r%s", b.barMsg())
			} else {
				return
			}
		}
	}
}

func (b *Bar) barMsg() string {
	speedNumber, unit := b.speedUnit(0.01 * float64(b.speed))
	prefix := fmt.Sprintf("%s", b.prefix)
	rate := fmt.Sprintf("%3d%%", b.rate)
	speed := fmt.Sprintf("%3.2f %s ps", speedNumber, unit)
	cost := (time.Duration(b.cost) * time.Second).String()
	estimate := (time.Duration(b.estimate) * time.Second).String()
	ct := fmt.Sprintf(" (%d/%d)", b.current, b.total)
	barLen := b.width - utf8.RuneCountInString(prefix) - len(rate) - len(speed) - len(cost) - len(estimate) - len(ct) - 10
	if barLen < 0 {
		barLen = 0
	}
	bar1len := barLen * b.rate / 100
	if bar1len < 0 {
		bar1len = 0
	}
	bar2len := barLen - bar1len
	if bar2len < 0 {
		bar2len = 0
	}

	realBar1Len := bar1len
	if realBar1Len > len(b.BarRender.bar1) {
		realBar1Len = len(b.BarRender.bar1)
	}
	realBar1 := b.BarRender.bar1[:realBar1Len]
	var realBar2 string
	if bar2len > 0 {
		realBar2Len := bar2len
		if realBar2Len > len(b.BarRender.bar2)+1 {
			realBar2Len = len(b.BarRender.bar2) + 1
		}
		if realBar2Len > 1 {
			realBar2 = ">" + b.BarRender.bar2[:realBar2Len-1]
		} else if realBar2Len == 1 {
			realBar2 = ">"
		}
	}
	msg := fmt.Sprintf(`%s %s%s [%s%s] %s %s in: %s`, prefix, rate, ct, realBar1, realBar2, speed, cost, estimate)
	switch {
	case b.speed <= b.BarRender.slow*100:
		return msg
	case b.speed > b.BarRender.slow*100 && b.speed < b.BarRender.fast*100:
		return msg
	default:
		return msg
	}
}
