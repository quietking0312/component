package mcachedb

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// 事务层默认常量
const (
	defaultTxTimeout = 30 * time.Second
)

// Tx 事务
type Tx struct {
	cache      *Cache
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	opStack    []txOp
	committed  bool
	rolledBack bool
}

// txOp 事务操作
type txOp struct {
	typ    string // "set" | "delete"
	key    string
	entity Entity
	old    Entity // 旧值，用于回滚
}

// Begin 开始事务
func (c *Cache) Begin() (*Tx, error) {
	if c.store == nil {
		return nil, fmt.Errorf("no store configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTxTimeout)

	return &Tx{
		cache:   c,
		ctx:     ctx,
		cancel:  cancel,
		opStack: make([]txOp, 0),
	}, nil
}

// Get 事务中获取
func (tx *Tx) Get(key string) (Entity, error) {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return nil, fmt.Errorf("transaction already finished")
	}

	// 检查未提交的操作栈
	for i := len(tx.opStack) - 1; i >= 0; i-- {
		op := tx.opStack[i]
		if op.key == key {
			if op.typ == "delete" {
				return nil, nil
			}
			return op.entity.Copy(), nil
		}
	}

	// 从缓存获取
	return tx.cache.Get(key)
}

// Set 事务中设置
func (tx *Tx) Set(entity Entity) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}

	key := entity.CacheKey()

	// 保存旧值
	old, _ := tx.cache.Get(key)

	entity.IncrementVersion()

	tx.opStack = append(tx.opStack, txOp{
		typ:    "set",
		key:    key,
		entity: entity.Copy(),
		old:    old,
	})

	return nil
}

// Delete 事务中删除
func (tx *Tx) Delete(key string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}

	// 保存旧值
	old, _ := tx.cache.Get(key)

	tx.opStack = append(tx.opStack, txOp{
		typ: "delete",
		key: key,
		old: old,
	})

	return nil
}

// Commit 提交事务
func (tx *Tx) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}

	tx.committed = true
	defer tx.cancel()

	// 执行所有操作
	executed := make([]int, 0, len(tx.opStack))
	for i, op := range tx.opStack {
		switch op.typ {
		case "set":
			if err := tx.cache.Set(op.entity); err != nil {
				tx.rollbackOps(tx.opStack[:len(executed)])
				return fmt.Errorf("commit failed: %w", err)
			}
		case "delete":
			if err := tx.cache.Delete(op.key); err != nil {
				tx.rollbackOps(tx.opStack[:len(executed)])
				return fmt.Errorf("commit failed: %w", err)
			}
		}
		executed = append(executed, i)
	}

	// 如果是同步模式，确保数据写入数据库
	if tx.cache.config.WriteMode != WriteModeAsync {
		if err := tx.cache.Flush(); err != nil {
			return fmt.Errorf("flush failed: %w", err)
		}
	}

	return nil
}

// Rollback 回滚事务
func (tx *Tx) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}

	tx.rolledBack = true
	defer tx.cancel()

	tx.rollbackOps(tx.opStack)

	return nil
}

// rollbackOps 回滚操作
func (tx *Tx) rollbackOps(ops []txOp) {
	// 逆序回滚
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		if op.old != nil {
			// 恢复旧值
			tx.cache.mu.Lock()
			tx.cache.data[op.key] = &entry{
				entity:    op.old.Copy(),
				createdAt: time.Now(),
				dirty:     false,
			}
			tx.cache.mu.Unlock()
		} else {
			// 删除新添加的
			tx.cache.mu.Lock()
			delete(tx.cache.data, op.key)
			tx.cache.mu.Unlock()
		}
	}
}

// Pipeline 管道批量操作
type Pipeline struct {
	cache *Cache
	ops   []pipeOp
	mu    sync.Mutex
}

type pipeOp struct {
	typ    string // "set" | "delete"
	key    string
	entity Entity
}

// Pipeline 创建管道
func (c *Cache) Pipeline() *Pipeline {
	return &Pipeline{
		cache: c,
		ops:   make([]pipeOp, 0),
	}
}

// Set 管道设置
func (p *Pipeline) Set(entity Entity) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ops = append(p.ops, pipeOp{
		typ:    "set",
		key:    entity.CacheKey(),
		entity: entity,
	})
	return p
}

// Delete 管道删除
func (p *Pipeline) Delete(key string) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ops = append(p.ops, pipeOp{
		typ: "delete",
		key: key,
	})
	return p
}

// Exec 执行管道
func (p *Pipeline) Exec() error {
	p.mu.Lock()
	ops := make([]pipeOp, len(p.ops))
	copy(ops, p.ops)
	p.ops = p.ops[:0]
	p.mu.Unlock()

	// 批量执行
	for _, op := range ops {
		switch op.typ {
		case "set":
			if err := p.cache.Set(op.entity); err != nil {
				return err
			}
		case "delete":
			if err := p.cache.Delete(op.key); err != nil {
				return err
			}
		}
	}

	// 如果是同步模式，立即刷新
	if p.cache.config.WriteMode != WriteModeAsync {
		return p.cache.Flush()
	}

	return nil
}

// Discard 丢弃管道操作
func (p *Pipeline) Discard() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ops = p.ops[:0]
}

// Len 返回管道长度
func (p *Pipeline) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.ops)
}
