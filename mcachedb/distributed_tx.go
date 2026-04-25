package mcachedb

import (
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

// DistTx 跨 MultiCache 的分布式事务协调器
// 采用"预读旧值 + 顺序提交 + 失败补偿"策略
type DistTx struct {
	mu         sync.Mutex
	steps      []distTxStep
	oldVals    map[string]Entity // key: cachePtr:entityKey
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
	// 用指针地址区分不同的 MultiCache 实例
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
	if entity == nil {
		return fmt.Errorf("entity cannot be nil")
	}

	key := entity.CacheKey()
	if key == "" {
		return fmt.Errorf("entity key cannot be empty")
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

// Commit 提交事务：顺序执行，失败时自动尽最大努力回滚
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

// CommitAndFlush 提交事务，并立即把所有涉及缓存的脏数据刷到数据库
// 适用于充值、交易等必须立即落盘的场景
func (dt *DistTx) CommitAndFlush() error {
	if err := dt.Commit(); err != nil {
		return err
	}

	// 去重，避免同一个 cache Flush 多次
	flushed := make(map[*MultiCache]struct{})
	for _, step := range dt.steps {
		if _, ok := flushed[step.cache]; !ok {
			_ = step.cache.Flush()
			flushed[step.cache] = struct{}{}
		}
	}
	return nil
}

// Rollback 手动回滚（只能在 Commit 前调用）
func (dt *DistTx) Rollback() error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.committed || dt.rolledBack {
		return fmt.Errorf("distTx already finished")
	}
	dt.rolledBack = true
	return nil
}

// rollback 尽最大努力回滚已执行的步骤
func (dt *DistTx) rollback(executedIdx []int) {
	for i := len(executedIdx) - 1; i >= 0; i-- {
		step := dt.steps[executedIdx[i]]
		ck := dt.cacheKey(step.cache, step.key)
		old := dt.oldVals[ck]

		switch step.typ {
		case DistTxSet:
			if old != nil {
				_ = step.cache.Set(old) // 恢复旧值
			} else {
				_ = step.cache.Delete(step.key) // 原来是新增，撤销
			}
		case DistTxDelete:
			if old != nil {
				_ = step.cache.Set(old) // 恢复被删数据
			}
		}
	}
}
