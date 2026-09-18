package mreqsign

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/quietking0312/component/mcyptos"
)

// KeyPool 是一个按时间片轮转取密钥的密钥池。
//
// 双方（签名方与验证方）预先约定好同一份密钥列表、时间片长度（Window）以及起始时间（Epoch），
// 之后即可各自根据请求发起时间独立算出应使用的密钥，无需在请求中额外传递密钥本身或密钥版本号。
//
// 时间片编号计算方式：slot = floor((t - Epoch) / Window)，实际使用的密钥为 keys[slot % len(keys)]。
type KeyPool struct {
	keys   [][]byte
	window time.Duration
	epoch  time.Time
}

// Option 用于配置 KeyPool 的可选参数
type Option func(*KeyPool)

// WithEpoch 设置时间片编号的起始时间，默认为 Unix 时间零点（1970-01-01T00:00:00Z）。
// 双方必须使用相同的 Epoch，否则算出的时间片编号会不一致。
func WithEpoch(t time.Time) Option {
	return func(p *KeyPool) {
		p.epoch = t
	}
}

// NewKeyPool 创建一个密钥池。
//   - keys: 预先约定的密钥列表，长度至少为 1，且每个密钥不能为空
//   - window: 时间片长度，即多久轮换到下一个密钥，必须大于 0
func NewKeyPool(keys [][]byte, window time.Duration, opts ...Option) (*KeyPool, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("mreqsign: key pool must contain at least one key")
	}
	for i, k := range keys {
		if len(k) == 0 {
			return nil, fmt.Errorf("mreqsign: key at index %d is empty", i)
		}
	}
	if window <= 0 {
		return nil, fmt.Errorf("mreqsign: window must be greater than 0")
	}

	p := &KeyPool{
		keys:   keys,
		window: window,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// Slot 返回时间 t 所处的时间片编号
func (p *KeyPool) Slot(t time.Time) int64 {
	return int64(t.Sub(p.epoch) / p.window)
}

// KeyAt 返回时间 t 对应的时间片编号及应使用的密钥
func (p *KeyPool) KeyAt(t time.Time) (slot int64, key []byte) {
	slot = p.Slot(t)
	idx := slot % int64(len(p.keys))
	if idx < 0 {
		idx += int64(len(p.keys))
	}
	return slot, p.keys[idx]
}

// Size 返回密钥池中密钥的数量
func (p *KeyPool) Size() int {
	return len(p.keys)
}

// Window 返回时间片长度
func (p *KeyPool) Window() time.Duration {
	return p.window
}

// GenerateKeys 根据 seed 确定性地派生出 n 个长度为 keyLen 字节的密钥。
//
// 只要多个服务使用相同的 seed、n、keyLen，各自独立调用 GenerateKeys 算出来的密钥池
// 一定完全一致，因此密钥池本身不需要在网络上传输或存储到共享存储中，只需要各服务
// 通过安全渠道（如启动参数、密钥管理服务）持有同一个 seed 即可。
//
// seed 的机密性等同于整个密钥池的机密性，必须妥善保管，且长度建议不少于 16 字节。
func GenerateKeys(seed []byte, n, keyLen int) ([][]byte, error) {
	if len(seed) == 0 {
		return nil, fmt.Errorf("mreqsign: seed must not be empty")
	}
	if n <= 0 {
		return nil, fmt.Errorf("mreqsign: n must be greater than 0")
	}
	if keyLen <= 0 {
		return nil, fmt.Errorf("mreqsign: keyLen must be greater than 0")
	}

	keys := make([][]byte, n)
	for i := 0; i < n; i++ {
		keys[i] = deriveKey(seed, i, keyLen)
	}
	return keys, nil
}

// deriveKey 基于 seed 与索引 index，通过 HMAC-SHA256 计数器模式扩展出 keyLen 字节的
// 确定性密钥：block_i = HMAC-SHA256(seed, "mreqsign-key-pool|index|counter")，
// 依次拼接各 block 直到长度足够，再截取所需长度。
func deriveKey(seed []byte, index, keyLen int) []byte {
	out := make([]byte, 0, keyLen+32)
	for counter := 0; len(out) < keyLen; counter++ {
		label := fmt.Sprintf("mreqsign-key-pool|%d|%d", index, counter)
		block := mcyptos.HMACSHA256(seed, []byte(label))
		raw, _ := hex.DecodeString(block)
		out = append(out, raw...)
	}
	return out[:keyLen]
}
