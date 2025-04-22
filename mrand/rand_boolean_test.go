package mrand

import (
	"fmt"
	"testing"
)

func TestNewRandBoolean(t *testing.T) {
	r := NewRandBoolean(1500, 1600, 10000)

	total := 1000000
	c := 0
	for i := 0; i < total; i++ {
		if r.attack() {
			c++
		}
	}
	fmt.Printf("%.4f%%\n", float64(c)/float64(total)*100)
}
