package mrand

import (
	"math/rand"
	"runtime"
	"sync/atomic"
)

// golang rand 的封装
// 使用协程提前计算好 随机数
// 实际测试并没有比原版块,原版已经很快了， 这个版本增加了原子操作和取模
// 仅作参考
type MSource struct {
	source    rand.Source
	cache     []int64
	head      uint32 // 生产指针
	tail      uint32 // 消费指针
	size      uint32
	closeChan chan struct{}
}

var _ rand.Source = (*MSource)(nil)

func NewSource(seed int64, cacheSize uint32) *MSource {
	if cacheSize == 0 {
		cacheSize = 1024
	}
	s := &MSource{
		source:    rand.NewSource(seed),
		cache:     make([]int64, cacheSize),
		closeChan: make(chan struct{}),
		size:      cacheSize,
	}
	for i := 0; i < int(cacheSize/2); i++ {
		s.cache[i] = s.source.Int63()
		atomic.AddUint32(&s.head, 1)
	}
	go s.start()
	return s
}

func (m *MSource) start() {
	for {
		select {
		case <-m.closeChan:
			return
		default:
			head := atomic.LoadUint32(&m.head)
			tail := atomic.LoadUint32(&m.tail)
			if head-tail < m.size-1 {
				idx := head % m.size
				m.cache[idx] = m.source.Int63()
				atomic.AddUint32(&m.head, 1)
			} else {
				runtime.Gosched()
			}
		}
	}
}

func (m *MSource) Int63() int64 {
	for {
		head := atomic.LoadUint32(&m.head)
		tail := atomic.LoadUint32(&m.tail)
		if tail < head {
			idx := tail % m.size
			val := m.cache[idx]
			if atomic.CompareAndSwapUint32(&m.tail, tail, tail+1) {
				return val
			}
		} else {
			return m.source.Int63()
		}
	}
}

func (m *MSource) Seed(seed int64) {
	m.closeChan <- struct{}{}
	m.source.Seed(seed)
	atomic.StoreUint32(&m.head, 0)
	atomic.StoreUint32(&m.tail, 0)
	go m.start()
}
