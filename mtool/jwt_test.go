package mtool

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"testing"
	"time"
)

func TestJWT(t *testing.T) {
	//var data = struct {
	//	*jwt.RegisteredClaims
	//}{
	//	&jwt.RegisteredClaims{
	//		ID:      "1",
	//		Issuer:  "server",
	//		Subject: "username",
	//	},
	//}
	j := NewJWT([]byte("hello"), jwt.SigningMethodHS256)
	j.SetData(map[string]any{
		"id":  1,
		"iss": "server",
		"sub": "username",
		"exp": time.Second * 30,
	})
	token, err := j.Token()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(token)

	j2 := NewJWT([]byte("hello"), jwt.SigningMethodHS256)
	j2, err = j2.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(j2.GetData())
}
