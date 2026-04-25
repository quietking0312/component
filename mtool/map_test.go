package mtool

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestNewMap(t *testing.T) {
	a := NewOrderedMap[int, any]()
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
	for _, k := range a.Keys() {
		v, _ := a.Get(k)
		fmt.Printf("%v, %v\n", k, v)
	}
}
