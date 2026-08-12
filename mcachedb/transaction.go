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
// 通过 MultiCache.Transaction 使用，不可手动创建或手动提交/回滚。
type Tx struct {
	mc        *MultiCache
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	ops       []txOp
	committed bool
}

type txOp struct {
	typ    string // "set" | "delete"
	key    string
	entity Entity
	oldL3  Entity
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
	if tx.committed {
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

// Set 将实体写入事务（Commit 时写 L3）
func (tx *Tx) Set(entity Entity) error {
	if err := validateEntity(entity); err != nil {
		return err
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.committed {
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
	if tx.committed {
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

// commit 写 L3 并使 L1/L2 失效，由 Transaction 调用。
// 若 L3 实现了 TxCapableDBStore，则使用原生事务；否则顺序写入并尽力回滚。
func (tx *Tx) commit() error {
	tx.mu.Lock()
	if tx.committed {
		tx.mu.Unlock()
		return fmt.Errorf("transaction already committed")
	}
	tx.committed = true
	tx.mu.Unlock()

	if txStore, ok := tx.mc.l3.(TxCapableDBStore); ok {
		return tx.commitWithNativeTx(txStore)
	}
	return tx.commitBestEffort()
}

// commitWithNativeTx 使用 L3 原生事务执行所有操作，保证原子性。
func (tx *Tx) commitWithNativeTx(store TxCapableDBStore) error {
	nativeTx, err := store.BeginTx(tx.ctx)
	if err != nil {
		return fmt.Errorf("BeginTx: %w", err)
	}
	for _, op := range tx.ops {
		switch op.typ {
		case "set":
			if op.oldL3 == nil {
				err = nativeTx.Insert(tx.ctx, op.entity)
			} else {
				op.entity.IncrementVersion()
				err = nativeTx.Update(tx.ctx, op.entity)
			}
		case "delete":
			err = nativeTx.Delete(tx.ctx, op.key)
		}
		if err != nil {
			_ = nativeTx.Rollback()
			return fmt.Errorf("native tx op [%s %s]: %w", op.typ, op.key, err)
		}
	}
	if err := nativeTx.Commit(); err != nil {
		_ = nativeTx.Rollback()
		return fmt.Errorf("native tx commit: %w", err)
	}
	for _, op := range tx.ops {
		tx.invalidateCache(op.key)
	}
	return nil
}

// commitBestEffort 顺序写 L3，每步成功后删 L1/L2；失败则尽力回滚已执行步骤。
func (tx *Tx) commitBestEffort() error {
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
				_ = tx.mc.l3.Delete(ctx, op.key)
			} else {
				_ = tx.mc.l3.Update(ctx, op.oldL3)
			}
		case "delete":
			if op.oldL3 != nil {
				op.oldL3.IncrementVersion()
				_ = tx.mc.l3.Update(ctx, op.oldL3)
			}
		}
		cancel()
	}
}
