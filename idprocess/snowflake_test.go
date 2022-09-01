package idprocess

import (
	"fmt"
	"testing"
)

func TestNewWorker(t *testing.T) {
	worker, err := NewWorker(1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(worker.GetId())
}

func TestWorker_UnId(t *testing.T) {
	worker, err := NewWorker(1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(worker.UnId(14971982361661440))
}
