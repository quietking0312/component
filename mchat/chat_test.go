package mchat

import (
	"testing"
	"time"
)

func TestNewChat(t *testing.T) {
	c, err := NewChat()
	if err != nil {
		t.Fatal(err)
	}
	go c.Sub(1)
	go c.Sub(2)
	time.Sleep(10 * time.Second)
	c.Publish()
}
