package mmath

import (
	"math"
)

// 求 1 / 平方根 x
func mRSqrt(x float32) float32 {
	x2 := x
	i := math.Float32bits(x)
	i = 0x5f3759df - (i >> 1)
	x2 = math.Float32frombits(i)
	x2 = x2 * (1.5 - (0.5 * x * x2 * x2))
	return x2
}

// 牛顿求平方根
func mSqrt(x float64) float64 {
	x2 := x
	for (x2*x2 - x) > 0.00001 { // 精度
		x2 = (x2 + x/x2) / 2
	}
	return x2
}
