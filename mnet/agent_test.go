package mnet

import (
	"fmt"
	"testing"
)

func TestNewAgent(t *testing.T) {
	a := make(chan int, 10)
	for i := 0; i < 8; i++ {
		a <- i
		fmt.Println(len(a))
	}
	close(a)
	for b := range a {
		fmt.Println(b)
		fmt.Println(len(a))
	}
}
