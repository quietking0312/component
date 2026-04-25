// Package mtimewheel 提供时间轮实现
// 适合大量延迟任务的调度和管理
package mtimewheel

import (
	"container/list"
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quietking0312/component/mlog"

	"go.uber.org/zap"
)

// Task 延迟任务
type Task struct {
	// ID 任务唯一标识
	ID string
	// Delay 延迟时间
	Delay time.Duration
	// Round 时间轮圈数（内部使用）
	Round int
	// Callback 任务回调函数
	Callback func(ctx context.Context) error
	// ExecuteAt 计划执行时间
	ExecuteAt time.Time
	// CreatedAt 创建时间
	CreatedAt time.Time
	// Executed 是否已执行
	Executed bool
	// Canceled 是否已取消
	Canceled bool
	// element 链表元素（内部使用）
	element *list.Element
}

// TimeWheel 时间轮
type TimeWheel struct {
	// tick 时间间隔（刻度）
	tick time.Duration
	// wheelSize 时间轮大小（槽位数）
	wheelSize int
	// currentPos 当前指针位置
	currentPos int
	// slots 时间轮槽位，每个槽位是一个任务链表
	slots []*list.List
	// taskMap 任务映射表，用于快速查找和取消
	taskMap map[string]*Task
	// addTaskChan 添加任务通道
	addTaskChan chan *Task
	// cancelTaskChan 取消任务通道
	cancelTaskChan chan string
	// stopChan 停止信号
	stopChan chan struct{}
	// running 是否运行中
	running int32
	// ticker 定时器
	ticker *time.Ticker
	// mu 互斥锁
	mu sync.RWMutex
	// ctx 上下文
	ctx context.Context
	// cancel 取消函数
	cancel context.CancelFunc
	// wg 等待组
	wg sync.WaitGroup
	// taskCount 任务计数
	taskCount int64
	// executedCount 已执行任务计数
	executedCount int64
}

// Options 配置选项
type Options struct {
	// Tick 时间间隔，默认 100ms
	Tick time.Duration
	// WheelSize 时间轮大小，默认 100
	WheelSize int
}

// Option 配置函数
type Option func(*Options)

// WithTick 设置时间间隔
func WithTick(tick time.Duration) Option {
	return func(o *Options) {
		o.Tick = tick
	}
}

// WithWheelSize 设置时间轮大小
func WithWheelSize(size int) Option {
	return func(o *Options) {
		o.WheelSize = size
	}
}

// New 创建时间轮
func New(opts ...Option) *TimeWheel {
	options := &Options{
		Tick:      100 * time.Millisecond,
		WheelSize: 100,
	}

	for _, opt := range opts {
		opt(options)
	}

	ctx, cancel := context.WithCancel(context.Background())

	tw := &TimeWheel{
		tick:           options.Tick,
		wheelSize:      options.WheelSize,
		slots:          make([]*list.List, options.WheelSize),
		taskMap:        make(map[string]*Task),
		addTaskChan:    make(chan *Task, 1000),
		cancelTaskChan: make(chan string, 100),
		stopChan:       make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 初始化槽位
	for i := 0; i < options.WheelSize; i++ {
		tw.slots[i] = list.New()
	}

	return tw
}

// Start 启动时间轮
func (tw *TimeWheel) Start() {
	if !atomic.CompareAndSwapInt32(&tw.running, 0, 1) {
		return
	}

	tw.ticker = time.NewTicker(tw.tick)

	tw.wg.Add(1)
	go tw.run()

	mlog.Info("time wheel started",
		zap.Duration("tick", tw.tick),
		zap.Int("wheel_size", tw.wheelSize),
	)
}

// Stop 停止时间轮
func (tw *TimeWheel) Stop() {
	if !atomic.CompareAndSwapInt32(&tw.running, 1, 0) {
		return
	}

	tw.cancel()
	close(tw.stopChan)

	if tw.ticker != nil {
		tw.ticker.Stop()
	}

	tw.wg.Wait()

	// 清空任务
	tw.mu.Lock()
	tw.taskMap = make(map[string]*Task)
	tw.mu.Unlock()

	mlog.Info("time wheel stopped",
		zap.Int64("total_tasks", tw.taskCount),
		zap.Int64("executed_tasks", tw.executedCount),
	)
}

// AddTask 添加延迟任务
func (tw *TimeWheel) AddTask(id string, delay time.Duration, callback func(ctx context.Context) error) error {
	if delay < 0 {
		return fmt.Errorf("delay must be non-negative")
	}

	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	task := &Task{
		ID:        id,
		Delay:     delay,
		Callback:  callback,
		ExecuteAt: time.Now().Add(delay),
		CreatedAt: time.Now(),
	}

	// 如果时间轮未启动，直接执行（延迟为0）或报错
	if atomic.LoadInt32(&tw.running) == 0 {
		if delay == 0 {
			return tw.executeTask(task)
		}
		return fmt.Errorf("time wheel is not running")
	}

	select {
	case tw.addTaskChan <- task:
		atomic.AddInt64(&tw.taskCount, 1)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("add task timeout")
	}
}

// CancelTask 取消任务
func (tw *TimeWheel) CancelTask(id string) bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if task, ok := tw.taskMap[id]; ok {
		task.Canceled = true
		delete(tw.taskMap, id)
		return true
	}

	return false
}

// GetTask 获取任务
func (tw *TimeWheel) GetTask(id string) *Task {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	return tw.taskMap[id]
}

// TaskCount 获取任务数量
func (tw *TimeWheel) TaskCount() int64 {
	return atomic.LoadInt64(&tw.taskCount)
}

// ExecutedCount 获取已执行任务数量
func (tw *TimeWheel) ExecutedCount() int64 {
	return atomic.LoadInt64(&tw.executedCount)
}

// run 时间轮主循环
func (tw *TimeWheel) run() {
	defer tw.wg.Done()

	for {
		select {
		case <-tw.ticker.C:
			tw.tickHandler()
		case task := <-tw.addTaskChan:
			tw.addTaskInternal(task)
		case id := <-tw.cancelTaskChan:
			tw.cancelTaskInternal(id)
		case <-tw.stopChan:
			return
		case <-tw.ctx.Done():
			return
		}
	}
}

// tickHandler 处理时间刻度
func (tw *TimeWheel) tickHandler() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	// 获取当前槽位
	slot := tw.slots[tw.currentPos]

	// 遍历槽位中的所有任务
	for e := slot.Front(); e != nil; {
		task := e.Value.(*Task)
		next := e.Next()

		if task.Canceled {
			// 任务已取消，移除
			slot.Remove(e)
			delete(tw.taskMap, task.ID)
			e = next
			continue
		}

		if task.Round > 0 {
			// 还需要等待下一轮
			task.Round--
		} else {
			// 执行任务
			slot.Remove(e)
			delete(tw.taskMap, task.ID)
			task.Executed = true

			// 异步执行
			tw.wg.Add(1)
			go func(t *Task) {
				defer tw.wg.Done()
				tw.executeTask(t)
			}(task)
		}

		e = next
	}

	// 移动指针
	tw.currentPos = (tw.currentPos + 1) % tw.wheelSize
}

// addTaskInternal 内部添加任务
func (tw *TimeWheel) addTaskInternal(task *Task) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	// 计算延迟的刻度数
	ticks := int(task.Delay / tw.tick)
	if task.Delay%tw.tick != 0 {
		ticks++
	}

	// 计算圈数和位置
	task.Round = ticks / tw.wheelSize
	pos := (tw.currentPos + ticks) % tw.wheelSize

	// 添加到槽位
	task.element = tw.slots[pos].PushBack(task)
	tw.taskMap[task.ID] = task

	mlog.Debug("task added to time wheel",
		zap.String("id", task.ID),
		zap.Duration("delay", task.Delay),
		zap.Int("pos", pos),
		zap.Int("round", task.Round),
	)
}

// cancelTaskInternal 内部取消任务
func (tw *TimeWheel) cancelTaskInternal(id string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if task, ok := tw.taskMap[id]; ok {
		task.Canceled = true
		delete(tw.taskMap, id)
	}
}

// executeTask 执行任务
func (tw *TimeWheel) executeTask(task *Task) (err error) {
	start := time.Now()

	// 创建上下文
	ctx := context.WithValue(tw.ctx, "request_id", fmt.Sprintf("tw_%s_%d", task.ID, start.Unix()))

	// 记录开始日志
	mlog.Debug("task executing",
		zap.String("id", task.ID),
		zap.Duration("delay", task.Delay),
	)

	// 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, string(debug.Stack()))
			mlog.Error("task panic",
				zap.String("id", task.ID),
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())),
			)
		}
	}()

	// 执行回调
	err = task.Callback(ctx)

	duration := time.Since(start)
	atomic.AddInt64(&tw.executedCount, 1)

	if err != nil {
		mlog.Error("task failed",
			zap.String("id", task.ID),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
	} else {
		mlog.Debug("task completed",
			zap.String("id", task.ID),
			zap.Duration("duration", duration),
		)
	}

	return err
}

// Delay 便捷函数：延迟执行
func Delay(tw *TimeWheel, id string, delay time.Duration, callback func(ctx context.Context) error) error {
	return tw.AddTask(id, delay, callback)
}

// After 便捷函数：类似于 time.AfterFunc
func After(delay time.Duration, callback func()) string {
	tw := defaultTimeWheel
	if tw == nil || atomic.LoadInt32(&tw.running) == 0 {
		// 如果默认时间轮未启动，使用标准库的 time.AfterFunc
		time.AfterFunc(delay, callback)
		return ""
	}

	id := fmt.Sprintf("after_%d", time.Now().UnixNano())
	tw.AddTask(id, delay, func(ctx context.Context) error {
		callback()
		return nil
	})
	return id
}

// defaultTimeWheel 默认时间轮实例
var defaultTimeWheel *TimeWheel
var once sync.Once

// InitDefault 初始化默认时间轮
func InitDefault(opts ...Option) {
	once.Do(func() {
		defaultTimeWheel = New(opts...)
		defaultTimeWheel.Start()
	})
}

// StopDefault 停止默认时间轮
func StopDefault() {
	if defaultTimeWheel != nil {
		defaultTimeWheel.Stop()
	}
}

// Default 获取默认时间轮
func Default() *TimeWheel {
	if defaultTimeWheel == nil {
		InitDefault()
	}
	return defaultTimeWheel
}

// ==================== 分层时间轮（Hierarchical Time Wheel）====================

// HierarchicalTimeWheel 分层时间轮
// 支持更大范围的延迟时间
type HierarchicalTimeWheel struct {
	// wheels 各级时间轮
	// wheels[0]: 毫秒级（tick = 1ms, size = 1000）
	// wheels[1]: 秒级（tick = 1s, size = 60）
	// wheels[2]: 分钟级（tick = 1m, size = 60）
	// wheels[3]: 小时级（tick = 1h, size = 24）
	wheels []*TimeWheel
	// ctx 上下文
	ctx context.Context
	// cancel 取消函数
	cancel context.CancelFunc
	// wg 等待组
	wg sync.WaitGroup
}

// NewHierarchical 创建分层时间轮
func NewHierarchical() *HierarchicalTimeWheel {
	ctx, cancel := context.WithCancel(context.Background())

	htw := &HierarchicalTimeWheel{
		wheels: make([]*TimeWheel, 4),
		ctx:    ctx,
		cancel: cancel,
	}

	// 创建各级时间轮
	htw.wheels[0] = New(WithTick(time.Millisecond), WithWheelSize(1000)) // 毫秒级：0-1s
	htw.wheels[1] = New(WithTick(time.Second), WithWheelSize(60))        // 秒级：0-1m
	htw.wheels[2] = New(WithTick(time.Minute), WithWheelSize(60))        // 分钟级：0-1h
	htw.wheels[3] = New(WithTick(time.Hour), WithWheelSize(24))          // 小时级：0-1d

	return htw
}

// Start 启动分层时间轮
func (htw *HierarchicalTimeWheel) Start() {
	for _, tw := range htw.wheels {
		tw.Start()
	}
	mlog.Info("hierarchical time wheel started")
}

// Stop 停止分层时间轮
func (htw *HierarchicalTimeWheel) Stop() {
	htw.cancel()
	for _, tw := range htw.wheels {
		tw.Stop()
	}
	mlog.Info("hierarchical time wheel stopped")
}

// AddTask 添加延迟任务
func (htw *HierarchicalTimeWheel) AddTask(id string, delay time.Duration, callback func(ctx context.Context) error) error {
	if delay < 0 {
		return fmt.Errorf("delay must be non-negative")
	}

	// 根据延迟时间选择合适的时间轮
	var tw *TimeWheel
	switch {
	case delay < time.Second:
		tw = htw.wheels[0] // 毫秒级
	case delay < time.Minute:
		tw = htw.wheels[1] // 秒级
	case delay < time.Hour:
		tw = htw.wheels[2] // 分钟级
	default:
		tw = htw.wheels[3] // 小时级
	}

	return tw.AddTask(id, delay, callback)
}

// CancelTask 取消任务
func (htw *HierarchicalTimeWheel) CancelTask(id string) bool {
	for _, tw := range htw.wheels {
		if tw.CancelTask(id) {
			return true
		}
	}
	return false
}
