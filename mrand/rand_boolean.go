package mrand

import (
	"math/rand"
	"time"
)

// 该方法是为概率在运气特差的情况下 做补措施， golang rand 模块使用的 LCG 算法 周期短, 不适合该方法
// 当返回 false 时 将为基础概率增加一个值， 来促使更快的达成 true
type RandBoolean struct {
	initial    int64
	current    int64
	target     int64
	max        int64
	increment  int64
	rand       *rand.Rand
	randValues []int64
	idx        int
}

func NewRandBoolean(initial, target, max int64) *RandBoolean {
	r := &RandBoolean{
		initial:    initial,
		current:    initial,
		target:     target,
		max:        max,
		randValues: make([]int64, max),
	}
	if r.initial < target {
		r.increment = (target - initial) / 10
	}
	r.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range r.randValues {
		r.randValues[i] = r.rand.Int63n(max)
	}
	return r
}

func (r *RandBoolean) attack() bool {
	if r.idx >= len(r.randValues) {
		r.randValues = make([]int64, r.max)
		for i := range r.randValues {
			r.randValues[i] = r.rand.Int63n(r.max)
		}
		r.idx = 0
	}
	n := r.randValues[r.idx]
	r.idx++
	if n < r.current {
		r.current = r.initial
		return true
	} else {
		if r.current < r.max {
			r.current += r.increment
		} else {
			r.current = r.max
		}
		return false
	}
}
