package mcachedb

import (
	"context"
	"fmt"
	"sync"
)

// DistTxOpType 操作类型
type DistTxOpType string

const (
	DistTxSet    DistTxOpType = "set"
	DistTxDelete DistTxOpType = "delete"
)

type distTxStep struct {
	cache  *MultiCache
	typ    DistTxOpType
	key    string
	entity Entity // 仅 set 操作有效
}

// DistTx 跨 MultiCache 的分布式事务协调器（最大努力）。
// 通过 RunDistTx 使用，不可手动创建或手动提交/回滚。
type DistTx struct {
	mu        sync.Mutex
	steps     []distTxStep
	oldVals   map[string]Entity // key: cachePtr:entityKey → 提交前旧值
	committed bool
}

// RunDistTx 在分布式事务中执行 fn。
// fn 应通过 dtx.AddSet / dtx.AddDelete 注册操作。
// fn 返回 nil 则提交，返回非 nil 则事务取消（L3 尚未写入，无需回滚）。
// 若所有步骤共享同一 TxCapableDBStore 且均为 WriteModeCacheAside，则使用 L3 原生事务。
func RunDistTx(fn func(*DistTx) error) error {
	dtx := &DistTx{
		steps:   make([]distTxStep, 0, 4),
		oldVals: make(map[string]Entity),
	}
	if err := fn(dtx); err != nil {
		return err
	}
	return dtx.commit()
}

func (dt *DistTx) cacheKey(cache *MultiCache, key string) string {
	return fmt.Sprintf("%p:%s", cache, key)
}

// AddSet 注册 Set 操作，同时预读旧值用于提交失败时的尽力回滚
func (dt *DistTx) AddSet(cache *MultiCache, entity Entity) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.committed {
		return fmt.Errorf("distTx already finished")
	}
	if cache == nil {
		return fmt.Errorf("cache cannot be nil")
	}
	if err := validateEntity(entity); err != nil {
		return err
	}

	key := entity.CacheKey()
	ck := dt.cacheKey(cache, key)
	if _, ok := dt.oldVals[ck]; !ok {
		old, _ := cache.Get(key)
		if old != nil {
			dt.oldVals[ck] = old.Copy()
		} else {
			dt.oldVals[ck] = nil
		}
	}

	dt.steps = append(dt.steps, distTxStep{
		cache:  cache,
		typ:    DistTxSet,
		key:    key,
		entity: entity.Copy(),
	})
	return nil
}

// AddDelete 注册 Delete 操作，同时预读旧值用于提交失败时的尽力回滚
func (dt *DistTx) AddDelete(cache *MultiCache, key string) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.committed {
		return fmt.Errorf("distTx already finished")
	}
	if cache == nil {
		return fmt.Errorf("cache cannot be nil")
	}
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	ck := dt.cacheKey(cache, key)
	if _, ok := dt.oldVals[ck]; !ok {
		old, _ := cache.Get(key)
		if old != nil {
			dt.oldVals[ck] = old.Copy()
		} else {
			dt.oldVals[ck] = nil
		}
	}

	dt.steps = append(dt.steps, distTxStep{
		cache: cache,
		typ:   DistTxDelete,
		key:   key,
	})
	return nil
}

// commit 由 RunDistTx 调用，执行所有已注册操作。
func (dt *DistTx) commit() error {
	dt.mu.Lock()
	if dt.committed {
		dt.mu.Unlock()
		return fmt.Errorf("distTx already committed")
	}
	dt.committed = true
	dt.mu.Unlock()

	if txStore, ok := dt.singleTxCapableStore(); ok {
		return dt.commitWithNativeTx(txStore)
	}
	return dt.commitBestEffort()
}

// singleTxCapableStore 检查所有步骤是否共享同一个 TxCapableDBStore 且均为 CacheAside 模式。
func (dt *DistTx) singleTxCapableStore() (TxCapableDBStore, bool) {
	if len(dt.steps) == 0 {
		return nil, false
	}
	var store TxCapableDBStore
	for _, step := range dt.steps {
		if step.cache.config.WriteMode != WriteModeCacheAside {
			return nil, false
		}
		txStore, ok := step.cache.l3.(TxCapableDBStore)
		if !ok {
			return nil, false
		}
		if store == nil {
			store = txStore
		} else if store != txStore {
			return nil, false
		}
	}
	return store, store != nil
}

// commitWithNativeTx 在单个 L3 原生事务中执行所有步骤，提交后统一使 L1/L2 失效。
func (dt *DistTx) commitWithNativeTx(store TxCapableDBStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTxTimeout)
	defer cancel()

	nativeTx, err := store.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("distTx BeginTx: %w", err)
	}

	for i, step := range dt.steps {
		ck := dt.cacheKey(step.cache, step.key)
		old := dt.oldVals[ck]
		switch step.typ {
		case DistTxSet:
			if old == nil {
				err = nativeTx.Insert(ctx, step.entity)
			} else {
				step.entity.IncrementVersion()
				err = nativeTx.Update(ctx, step.entity)
			}
		case DistTxDelete:
			err = nativeTx.Delete(ctx, step.key)
		}
		if err != nil {
			_ = nativeTx.Rollback()
			return fmt.Errorf("distTx native tx step %d (%s %s): %w", i, step.typ, step.key, err)
		}
	}

	if err := nativeTx.Commit(); err != nil {
		_ = nativeTx.Rollback()
		return fmt.Errorf("distTx native tx commit: %w", err)
	}

	for _, step := range dt.steps {
		step.cache.l1.Remove(step.key)
		if step.cache.l2 != nil && !step.cache.isL2Down() {
			ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
			if err2 := step.cache.l2.Delete(ctx2, step.key); err2 != nil {
				step.cache.markL2Down()
			}
			cancel2()
		}
	}
	return nil
}

// commitBestEffort 顺序执行所有步骤，失败时尽力回滚已执行步骤。
func (dt *DistTx) commitBestEffort() error {
	executed := make([]int, 0, len(dt.steps))
	for i, step := range dt.steps {
		var err error
		switch step.typ {
		case DistTxSet:
			err = step.cache.Set(step.entity)
		case DistTxDelete:
			err = step.cache.Delete(step.key)
		}
		if err != nil {
			dt.rollback(executed)
			return fmt.Errorf("distTx step %d (%s %s) failed: %w", i, step.typ, step.key, err)
		}
		executed = append(executed, i)
	}
	return nil
}

// restoreCache 仅恢复 L1/L2 缓存，不触及 L3
func (dt *DistTx) restoreCache(step distTxStep, old Entity) {
	switch step.typ {
	case DistTxSet:
		if old != nil {
			step.cache.l1.Load(old)
		} else {
			step.cache.l1.Remove(step.key)
			if step.cache.l2 != nil && !step.cache.isL2Down() {
				ctx, cancel := context.WithTimeout(context.Background(), l2Timeout)
				_ = step.cache.l2.Delete(ctx, step.key)
				cancel()
			}
		}
	case DistTxDelete:
		if old != nil {
			step.cache.l1.Load(old)
		}
	}
}

// rollback 尽力回滚已执行步骤（commitBestEffort 中途失败时调用）
func (dt *DistTx) rollback(executedIdx []int) {
	for i := len(executedIdx) - 1; i >= 0; i-- {
		step := dt.steps[executedIdx[i]]
		ck := dt.cacheKey(step.cache, step.key)
		old := dt.oldVals[ck]

		if step.cache.config.WriteMode == WriteModeCacheAside {
			dt.rollbackCacheAsideStep(step, old)
		} else {
			dt.restoreCache(step, old)
		}
	}
}

// rollbackCacheAsideStep 回滚 CacheAside 模式中已执行的步骤
func (dt *DistTx) rollbackCacheAsideStep(step distTxStep, old Entity) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultDBWriteTimeout)
	defer cancel()

	switch step.typ {
	case DistTxSet:
		if old == nil {
			_ = step.cache.l3.Delete(ctx, step.key)
		} else {
			_ = step.cache.l3.Update(ctx, old)
		}
	case DistTxDelete:
		if old != nil {
			old.IncrementVersion()
			_ = step.cache.l3.Update(ctx, old)
		}
	}

	step.cache.l1.Remove(step.key)
	if step.cache.l2 != nil && !step.cache.isL2Down() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
		_ = step.cache.l2.Delete(ctx2, step.key)
		cancel2()
	}
}
