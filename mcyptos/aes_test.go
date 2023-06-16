package mcyptos

import (
	"fmt"
	"testing"
)

func TestDecryptAES(t *testing.T) {
	k := "teujWMYGcQob6OcCVRruyHMtENRcIlvuM4ghIWqYCiF"
	kstr, err := DecodeBase64(k + "=")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(kstr))
	fmt.Println(string(kstr))
}
