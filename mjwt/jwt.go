package mjwt

import (
	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	key []byte
	*jwt.Token
}

func NewJWT(key []byte, method jwt.SigningMethod) *JWT {
	return &JWT{
		key:   key,
		Token: jwt.New(method),
	}
}

func (j *JWT) SetData(data jwt.Claims) {
	j.Token.Claims = data
}

func (j *JWT) SignedString() (string, error) {
	return j.Token.SignedString(j.key)
}

func (j *JWT) Parse(token string, data jwt.Claims) (*JWT, error) {
	_, err := jwt.ParseWithClaims(token, data, func(token *jwt.Token) (interface{}, error) {
		return j.key, nil
	})
	if err != nil {
		return nil, err
	}
	return &JWT{
		key:   j.key,
		Token: &jwt.Token{Claims: data},
	}, nil
}
