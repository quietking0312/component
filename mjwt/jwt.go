package mjwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 用户自定义数据必须内嵌 jwt.RegisteredClaims。
// 泛型约束：T 是指针类型且实现了 jwt.Claims 接口。
type Claims[T any] struct {
	Data T `json:"data"`
	jwt.RegisteredClaims
}

// JWT 泛型 JWT 封装，T 为业务数据类型。
type JWT[T any] struct {
	key    []byte
	method jwt.SigningMethod
}

// New 创建 JWT 实例。
// key: 签名密钥；method: 签名算法（如 jwt.SigningMethodHS256）。
func New[T any](key []byte, method jwt.SigningMethod) *JWT[T] {
	return &JWT[T]{key: key, method: method}
}

// Sign 生成 token，data 为业务数据，opts 可选设置过期时间等。
func (j *JWT[T]) Sign(data T, opts ...Option) (string, error) {
	cfg := &signConfig{}
	for _, o := range opts {
		o(cfg)
	}

	rc := jwt.RegisteredClaims{}
	if cfg.expiry != 0 {
		rc.ExpiresAt = jwt.NewNumericDate(time.Now().Add(cfg.expiry))
	}
	if cfg.issuer != "" {
		rc.Issuer = cfg.issuer
	}
	if cfg.subject != "" {
		rc.Subject = cfg.subject
	}
	if !cfg.notBefore.IsZero() {
		rc.NotBefore = jwt.NewNumericDate(cfg.notBefore)
	}
	rc.IssuedAt = jwt.NewNumericDate(time.Now())

	claims := &Claims[T]{Data: data, RegisteredClaims: rc}
	token := jwt.NewWithClaims(j.method, claims)
	return token.SignedString(j.key)
}

// Parse 解析 token，返回业务数据及注册声明。
func (j *JWT[T]) Parse(tokenStr string) (T, *jwt.RegisteredClaims, error) {
	var zero T
	claims := &Claims[T]{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != j.method.Alg() {
			return nil, errors.New("unexpected signing method: " + token.Method.Alg())
		}
		return j.key, nil
	}, jwt.WithLeeway(0))
	if err != nil {
		return zero, nil, err
	}
	return claims.Data, &claims.RegisteredClaims, nil
}

// ========== 选项 ==========

type signConfig struct {
	expiry    time.Duration
	issuer    string
	subject   string
	notBefore time.Time
}

// Option Sign 选项函数。
type Option func(*signConfig)

// WithExpiry 设置过期时长。
func WithExpiry(d time.Duration) Option {
	return func(c *signConfig) { c.expiry = d }
}

// WithIssuer 设置签发者。
func WithIssuer(iss string) Option {
	return func(c *signConfig) { c.issuer = iss }
}

// WithSubject 设置主题。
func WithSubject(sub string) Option {
	return func(c *signConfig) { c.subject = sub }
}

// WithNotBefore 设置生效时间。
func WithNotBefore(t time.Time) Option {
	return func(c *signConfig) { c.notBefore = t }
}
