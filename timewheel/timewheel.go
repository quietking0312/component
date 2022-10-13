package timewheel

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"
)

type TimeWheel struct {
	interval    time.Duration
	ticker      *time.Ticker
	slots       []*list.List
	currentPos  int
	slotNum     int
	addTaskChan chan *task // 添加任务管道
	stopChan    chan bool  // 停止轮盘管道
	taskRecord  *sync.Map  // 任务
}

type Job func(TaskData)

type TaskData map[interface{}]interface{}

type task struct {
	interval time.Duration
	times    int
	circle   int
	key      interface{}
	job      Job
	taskData TaskData
}

// NewTimeWheel 时间间隔
// 轮盘大小
func NewTimeWheel(interval time.Duration, slotNam int) *TimeWheel {
	if interval <= 0 || slotNam <= 0 {
		return nil
	}
	tw := &TimeWheel{
		interval:    interval,
		slots:       make([]*list.List, slotNam),
		currentPos:  0,
		slotNum:     slotNam,
		addTaskChan: make(chan *task),
		stopChan:    make(chan bool),
		taskRecord:  &sync.Map{},
	}
	tw.init()
	return tw
}

func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(tw.interval)
	go tw.start()
}

func (tw *TimeWheel) Stop() {
	tw.stopChan <- true
}

func (tw *TimeWheel) start() {
	defer func() {
		fmt.Println("warn: TimeWheel stop")
	}()
	for {
		select {
		case <-tw.ticker.C:
			tw.tickHandle()
		case task := <-tw.addTaskChan:
			tw.addTask(task)
		case <-tw.stopChan:
			tw.ticker.Stop()
			return
		}
	}
}

func (tw *TimeWheel) RemoveTask(key interface{}) error {
	if key == nil {
		return nil
	}
	value, ok := tw.taskRecord.Load(key)
	if !ok {
		return errors.New("task not exists")
	} else {
		task := value.(*task)
		task.times = 0
		tw.taskRecord.Delete(task.key)
	}
	return nil
}

func (tw *TimeWheel) UpdateTask(key interface{}, interval time.Duration, data TaskData) error {
	if key == nil {
		return errors.New("illegal key, please try again")
	}
	value, ok := tw.taskRecord.Load(key)
	if !ok {
		return errors.New("task not exists, please check you task key")
	}
	task := value.(*task)
	task.taskData = data
	task.interval = interval
	return nil
}

func (tw *TimeWheel) init() {
	for i := 0; i < tw.slotNum; i++ {
		tw.slots[i] = list.New()
	}
}

// AddTask 添加任务 到时间轮
// interval 时间间隔
// 次数
// 索引
// 参数
// 任务
func (tw *TimeWheel) AddTask(interval time.Duration, times int, key interface{}, data TaskData, job Job) error {
	if interval <= 0 || key == nil || job == nil || times < -1 || times == 0 {
		return errors.New("illegal task params")
	}
	if _, ok := tw.taskRecord.Load(key); ok {
		return errors.New("duplicate task key")
	}
	tw.addTaskChan <- &task{interval: interval, times: times, key: key, taskData: data, job: job}
	return nil
}

func (tw *TimeWheel) tickHandle() {
	l := tw.slots[tw.currentPos]
	tw.scanAddRunTask(l)
	if tw.currentPos == tw.slotNum-1 {
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}
}
func (tw *TimeWheel) addTask(task *task) {
	if task.times == 0 {
		return
	}
	pos, circle := tw.getPositionAndCircle(task.interval)
	task.circle = circle
	tw.slots[pos].PushBack(task)
	tw.taskRecord.Store(task.key, task)
}

func (tw *TimeWheel) scanAddRunTask(l *list.List) {
	if l == nil {
		return
	}
	for item := l.Front(); item != nil; {
		task := item.Value.(*task)
		if task.times == 0 { // 次数为0 删除任务
			next := item.Next()
			l.Remove(item)
			tw.taskRecord.Delete(task.key)
			item = next
			continue
		}
		if task.circle > 0 { // 计数器
			task.circle--
			item = item.Next()
			continue
		}
		go task.job(task.taskData) // 执行任务
		next := item.Next()
		l.Remove(item)
		item = next

		if task.times == 1 {
			task.times = 0
			tw.taskRecord.Delete(task.key)
		} else {
			if task.times > 0 {
				task.times--
			}
			tw.addTask(task)
		}

	}
}

func (tw *TimeWheel) getPositionAndCircle(d time.Duration) (pos int, circle int) {
	delaySeconds := int(d.Seconds())
	intervalSeconds := int(tw.interval.Seconds())
	// 圈数
	circle = int(delaySeconds / intervalSeconds / tw.slotNum)
	// 指针
	pos = int(tw.currentPos+delaySeconds/intervalSeconds) % tw.slotNum
	return
}
