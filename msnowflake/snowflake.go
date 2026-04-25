package msnowflake

import (
	"fmt"
	"sync"
	"time"
)

const (
	// 默认起始时间戳 (2024-01-01 00:00:00 UTC)
	defaultEpoch int64 = 1704067200000

	// 各部分的位数
	timestampBits uint8 = 41 // 时间戳位数
	workerIDBits  uint8 = 10 // 机器ID位数
	sequenceBits  uint8 = 12 // 序列号位数

	// 最大值
	maxWorkerID int64 = -1 ^ (-1 << workerIDBits) // 1023
	maxSequence int64 = -1 ^ (-1 << sequenceBits) // 4095

	// 左移位数
	workerIDShift  = sequenceBits
	timestampShift = sequenceBits + workerIDBits
)

// Generator 雪花算法ID生成器
type Generator struct {
	mu       sync.Mutex
	epoch    int64 // 起始时间戳
	workerID int64 // 机器ID
	sequence int64 // 序列号
	lastTime int64 // 上次生成ID的时间戳
}

// NewGenerator 创建一个新的雪花ID生成器
// workerID: 机器ID，范围 0-1023
func NewGenerator(workerID int64) (*Generator, error) {
	return NewGeneratorWithEpoch(workerID, defaultEpoch)
}

// NewGeneratorWithEpoch 创建一个新的雪花ID生成器，自定义起始时间
// workerID: 机器ID，范围 0-1023
// epoch: 起始时间戳（毫秒）
func NewGeneratorWithEpoch(workerID int64, epoch int64) (*Generator, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("worker ID must be between 0 and %d", maxWorkerID)
	}
	if epoch < 0 {
		return nil, fmt.Errorf("epoch must be non-negative")
	}
	if epoch > time.Now().UnixMilli() {
		return nil, fmt.Errorf("epoch cannot be in the future")
	}

	return &Generator{
		epoch:    epoch,
		workerID: workerID,
		sequence: 0,
		lastTime: 0,
	}, nil
}

// NextID 生成下一个唯一ID
func (g *Generator) NextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()

	// 如果当前时间小于上次生成时间，说明时钟回拨了
	if now < g.lastTime {
		// 等待直到时间追上
		for now < g.lastTime {
			now = time.Now().UnixMilli()
		}
	}

	// 如果是同一时间，增加序列号
	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & maxSequence
		// 序列号溢出，等待下一毫秒
		if g.sequence == 0 {
			for now <= g.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		// 新毫秒，重置序列号
		g.sequence = 0
	}

	g.lastTime = now

	// 组合ID
	id := ((now - g.epoch) << timestampShift) |
		(g.workerID << workerIDShift) |
		g.sequence

	return id
}

// NextIDs 批量生成多个唯一ID
func (g *Generator) NextIDs(count int) []int64 {
	ids := make([]int64, count)
	for i := 0; i < count; i++ {
		ids[i] = g.NextID()
	}
	return ids
}

// ParseID 解析ID的各个组成部分
func ParseID(id int64, epoch int64) (timestamp int64, workerID int64, sequence int64) {
	timestamp = (id >> timestampShift) + epoch
	workerID = (id >> workerIDShift) & maxWorkerID
	sequence = id & maxSequence
	return
}

// ParseIDWithDefaultEpoch 使用默认起始时间解析ID
func ParseIDWithDefaultEpoch(id int64) (timestamp int64, workerID int64, sequence int64) {
	return ParseID(id, defaultEpoch)
}

// GetWorkerID 获取生成器使用的机器ID
func (g *Generator) GetWorkerID() int64 {
	return g.workerID
}

// GetEpoch 获取生成器使用的起始时间戳
func (g *Generator) GetEpoch() int64 {
	return g.epoch
}

// MaxWorkerID 返回支持的最大机器ID
func MaxWorkerID() int64 {
	return maxWorkerID
}

// MaxSequence 返回每毫秒支持的最大序列号
func MaxSequence() int64 {
	return maxSequence
}

// ========== 全局默认生成器 ==========

var (
	defaultGenerator     *Generator
	defaultGeneratorOnce sync.Once
	defaultGeneratorErr  error
)

// InitDefault 初始化全局默认生成器
func InitDefault(workerID int64) error {
	defaultGeneratorOnce.Do(func() {
		defaultGenerator, defaultGeneratorErr = NewGenerator(workerID)
	})
	return defaultGeneratorErr
}

// InitDefaultWithEpoch 初始化全局默认生成器，自定义起始时间
func InitDefaultWithEpoch(workerID int64, epoch int64) error {
	defaultGeneratorOnce.Do(func() {
		defaultGenerator, defaultGeneratorErr = NewGeneratorWithEpoch(workerID, epoch)
	})
	return defaultGeneratorErr
}

// Default 获取全局默认生成器
// 注意：使用前必须先调用 InitDefault 或 InitDefaultWithEpoch
func Default() *Generator {
	if defaultGenerator == nil {
		panic("msnowflake: default generator not initialized, call InitDefault first")
	}
	return defaultGenerator
}

// Generate 使用全局默认生成器生成ID
func Generate() int64 {
	return Default().NextID()
}

// Generates 使用全局默认生成器批量生成ID
func Generates(count int) []int64 {
	return Default().NextIDs(count)
}
