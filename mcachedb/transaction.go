package mcachedb

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	defaultTxTimeout      = 30 * time.Second
	defaultDBWriteTimeout = 5 * time.Second
)

// Tx 单个 MultiCache 上的 CacheAside 事务。
//
// 操作顺序：
//  1. Tx.Set / Tx.Delete 收集操作，同时预读 L3 旧值（用于 Commit 失败时的尽力回滚）。
//  2. Tx.Commit 按顺序写 L3，每步成功后立即删除 L1/L2 缓存。
//     若某步 L3 写入失败，对已执行步骤进行尽力回滚，并返回错误。
//  3. Tx.Rollback 在 Commit 之前调用时，L3 尚未被修改，无需额外处理。
//
// 注意：Tx 不提供跨进程的严格原子性，适用于同进程内的最大努力一致性场景。
type Tx struct {
	mc         *MultiCache
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	ops        []txOp
	committed  bool
	rolledBack bool
}

type txOp struct {
	typ    string // "set" | "delete"
	key    string
	entity Entity // 新值（set 时有效）
	oldL3  Entity // 操作前 L3 中的旧值（用于失败回滚）
}

func newTx(mc *MultiCache) *Tx {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTxTimeout)
	return &Tx{
		mc:     mc,
		ctx:    ctx,
		cancel: cancel,
		ops:    make([]txOp, 0),
	}
}

// Get 在事务上下文中读取，优先返回当前事务中已暂存的操作结果
func (tx *Tx) Get(key string) (Entity, error) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.committed || tx.rolledBack {
		return nil, fmt.Errorf("transaction already finished")
	}
	for i := len(tx.ops) - 1; i >= 0; i-- {
		op := tx.ops[i]
		if op.key == key {
			if op.typ == "delete" {
				return nil, nil
			}
			return op.entity.Copy(), nil
		}
	}
	return tx.mc.l3.Get(tx.ctx, key)
}

// Set 将实体写入事务（不立即执行，Commit 时写 L3）
func (tx *Tx) Set(entity Entity) error {
	if err := validateEntity(entity); err != nil {
		return err
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}
	key := entity.CacheKey()
	old, err := tx.mc.l3.Get(tx.ctx, key)
	if err != nil {
		return fmt.Errorf("pre-read L3 for key %s: %w", key, err)
	}
	tx.ops = append(tx.ops, txOp{
		typ:    "set",
		key:    key,
		entity: entity.Copy(),
		oldL3:  old,
	})
	return nil
}

// Delete 将删除操作加入事务（Commit 时删 L3）
func (tx *Tx) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}
	old, err := tx.mc.l3.Get(tx.ctx, key)
	if err != nil {
		return fmt.Errorf("pre-read L3 for key %s: %w", key, err)
	}
	tx.ops = append(tx.ops, txOp{
		typ:   "delete",
		key:   key,
		oldL3: old,
	})
	return nil
}

// Commit 按顺序写 L3，每步成功后删 L1/L2。任一步失败则尽力回滚并返回错误。
func (tx *Tx) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}
	tx.committed = true
	defer tx.cancel()

	executed := make([]txOp, 0, len(tx.ops))
	for _, op := range tx.ops {
		var err error
		switch op.typ {
		case "set":
			if op.oldL3 == nil {
				err = tx.mc.l3.Insert(tx.ctx, op.entity)
			} else {
				op.entity.IncrementVersion()
				err = tx.mc.l3.Update(tx.ctx, op.entity)
			}
		case "delete":
			err = tx.mc.l3.Delete(tx.ctx, op.key)
		}
		if err != nil {
			tx.rollback(executed)
			return fmt.Errorf("commit failed at [%s %s]: %w", op.typ, op.key, err)
		}
		tx.invalidateCache(op.key)
		executed = append(executed, op)
	}
	return nil
}

// Rollback 在 Commit 前调用时，L3 尚未修改，直接标记结束即可
func (tx *Tx) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction already finished")
	}
	tx.rolledBack = true
	defer tx.cancel()
	return nil
}

// invalidateCache 删除 L1 和 L2 中的对应条目
func (tx *Tx) invalidateCache(key string) {
	tx.mc.l1.Remove(key)
	if tx.mc.l2 != nil && !tx.mc.isL2Down() {
		ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
		if err := tx.mc.l2.Delete(ctx, key); err != nil {
			tx.mc.markL2Down()
		}
		cancel()
	}
}

// rollback 尽力回滚 L3 中已执行的操作（逆序）
func (tx *Tx) rollback(executed []txOp) {
	for i := len(executed) - 1; i >= 0; i-- {
		op := executed[i]
		ctx, cancel := context.WithTimeout(context.Background(), defaultDBWriteTimeout)
		switch op.typ {
		case "set":
			if op.oldL3 == nil {
				// 我们做了 Insert，撤销：软删除
				_ = tx.mc.l3.Delete(ctx, op.key)
			} else {
				// 我们做了 Update，恢复旧值
				_ = tx.mc.l3.Update(ctx, op.oldL3)
			}
		case "delete":
			if op.oldL3 != nil {
				// 我们做了软删除，恢复：Update 时递增版本号以通过乐观锁
				op.oldL3.IncrementVersion()
				_ = tx.mc.l3.Update(ctx, op.oldL3)
			}
		}
		cancel()
	}
}
