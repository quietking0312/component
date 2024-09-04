package mtool

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"testing"
)

type N struct {
	Id int64
	*JWTClaims
}

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
	j := NewJWT([]byte("battle"), jwt.SigningMethodHS256)
	j.SetData(&N{
		Id:        1,
		JWTClaims: new(JWTClaims),
	})
	token, err := j.SignedString()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(token)

	j2 := NewJWT([]byte("battle"), jwt.SigningMethodHS256)
	var tokenData = &N{
		JWTClaims: new(JWTClaims),
	}
	j2, err = j2.Parse(token, tokenData)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tokenData.Id)
}
