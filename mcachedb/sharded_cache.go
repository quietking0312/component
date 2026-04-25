package mcachedb

import (
	"fmt"
	"hash/crc32"
	"sync"
)

// ShardFunc 分片函数：根据 key 返回分片索引
type ShardFunc func(key string) int

// DefaultShardFunc 默认 CRC32 取模分片
func DefaultShardFunc(shardCount int) ShardFunc {
	return func(key string) int {
		return int(crc32.ChecksumIEEE([]byte(key))) % shardCount
	}
}

// ShardedCache 分片多级缓存（L1/L2/L3 按 shard 隔离）
type ShardedCache struct {
	shards  []*MultiCache
	shardFn ShardFunc
}

// NewShardedCache 创建分片缓存池
//
//	shardCount: 分片数量（建议与 MySQL 分表数一致，如 16/32/64/128）
//	newShard:   工厂函数，传入分片索引 idx，返回该分片对应的 *MultiCache
//	shardFn:    可选自定义分片函数，nil 则使用 CRC32 取模
func NewShardedCache(shardCount int, newShard func(idx int) (*MultiCache, error), shardFn ShardFunc) (*ShardedCache, error) {
	if shardCount <= 0 {
		return nil, fmt.Errorf("shardCount must be > 0")
	}
	if shardFn == nil {
		shardFn = DefaultShardFunc(shardCount)
	}

	shards := make([]*MultiCache, shardCount)
	for i := 0; i < shardCount; i++ {
		mc, err := newShard(i)
		if err != nil {
			for j := 0; j < i; j++ {
				_ = shards[j].Close()
			}
			return nil, fmt.Errorf("create shard %d failed: %w", i, err)
		}
		shards[i] = mc
	}

	return &ShardedCache{
		shards:  shards,
		shardFn: shardFn,
	}, nil
}

func (sc *ShardedCache) shardIndex(key string) int {
	return sc.shardFn(key)
}

// Get 获取单个实体
func (sc *ShardedCache) Get(key string) (Entity, error) {
	return sc.shards[sc.shardIndex(key)].Get(key)
}

// MGet 批量获取（按分片并发查询后聚合结果）
func (sc *ShardedCache) MGet(keys []string) (map[string]Entity, error) {
	if len(keys) == 0 {
		return make(map[string]Entity), nil
	}

	// 按分片索引分组
	groups := make(map[int][]string, len(sc.shards))
	for _, k := range keys {
		idx := sc.shardIndex(k)
		groups[idx] = append(groups[idx], k)
	}

	var mu sync.Mutex
	result := make(map[string]Entity, len(keys))
	var firstErr error
	var wg sync.WaitGroup

	for idx, shardKeys := range groups {
		wg.Add(1)
		go func(shardIdx int, ks []string) {
			defer wg.Done()
			entities, err := sc.shards[shardIdx].MGet(ks)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			mu.Lock()
			for k, v := range entities {
				result[k] = v
			}
			mu.Unlock()
		}(idx, shardKeys)
	}

	wg.Wait()
	return result, firstErr
}

// Set 写入（自动路由到对应分片）
func (sc *ShardedCache) Set(entity Entity) error {
	return sc.shards[sc.shardIndex(entity.CacheKey())].Set(entity)
}

// Delete 删除
func (sc *ShardedCache) Delete(key string) error {
	return sc.shards[sc.shardIndex(key)].Delete(key)
}

// Flush 强制所有分片立即刷盘
func (sc *ShardedCache) Flush() error {
	var firstErr error
	for _, s := range sc.shards {
		if err := s.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Close 关闭所有分片
func (sc *ShardedCache) Close() error {
	var firstErr error
	for _, s := range sc.shards {
		if err := s.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Stats 聚合所有分片的命中统计
func (sc *ShardedCache) Stats() MultiCacheStats {
	var total MultiCacheStats
	for _, s := range sc.shards {
		st := s.Stats()
		total.L1Hits += st.L1Hits
		total.L2Hits += st.L2Hits
		total.L3Hits += st.L3Hits
		total.Misses += st.Misses
	}
	return total
}
