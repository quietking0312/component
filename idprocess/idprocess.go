package idprocess

import (
	"math"
	"strconv"
	"strings"
	"sync"
)

// workerId 和自增id 交叉填充 来生成 id，
// 使用交叉的方式 可以避免 自增数值最大的 不可控, 但是会损失一定的性能
// 可用来生成 数量规模庞大 的id， 比如位每个游戏用户 的属性不同 的大批量道具生成id
type IdProcess struct {
	mu       sync.Mutex
	workerId int64
	lastId   int64
}

func NewIdProcess(workerId int64) *IdProcess {
	return &IdProcess{
		workerId: workerId,
		lastId:   0,
	}
}

func (w *IdProcess) SetLastId(lastId int64) {
	w.lastId = lastId
}

func (w *IdProcess) GetId() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	workerId36 := strconv.FormatInt(w.workerId, 36)
	lastId36 := strconv.FormatInt(w.lastId, 36)
	d := len(workerId36) - len(lastId36)
	if d > 0 {
		lastId36 = strings.Repeat("0", d) + lastId36
	} else if d < 0 {
		workerId36 = strings.Repeat("0", int(math.Abs(float64(d)))) + workerId36
	}
	var result = make([]byte, 2*len(workerId36))
	for i := 0; i < len(workerId36); i++ {
		result[2*i] = workerId36[i]
		result[2*i+1] = lastId36[i]
	}
	w.lastId += 1
	return string(result)
}
