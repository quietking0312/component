package mtool

import (
	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	key    []byte
	method jwt.SigningMethod
	data   jwt.MapClaims
}

func NewJWT(key []byte, method jwt.SigningMethod) *JWT {
	return &JWT{
		key:    key,
		method: method,
		data:   make(jwt.MapClaims),
	}
}

func (j *JWT) SetData(data map[string]any) {
	for k, v := range data {
		j.data[k] = v
	}
}

func (j *JWT) GetData() map[string]any {
	return j.data
}

func (j *JWT) Token() (string, error) {
	t := jwt.NewWithClaims(j.method, j.data)
	return t.SignedString(j.key)
}

func (j *JWT) Parse(token string) (*JWT, error) {
	_, err := jwt.ParseWithClaims(token, &j.data, func(token *jwt.Token) (interface{}, error) {
		return j.key, nil
	})
	if err != nil {
		return nil, err
	}
	return j, nil
}
