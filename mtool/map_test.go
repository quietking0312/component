package mtool

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestNewMap(t *testing.T) {
	a := NewMap(5)
	r := rand.New(rand.NewSource(5))
	var m sync.WaitGroup
	for i := 0; i <= 5000; i++ {
		m.Add(1)
		go func(i int) {
			defer m.Done()
			v := r.Intn(5)
			a.Set(i, v)
		}(i)
	}
	m.Wait()
	a.Range(func(k, v any) bool {
		fmt.Printf("%v, %v\n", k, v)
		return true
	})
}

func TestNewMap2(t *testing.T) {
	a := NewMap(5)
	a.Set("5", 6)
	a.Set(5, "6")
	v1, _ := a.Get("5")
	fmt.Println(v1.(int))
	v2, _ := a.Get(5)
	fmt.Println(v2.(string))

	b := NewMap(10)
	b.Set(1, 1)
	b.Set(5, 5)
	b.Set(6, 6)
	b.Set(3, 3)
	b.Range(func(k, v any) bool {
		fmt.Println(k, v)
		return true
	})
}
