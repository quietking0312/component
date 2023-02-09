package mrand

import (
	"fmt"
	"testing"
)

func TestRandStr(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println(RandStr(5))
	}
}
