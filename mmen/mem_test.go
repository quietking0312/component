package mmen

import (
	"fmt"
	"testing"
)

func TestGetProcesses(t *testing.T) {
	a, err := GetProcesses()
	if err != nil {
		t.Log(err)
	}
	fmt.Println(a)
}
