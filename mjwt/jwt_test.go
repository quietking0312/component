package mjwt

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserInfo struct {
	ID   int64
	Name string
	Role string
}

func TestSign(t *testing.T) {
	j := New[UserInfo]([]byte("secret"), jwt.SigningMethodHS256)

	token, err := j.Sign(UserInfo{ID: 1, Name: "alice", Role: "admin"},
		WithExpiry(2*time.Hour),
		WithIssuer("myapp"),
		WithSubject("auth"),
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("token:", token)

	data, rc, err := j.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("data: %+v\n", data)
	fmt.Printf("issuer: %s, subject: %s, expires: %s\n", rc.Issuer, rc.Subject, rc.ExpiresAt)
}

func TestParse_WrongKey(t *testing.T) {
	j1 := New[UserInfo]([]byte("secret"), jwt.SigningMethodHS256)
	j2 := New[UserInfo]([]byte("wrongkey"), jwt.SigningMethodHS256)

	token, err := j1.Sign(UserInfo{ID: 2, Name: "bob", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = j2.Parse(token)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
	fmt.Println("wrong key error:", err)
}

func TestParse_Expired(t *testing.T) {
	j := New[UserInfo]([]byte("secret"), jwt.SigningMethodHS256)

	token, err := j.Sign(UserInfo{ID: 3}, WithExpiry(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = j.Parse(token)
	if err == nil {
		t.Fatal("expected expiry error")
	}
	fmt.Println("expired error:", err)
}
