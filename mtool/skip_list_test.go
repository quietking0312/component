package mtool

import (
	"fmt"
	"testing"
)

func TestNewNode(t *testing.T) {
	slt := NewSkipList()
	slt.Insert(10, "hello")
	slt.Insert(10, "world")
	slt.PrintSkipList()
	fmt.Println(slt.Search(10))
}
