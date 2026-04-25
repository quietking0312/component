package mmath

import (
	"math"
)

// RSqrt 求 1 / 平方根 x (快速逆平方根)
func RSqrt(x float32) float32 {
	x2 := x
	i := math.Float32bits(x)
	i = 0x5f3759df - (i >> 1)
	x2 = math.Float32frombits(i)
	x2 = x2 * (1.5 - (0.5 * x * x2 * x2))
	return x2
}

// Sqrt 牛顿求平方根
func Sqrt(x float64) float64 {
	if x < 0 {
		return math.NaN()
	}
	if x == 0 {
		return 0
	}
	x2 := x
	for math.Abs(x2*x2-x) > 0.00001 { // 精度
		x2 = (x2 + x/x2) / 2
	}
	return x2
}
