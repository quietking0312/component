package mjwt

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"testing"
)

type N struct {
	Id int64
	*jwt.RegisteredClaims
}

func TestJWT(t *testing.T) {
	j := NewJWT([]byte("battle"), jwt.SigningMethodHS256)
	j.SetData(&N{
		Id:               1,
		RegisteredClaims: &jwt.RegisteredClaims{},
	})
	token, err := j.SignedString()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(token)

	j2 := NewJWT([]byte("battle"), jwt.SigningMethodHS256)
	var tokenData = &N{
		RegisteredClaims: &jwt.RegisteredClaims{},
	}
	j2, err = j2.Parse(token, tokenData)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tokenData.Id)
}
