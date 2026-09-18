package mreqsign

import (
	"crypto/hmac"
	"errors"
	"fmt"
	"time"

	"github.com/quietking0312/component/mcyptos"
)

// ErrExpired 表示请求时间戳超出了允许的时间偏差范围
var ErrExpired = errors.New("mreqsign: request timestamp expired or in the future")

// ErrInvalidSignature 表示签名校验不通过（数据被篡改或密钥不匹配）
var ErrInvalidSignature = errors.New("mreqsign: signature mismatch")

// Sign 使用密钥池中与时间 t 对应的密钥对 payload 进行签名。
// 返回值 ts 为签名时使用的 Unix 时间戳（秒），调用方需要将 ts 和 sig 一并发送给对方用于校验。
func (p *KeyPool) Sign(payload []byte, t time.Time) (ts int64, sig string) {
	ts = t.Unix()
	_, key := p.KeyAt(t)
	sig = signWithKey(key, ts, payload)
	return ts, sig
}

// Verify 校验请求是否被篡改。
//   - payload: 参与签名的原始数据（通常是请求方法、路径、body 等按约定拼接而成）
//   - ts: 对方发送过来的签名时间戳（Unix 秒）
//   - sig: 对方发送过来的签名
//   - maxSkew: 允许的最大时间偏差，超过该偏差的请求会被拒绝（用于防止重放及应对时钟漂移）
func (p *KeyPool) Verify(payload []byte, ts int64, sig string, maxSkew time.Duration) error {
	reqTime := time.Unix(ts, 0)
	if d := time.Since(reqTime); d > maxSkew || d < -maxSkew {
		return ErrExpired
	}

	_, key := p.KeyAt(reqTime)
	expected := signWithKey(key, ts, payload)

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return ErrInvalidSignature
	}
	return nil
}

func signWithKey(key []byte, ts int64, payload []byte) string {
	data := make([]byte, 0, len(payload)+20)
	data = append(data, []byte(fmt.Sprintf("%d.", ts))...)
	data = append(data, payload...)
	return mcyptos.HMACSHA256(key, data)
}
