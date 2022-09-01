package idprocess

import (
	"errors"
	"sync"
	"time"
)

/*
雪花算法
*/

const (
	// 以下俩值 可以修改, 但和不能改变
	workerBits uint8 = 10 // 每台机器(节点)的ID位数 10位最大可以有2^10=1024个节点
	numberBits uint8 = 12 // 表示每个集群下的每个节点，1毫秒内可生成的id序号的二进制位数 即每毫秒可生成 2^12-1=4096个唯一ID

	workerMax int64 = -1 ^ (-1 << workerBits) // 节点id 最大值, 用于防止溢出
	numberMax int64 = -1 ^ (-1 << numberBits) // 用于生成id 序号的最大值

	timeShift   uint8 = workerBits + numberBits // 时间戳向左偏移量
	workerShift uint8 = numberBits              // 节点id 向左偏移量
	// 41位字节作为时间戳数值 大约68年用完

	// 生成id 后 不可更改，不然会生成相同的id
	epoch int64 = 1652751577000 // 起始时间 2022-05-17 09:39:37
)

type Worker struct {
	mu        sync.Mutex
	timestamp int64 // 时间戳
	workerId  int64 // 节点id
	number    int64 // 当前毫秒已经生成的id 序列号 1毫秒最多生成4096个ID
}

func NewWorker(workerId int64) (*Worker, error) {

	if workerId < 0 || workerId > workerMax {
		return nil, errors.New("worker Id excess of quantity")
	}
	return &Worker{
		timestamp: 0,
		workerId:  workerId,
		number:    0,
	}, nil
}

func (w *Worker) GetId() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().UnixNano() / 1e6 // 纳秒转毫秒

	if w.timestamp == now {
		w.number++
		if w.number > numberMax {
			for now <= w.timestamp { // 等待下一毫秒
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		w.number = 0
		w.timestamp = now
	}
	// 第一段 now - epoch 为该算法目前已经奔跑了xxx毫秒
	// 如果在程序跑了一段时间修改了epoch这个值 可能会导致生成相同的ID
	ID := int64((now-epoch)<<timeShift | (w.workerId << workerShift) | (w.number))
	return ID
}

func (w *Worker) UnId(id int64) map[string]int64 {
	t := ((id >> timeShift) + epoch) / 1e3
	number := id & (1<<workerShift - 1)
	worker := id & (1<<timeShift - 1) >> workerShift
	return map[string]int64{
		"id":     id,
		"time":   t,
		"worker": worker,
		"number": number,
	}
}
