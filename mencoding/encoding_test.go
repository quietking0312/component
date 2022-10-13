package mencoding

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

func TestDecoder(t *testing.T) {
	newBytes, err := Decoder(bufio.NewReader(strings.NewReader("helloworld")))
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(newBytes))
}
