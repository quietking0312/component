// Package middleware 提供熔断器和限流器，供 grpc 和 gorpc 共用。
package middleware

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ========== 错误定义 ==========

var (
	// ErrCircuitOpen 熔断器开路，拒绝请求
	ErrCircuitOpen = errors.New("middleware: circuit breaker is open")
	// ErrRateLimited 请求被限流
	ErrRateLimited = errors.New("middleware: rate limit exceeded")
)

// ========== 熔断器（滑动窗口） ==========

// CircuitState 熔断器状态
type CircuitState int32

const (
	StateClosed   CircuitState = iota // 关闭（正常）
	StateOpen                         // 开路（拒绝请求）
	StateHalfOpen                     // 半开（试探）
)

// CircuitConfig 熔断器配置
type CircuitConfig struct {
	// 滑动窗口大小（请求数），默认 100
	WindowSize int
	// 窗口内失败率阈值（0-1），超过则开路，默认 0.5
	FailureRatio float64
	// 开路后等待多久进入半开状态，默认 5s
	OpenTimeout time.Duration
	// 半开状态下允许通过的试探请求数，默认 3
	HalfOpenRequests int
}

func (c *CircuitConfig) fill() {
	if c.WindowSize <= 0 {
		c.WindowSize = 100
	}
	if c.FailureRatio <= 0 {
		c.FailureRatio = 0.5
	}
	if c.OpenTimeout <= 0 {
		c.OpenTimeout = 5 * time.Second
	}
	if c.HalfOpenRequests <= 0 {
		c.HalfOpenRequests = 3
	}
}

// CircuitBreaker 滑动窗口熔断器
type CircuitBreaker struct {
	cfg CircuitConfig

	mu       sync.Mutex
	state    CircuitState
	openedAt time.Time

	// 滑动窗口：环形 bool 数组，true=失败
	window  []bool
	wIdx    int
	wTotal  int // 窗口内总请求数（最多 WindowSize）
	wFail   int // 窗口内失败数
	halfCnt int // 半开状态已通过的试探数
}

// NewCircuitBreaker 创建熔断器。
func NewCircuitBreaker(cfg CircuitConfig) *CircuitBreaker {
	cfg.fill()
	return &CircuitBreaker{
		cfg:    cfg,
		window: make([]bool, cfg.WindowSize),
	}
}

// Allow 判断当前请求是否允许通过。
// 返回 done 函数，请求完成后必须调用 done(err) 上报结果。
func (cb *CircuitBreaker) Allow() (done func(err error), allowed bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateOpen:
		if time.Since(cb.openedAt) < cb.cfg.OpenTimeout {
			return nil, false
		}
		// 超时后进入半开
		cb.state = StateHalfOpen
		cb.halfCnt = 0

	case StateHalfOpen:
		if cb.halfCnt >= cb.cfg.HalfOpenRequests {
			return nil, false
		}
		cb.halfCnt++
	}

	return cb.report, true
}

func (cb *CircuitBreaker) report(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	failed := err != nil

	if cb.state == StateHalfOpen {
		if failed {
			// 试探失败，重新开路
			cb.state = StateOpen
			cb.openedAt = time.Now()
		} else if cb.halfCnt >= cb.cfg.HalfOpenRequests {
			// 所有试探成功，关闭熔断
			cb.state = StateClosed
			cb.resetWindow()
		}
		return
	}

	// 滑动窗口更新
	old := cb.window[cb.wIdx]
	cb.window[cb.wIdx] = failed
	cb.wIdx = (cb.wIdx + 1) % cb.cfg.WindowSize

	if cb.wTotal < cb.cfg.WindowSize {
		cb.wTotal++
	} else if old {
		cb.wFail--
	}
	if failed {
		cb.wFail++
	}

	// 窗口满后检查失败率
	if cb.wTotal >= cb.cfg.WindowSize {
		ratio := float64(cb.wFail) / float64(cb.wTotal)
		if ratio >= cb.cfg.FailureRatio {
			cb.state = StateOpen
			cb.openedAt = time.Now()
		}
	}
}

func (cb *CircuitBreaker) resetWindow() {
	cb.window = make([]bool, cb.cfg.WindowSize)
	cb.wIdx = 0
	cb.wTotal = 0
	cb.wFail = 0
}

// State 返回当前熔断状态（用于监控）。
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// ========== 限流器（令牌桶） ==========

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	// 每秒补充令牌数
	Rate float64
	// 桶容量（最大突发量）
	Burst int
}

// RateLimiter 令牌桶限流器
type RateLimiter struct {
	rate     float64 // 每纳秒补充的令牌数
	burst    float64
	tokens   float64
	lastTime int64 // unix nano，atomic 读写
	mu       sync.Mutex
}

// NewRateLimiter 创建限流器。rate: 每秒请求数；burst: 最大突发量。
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	burst := cfg.Burst
	if burst <= 0 {
		burst = int(cfg.Rate)
	}
	return &RateLimiter{
		rate:     cfg.Rate / 1e9,
		burst:    float64(burst),
		tokens:   float64(burst),
		lastTime: time.Now().UnixNano(),
	}
}

// Allow 尝试消费 1 个令牌，返回是否允许通过。
func (r *RateLimiter) Allow() bool {
	return r.AllowN(1)
}

// AllowN 尝试消费 n 个令牌。
func (r *RateLimiter) AllowN(n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&r.lastTime)
	elapsed := float64(now - last)
	r.tokens += elapsed * r.rate
	if r.tokens > r.burst {
		r.tokens = r.burst
	}
	atomic.StoreInt64(&r.lastTime, now)

	if r.tokens < float64(n) {
		return false
	}
	r.tokens -= float64(n)
	return true
}
