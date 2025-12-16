package mtool

import (
	"fmt"
	"testing"
)

func TestNewNode(t *testing.T) {
	slt := NewSkipList[int, any](func(a, b int) int {
		if a < b {
			return Less
		} else if a == b {
			return Equal
		}
		return Greater
	})
	for i := 0; i < 1000; i++ {
		slt.Insert(i, fmt.Sprintf("w%d", i))
	}

	slt.PrintSkipList()
	fmt.Println(slt.Search(468))
}
