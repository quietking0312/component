// Package mlock 提供两级分布式锁：进程内互斥锁 + Redis 分布式锁。
//
// 加锁流程：
//  1. TryLock 进程内 mutex；若失败，说明本进程内已有 goroutine 持锁，
//     直接等待并重试，跳过 Redis 调用，减少对 Redis 的读写压力。
//  2. 进程内锁获取成功后，再向 Redis 发起 SET NX，防止跨节点并发。
//  3. Redis 锁获取失败（其他节点持锁），释放进程内锁并等待重试。
//
// 用法：
//
//	mgr := distlock.NewManager(redisClient)
//
//	lk := mgr.NewLock("lock:foo", 10*time.Second)
//	if err := lk.Lock(ctx); err != nil { ... }
//	defer lk.Unlock(context.Background())
package mlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"github.com/quietking0312/component/mredis"
)

const (
	retryInterval = 20 * time.Millisecond
	shardCount    = 32
)

// ErrNotHeld 对未持有的锁调用 Unlock 时返回。
var ErrNotHeld = errors.New("distlock: lock not held")

// ErrAlreadyHeld 对已持有的锁再次调用 Lock/TryLock 时返回。
var ErrAlreadyHeld = errors.New("distlock: lock already held")

// ErrReuse Unlock 后复用 Lock 对象时返回。
var ErrReuse = errors.New("distlock: lock reused after unlock")

// ErrLost 续期时发现 Redis 锁已不属于当前持有者（可能已过期或被他人获取）。
var ErrLost = errors.New("distlock: lock lost in redis")

// unlockScript 只在 key 仍持有我们的 token 时才删除，防止误删其他节点的锁。
var unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end`

// renewScript 只在 key 仍持有我们的 token 时才续期。
var renewScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end`

// entry 是同一进程内针对同一 key 共享的状态。
type entry struct {
	mu   sync.Mutex
	refs int // 引用计数，归零时从 shard.entries 中删除
}

// shard 是进程内锁表的一个分片，独立加锁以降低全局竞争。
type shard struct {
	mu      sync.Mutex
	entries map[string]*entry
}

// Manager 持有进程内锁表，必须全局唯一（单例），确保同一 key 的所有 goroutine
// 共享同一个 entry。
type Manager struct {
	redis  mredis.Client
	shards [shardCount]*shard
}

// NewManager 创建一个使用给定 Redis 客户端的 Manager。
func NewManager(redis mredis.Client) *Manager {
	m := &Manager{redis: redis}
	for i := 0; i < shardCount; i++ {
		m.shards[i] = &shard{entries: make(map[string]*entry)}
	}
	return m
}

// NewLock 返回针对指定 key 和 Redis TTL 的锁句柄。
// 每次调用都返回全新的未持有状态的 Lock；Unlock 后不得复用。
func (m *Manager) NewLock(key string, ttl time.Duration) *Lock {
	if ttl <= 0 {
		panic("distlock: ttl must be positive")
	}
	return &Lock{mgr: m, key: key, ttl: ttl}
}

func (m *Manager) getShard(key string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return m.shards[h.Sum32()%shardCount]
}

func (m *Manager) acquireEntry(key string) *entry {
	s := m.getShard(key)
	s.mu.Lock()
	e, ok := s.entries[key]
	if !ok {
		e = &entry{}
		s.entries[key] = e
	}
	e.refs++
	s.mu.Unlock()
	return e
}

func (m *Manager) releaseEntry(key string, e *entry) {
	s := m.getShard(key)
	s.mu.Lock()
	e.refs--
	if e.refs == 0 {
		delete(s.entries, key)
	}
	s.mu.Unlock()
}

// Lock 是两级命名锁（进程内 mutex + Redis SET NX）。
// 必须通过 Manager.NewLock 创建；不得复制；不得在 Unlock 后复用。
type Lock struct {
	mgr    *Manager
	key    string
	ttl    time.Duration
	token  string // 持锁期间 Redis key 的值，用于安全释放
	closed bool   // Unlock 后设为 true，禁止复用
	e      *entry // 非 nil 表示进程内 mutex 已持有
}

// Lock 阻塞直到获取锁或 ctx 被取消。成功返回 nil，调用方必须调用 Unlock。
func (l *Lock) Lock(ctx context.Context) error {
	if l.closed {
		return ErrReuse
	}
	if l.e != nil {
		return ErrAlreadyHeld
	}
	for {
		e := l.mgr.acquireEntry(l.key)

		if !e.mu.TryLock() {
			// 进程内已有 goroutine 持锁，无需访问 Redis，直接等待。
			l.mgr.releaseEntry(l.key, e)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryInterval):
				continue
			}
		}

		// 进程内锁已拿到，向 Redis 申请。
		token := newToken()
		ok, err := l.mgr.redis.SetNX(ctx, l.key, token, l.ttl).Result()
		if err != nil {
			e.mu.Unlock()
			l.mgr.releaseEntry(l.key, e)
			return err
		}
		if ok {
			l.token = token
			l.e = e
			return nil
		}

		// Redis 锁被其他节点持有，释放进程内锁后等待重试。
		e.mu.Unlock()
		l.mgr.releaseEntry(l.key, e)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}
	}
}

// TryLock 尝试一次性获取锁，不阻塞。
// 返回 (true, nil) 表示成功；(false, nil) 表示锁已被持有；(false, err) 表示 Redis 错误。
func (l *Lock) TryLock(ctx context.Context) (bool, error) {
	if l.closed {
		return false, ErrReuse
	}
	if l.e != nil {
		return false, ErrAlreadyHeld
	}
	e := l.mgr.acquireEntry(l.key)

	if !e.mu.TryLock() {
		l.mgr.releaseEntry(l.key, e)
		return false, nil
	}

	token := newToken()
	ok, err := l.mgr.redis.SetNX(ctx, l.key, token, l.ttl).Result()
	if err != nil || !ok {
		e.mu.Unlock()
		l.mgr.releaseEntry(l.key, e)
		return false, err
	}

	l.token = token
	l.e = e
	return true, nil
}

// Unlock 释放锁。进程内 mutex 始终会被释放，即使 Redis 删除失败。
// 建议传入 context.Background() 确保 Redis 清理命令不被取消。
func (l *Lock) Unlock(ctx context.Context) error {
	if l.closed {
		return ErrReuse
	}
	if l.e == nil {
		return ErrNotHeld
	}
	l.closed = true
	e, token := l.e, l.token
	l.e = nil
	l.token = ""

	defer func() {
		e.mu.Unlock()
		l.mgr.releaseEntry(l.key, e)
	}()

	return l.mgr.redis.Eval(ctx, unlockScript, []string{l.key}, token).Err()
}

// Renew 续期当前持有的分布式锁，将 Redis key 的 TTL 重置为创建锁时指定的 ttl。
// 调用方必须已持有该锁；若锁已丢失（过期或被其他节点获取）则返回 ErrLost。
func (l *Lock) Renew(ctx context.Context) error {
	if l.closed {
		return ErrReuse
	}
	if l.e == nil {
		return ErrNotHeld
	}

	ttlMs := l.ttl.Milliseconds()
	if ttlMs <= 0 {
		ttlMs = 1
	}

	n, err := l.mgr.redis.Eval(ctx, renewScript, []string{l.key}, l.token, ttlMs).Int64()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLost
	}
	return nil
}

func newToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("distlock: failed to generate token: " + err.Error())
	}
	return hex.EncodeToString(b)
}
