package mredis

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cli, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	r := cli.Set(context.Background(), "a", 1, 5*time.Minute)
	fmt.Println(r.Result())
	result := cli.Get(context.Background(), "a")
	fmt.Println(result.Result())

}
