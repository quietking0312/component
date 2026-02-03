package idprocess

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type LogIdWorker struct {
	number    uint64
	timestamp int64
	workerId  int32
	mu        sync.Mutex
}

func NewLogIdWorker(workId int32) *LogIdWorker {
	return &LogIdWorker{
		workerId: workId,
	}
}

func (w *LogIdWorker) GetId() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now().UnixMilli()
	if w.timestamp == now {
		w.number++
		if w.number > math.MaxUint32 {
			for now <= w.timestamp {
				now = time.Now().UnixMilli()
			}
			w.timestamp = now
			w.number = 0
		}
	} else {
		w.number = 0
		w.timestamp = now
	}
	return fmt.Sprintf("%d-%d-%d", w.workerId, w.timestamp, w.number)
}
