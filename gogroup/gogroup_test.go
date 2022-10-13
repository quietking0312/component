package gogroup

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewGoGroup(t *testing.T) {
	var wg = sync.WaitGroup{}
	g := NewGoGroup(2)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		A1(g)
		A2(g)
		wg.Done()
	}
	wg.Wait()
}

func A1(g *GoGroup) {
	g.Run(TaskData{}, func(data TaskData) {

		time.Sleep(5 * time.Second)
		a := []int{1}
		fmt.Println("A1")
		fmt.Println(a[2])
	})
}

func A2(g *GoGroup) {
	g.Run(TaskData{}, func(data TaskData) {
		time.Sleep(3 * time.Second)
		a := []int{1}
		fmt.Println("A2")
		fmt.Println(a[2])
	})

}
