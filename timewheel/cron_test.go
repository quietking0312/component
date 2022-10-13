package timewheel

import (
	"fmt"
	"testing"
	"time"
)

func TestNewCron(t *testing.T) {
	c := NewCron()
	c.Start()
	b := new(a)
	id, err := c.AddJob("30 17 * * *", b)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(id)
	defer c.Stop()
	time.Sleep(10 * time.Minute)
}

type a struct {
}

func (a *a) Run() {
	fmt.Println("hello")
}
