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
	typ      string // "set" | "delete"
	key      string
	entity   Entity
	oldEntry *entry // L1 旧 entry 快照（含 dirty 状态），nil 表示操作前 key 不在 L1
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

	tx.opStack = append(tx.opStack, txOp{
		typ:      "set",
		key:      key,
		entity:   entity.Copy(),
		oldEntry: tx.snapshotL1(key),
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

	tx.opStack = append(tx.opStack, txOp{
		typ:      "delete",
		key:      key,
		oldEntry: tx.snapshotL1(key),
	})

	return nil
}

// snapshotL1 快照当前 key 在 L1 中的 entry（含 dirty 状态），key 不存在时返回 nil
func (tx *Tx) snapshotL1(key string) *entry {
	tx.cache.mu.RLock()
	defer tx.cache.mu.RUnlock()
	e, ok := tx.cache.data[key]
	if !ok {
		return nil
	}
	cloned := *e
	if e.entity != nil {
		cloned.entity = e.entity.Copy()
	}
	return &cloned
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

// rollbackOps 逆序回滚已执行的操作，恢复 L1 至操作前状态（含 dirty 标记）
func (tx *Tx) rollbackOps(ops []txOp) {
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		tx.cache.mu.Lock()
		if op.oldEntry != nil {
			tx.cache.data[op.key] = op.oldEntry
		} else {
			delete(tx.cache.data, op.key)
		}
		tx.cache.mu.Unlock()

		// 若旧 entry 是脏数据，需重新加入 dirty 队列，确保最终能 flush 到 DB
		if op.oldEntry != nil && op.oldEntry.dirty {
			tx.cache.addDirty(op.key)
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
