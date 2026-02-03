package mrand

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestMSource_Int63(t *testing.T) {
	rand.New(NewSource(time.Now().Unix(), 0))
	x := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 3000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for x := 0; x < 500; x++ {
				rand.Int63()
			}
		}()
	}
	wg.Wait()
	fmt.Println(time.Now().Sub(x).String())
}

func TestMSource_Int632(t *testing.T) {
	rand.New(rand.NewSource(time.Now().Unix()))
	x := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 3000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for x := 0; x < 500; x++ {
				rand.Int63()
			}
		}()
	}
	wg.Wait()
	fmt.Println(time.Now().Sub(x).String())
}
