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

// DistTx 跨 MultiCache 的分布式事务协调器（最大努力）
//
// 采用"预读旧值 + 顺序提交 + 失败补偿"策略。
// 不提供跨进程严格原子性（无两阶段提交），适用于同进程内多缓存的最大努力一致性。
type DistTx struct {
	mu         sync.Mutex
	steps      []distTxStep
	oldVals    map[string]Entity // key: cachePtr:entityKey → 提交前旧值
	committed  bool
	rolledBack bool
}

// NewDistTx 创建跨缓存事务
func NewDistTx() *DistTx {
	return &DistTx{
		steps:   make([]distTxStep, 0, 4),
		oldVals: make(map[string]Entity),
	}
}

func (dt *DistTx) cacheKey(cache *MultiCache, key string) string {
	return fmt.Sprintf("%p:%s", cache, key)
}

// AddSet 注册 Set 操作，同时预读旧值用于回滚
func (dt *DistTx) AddSet(cache *MultiCache, entity Entity) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.committed || dt.rolledBack {
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

// AddDelete 注册 Delete 操作，同时预读旧值用于回滚
func (dt *DistTx) AddDelete(cache *MultiCache, key string) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.committed || dt.rolledBack {
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

// Commit 顺序执行所有步骤，失败时尽力回滚已执行步骤
func (dt *DistTx) Commit() error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.committed || dt.rolledBack {
		return fmt.Errorf("distTx already finished")
	}
	dt.committed = true

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

// CommitAndFlush 提交后立即刷盘，适用于交易等需要立即落库的场景
func (dt *DistTx) CommitAndFlush() error {
	if err := dt.Commit(); err != nil {
		return err
	}
	flushed := make(map[*MultiCache]struct{})
	for _, step := range dt.steps {
		if _, ok := flushed[step.cache]; !ok {
			if err := step.cache.Flush(); err != nil {
				return fmt.Errorf("flush failed: %w", err)
			}
			flushed[step.cache] = struct{}{}
		}
	}
	return nil
}

// Rollback 手动回滚（只能在 Commit 前调用）；逆序恢复预读的旧值
func (dt *DistTx) Rollback() error {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.committed || dt.rolledBack {
		return fmt.Errorf("distTx already finished")
	}
	dt.rolledBack = true

	for i := len(dt.steps) - 1; i >= 0; i-- {
		step := dt.steps[i]
		ck := dt.cacheKey(step.cache, step.key)
		old := dt.oldVals[ck]
		dt.restoreCache(step, old)
	}
	return nil
}

// restoreCache 仅恢复 L1/L2 缓存，不触及 L3（Rollback 时 L3 未被修改）
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

// rollback 尽力回滚已执行步骤（Commit 中途失败时调用）
// 对 CacheAside 模式：L3 已写入，需要撤销 L3 并恢复 L1/L2。
// 对 Async/WriteL2 模式：L1 已写入，恢复 L1 到旧值；L3 尚未刷盘（大概率），
// 无法严格保证 L3 一致性，此为"尽力"语义。
func (dt *DistTx) rollback(executedIdx []int) {
	for i := len(executedIdx) - 1; i >= 0; i-- {
		step := dt.steps[executedIdx[i]]
		ck := dt.cacheKey(step.cache, step.key)
		old := dt.oldVals[ck]

		if step.cache.config.WriteMode == WriteModeCacheAside {
			// CacheAside：L3 已被修改，尝试恢复
			dt.rollbackCacheAsideStep(step, old)
		} else {
			// Async/WriteL2：仅恢复 L1/L2 内存状态
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
			// 原来不存在，我们写了 Insert，撤销：软删除
			_ = step.cache.l3.Delete(ctx, step.key)
		} else {
			// 原来存在，我们做了 Update，恢复旧值
			_ = step.cache.l3.Update(ctx, old)
		}
	case DistTxDelete:
		if old != nil {
			// 原来存在，我们做了软删除，恢复：递增版本通过乐观锁
			old.IncrementVersion()
			_ = step.cache.l3.Update(ctx, old)
		}
	}

	// L3 已尽力恢复，同时失效 L1/L2
	step.cache.l1.Remove(step.key)
	if step.cache.l2 != nil && !step.cache.isL2Down() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), l2Timeout)
		_ = step.cache.l2.Delete(ctx2, step.key)
		cancel2()
	}
}
