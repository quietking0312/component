package mtcp

import (
	"fmt"
	"testing"
)

func TestNewNode(t *testing.T) {
	router := NewRouter()
	router.AddRoute("/a/a/a/a/a")
	router.AddRoute("/b/c/d")
	fmt.Println(router.root.children[0].children[0])
}
