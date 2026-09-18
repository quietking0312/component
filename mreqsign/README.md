# mreqsign - 基于时间片密钥池的请求签名/防篡改组件

服务间通信如果长期使用同一个固定密钥做签名，一旦密钥泄露，攻击者可以无限期地伪造请求。
`mreqsign` 提供一种折中方案：双方预先约定一份密钥池（多个密钥）和一个时间片长度，
双方各自根据请求发起时间独立算出当前应使用的密钥，无需在请求中额外传递密钥或密钥版本号，
既避免了单一密钥长期暴露的风险，也不需要引入密钥分发/协商这种更重的机制。

## 原理

1. 双方预先约定：
   - 一份密钥列表 `keys`（密钥池）
   - 时间片长度 `window`（例如 1 分钟、5 分钟）
   - 时间片编号起始时间 `epoch`（默认 Unix 时间零点）
2. 对于任意时间 `t`，时间片编号 `slot = floor((t - epoch) / window)`，
   实际使用的密钥为 `keys[slot % len(keys)]`。
3. 签名方使用 `HMAC-SHA256(key, "{unix时间戳}.{payload}")` 计算签名，
   将 `payload`、时间戳、签名一起发送给对方。
4. 验证方收到请求后，先检查时间戳与当前时间的偏差是否在允许范围内（防重放/防止时钟漂移过大），
   再用相同的方式算出时间片对应的密钥，重新计算签名并比较。

时间片编号完全由请求中携带的时间戳计算得出，因此双方永远会用同一个密钥校验同一个请求，
不存在因为服务端/客户端时钟不同步导致取错密钥的问题；真正需要控制的时钟偏差只影响
"这个时间戳是否还新鲜"（`maxSkew`）。

## 快速开始

```go
package main

import (
    "fmt"
    "time"

    "github.com/quietking0312/component/mreqsign"
)

func main() {
    // 服务启动时根据约定好的 seed 确定性地派生出密钥池；
    // 只要各服务使用相同的 seed/数量/长度，独立生成出的密钥池完全一致，
    // 不需要在网络上传输密钥本身，只需要各服务安全地持有同一个 seed（例如通过配置中心分发）
    seed := []byte("从安全渠道获取的共享种子")
    keys, _ := mreqsign.GenerateKeys(seed, 8, 32) // 8 个密钥，每个 32 字节
    pool, _ := mreqsign.NewKeyPool(keys, 5*time.Minute)

    // 签名方（发起请求的一方）
    payload := []byte("POST&/api/order&{...}")
    ts, sig := pool.Sign(payload, time.Now())

    // 将 payload、ts、sig 一起放入请求（例如 header）发送出去

    // 验证方（接收请求的一方，使用同一份密钥池）
    err := pool.Verify(payload, ts, sig, 30*time.Second)
    if err != nil {
        fmt.Println("请求校验失败:", err) // ErrExpired 或 ErrInvalidSignature
        return
    }
    fmt.Println("请求校验通过")
}
```

## API

| 函数/方法 | 说明 |
|-----------|------|
| `GenerateKeys(seed []byte, n, keyLen int) ([][]byte, error)` | 根据 seed 确定性地派生出 n 个密钥；相同 seed/n/keyLen 在任意服务、任意时刻生成的结果都相同 |
| `NewKeyPool(keys [][]byte, window time.Duration, opts ...Option) (*KeyPool, error)` | 创建密钥池 |
| `WithEpoch(t time.Time) Option` | 设置时间片编号起始时间，双方必须一致 |
| `(*KeyPool).KeyAt(t time.Time) (slot int64, key []byte)` | 获取时间 t 对应的时间片编号与密钥 |
| `(*KeyPool).Sign(payload []byte, t time.Time) (ts int64, sig string)` | 对 payload 签名 |
| `(*KeyPool).Verify(payload []byte, ts int64, sig string, maxSkew time.Duration) error` | 校验签名，返回 `ErrExpired` 或 `ErrInvalidSignature` |

## 注意事项

1. **seed 分发**：只需要通过安全的方式（配置中心、启动时注入等）分发同一个 `seed` 给通信双方，双方各自调用 `GenerateKeys` 即可得到相同的密钥池，无需传输密钥本身；`seed` 的机密性等同于整个密钥池的机密性，`mreqsign` 本身不解决 seed 分发问题。
2. **时间片长度选择**：时间片越短，单个密钥暴露窗口越小，但要求双方时钟更同步；一般建议配合较宽松的 `maxSkew`（例如时间片长度的几倍）。
3. **防重放**：本组件通过时间戳窗口校验缓解重放攻击，但不做去重；如果需要严格防重放，建议结合 `nonce` + 缓存（例如 mredis）做请求去重。
4. **密钥池更新**：如需轮换整份密钥池（而不是池内单个密钥的自动轮转),需要双方同时更新配置并重启/热加载。

## 测试

```bash
cd mreqsign
go test -v
```
