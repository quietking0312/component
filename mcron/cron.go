// Package mcron 提供定时任务执行功能
// 支持 cron 表达式、固定间隔执行、延迟执行等多种模式
package mcron

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/quietking0312/component/mlog"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Job 任务接口
type Job interface {
	// Run 执行任务
	Run(ctx context.Context) error
	// Name 返回任务名称
	Name() string
}

// JobFunc 任务函数类型
type JobFunc func(ctx context.Context) error

// FuncJob 函数任务适配器
type FuncJob struct {
	name string
	fn   JobFunc
}

// Run 执行任务
func (j *FuncJob) Run(ctx context.Context) error {
	return j.fn(ctx)
}

// Name 返回任务名称
func (j *FuncJob) Name() string {
	return j.name
}

// NewFuncJob 创建函数任务
func NewFuncJob(name string, fn JobFunc) Job {
	return &FuncJob{name: name, fn: fn}
}

// Schedule 任务调度信息
type Schedule struct {
	// Cron 表达式（可选，与 Interval 二选一）
	// 格式：秒 分 时 日 月 周
	// 示例："0 0 9 * * 1" 每周一上午9点
	Cron string

	// Interval 固定间隔（可选，与 Cron 二选一）
	// 示例：time.Hour 每小时执行一次
	Interval time.Duration

	// Delay 首次执行的延迟时间（可选）
	Delay time.Duration

	// At 固定时间点执行（可选，格式 "15:04:05"）
	At string
}

// Entry 任务条目
type Entry struct {
	ID         cron.EntryID
	Name       string
	Schedule   Schedule
	Job        Job
	NextRun    time.Time
	PrevRun    time.Time
	RunCount   int64
	ErrorCount int64
	LastError  error
}

// Cron 定时任务调度器
type Cron struct {
	cron     *cron.Cron
	jobs     map[cron.EntryID]*Entry
	mu       sync.RWMutex
	location *time.Location
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Options 配置选项
type Options struct {
	// Location 时区，默认本地时区
	Location *time.Location
	// Logger 日志记录器
	Logger *zap.Logger
}

// Option 配置函数
type Option func(*Options)

// WithLocation 设置时区
func WithLocation(loc *time.Location) Option {
	return func(o *Options) {
		o.Location = loc
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger *zap.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// New 创建定时任务调度器
func New(opts ...Option) *Cron {
	options := &Options{
		Location: time.Local,
		Logger:   mlog.Logger(),
	}

	for _, opt := range opts {
		opt(options)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Cron{
		cron: cron.New(
			cron.WithLocation(options.Location),
			cron.WithSeconds(),
			cron.WithLogger(&cronLogger{logger: options.Logger}),
		),
		jobs:     make(map[cron.EntryID]*Entry),
		location: options.Location,
		ctx:      ctx,
		cancel:   cancel,
	}

	return c
}

// Start 启动调度器
func (c *Cron) Start() {
	c.cron.Start()
	mlog.Info("cron scheduler started")
}

// Stop 停止调度器
func (c *Cron) Stop() {
	c.cancel()
	c.cron.Stop()
	c.wg.Wait()
	mlog.Info("cron scheduler stopped")
}

// AddJob 添加任务
func (c *Cron) AddJob(name string, schedule Schedule, job Job) (cron.EntryID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 创建包装任务
	wrapper := &jobWrapper{
		cron: c,
		entry: &Entry{
			Name:     name,
			Schedule: schedule,
			Job:      job,
		},
	}

	// 解析调度规则
	var err error
	var id cron.EntryID

	switch {
	case schedule.Cron != "":
		// Cron 表达式
		id, err = c.cron.AddJob(schedule.Cron, wrapper)
	case schedule.Interval > 0:
		// 固定间隔
		id, err = c.cron.AddJob(fmt.Sprintf("@every %s", schedule.Interval), wrapper)
	case schedule.At != "":
		// 固定时间点
		id, err = c.cron.AddJob(schedule.At, wrapper)
	default:
		return 0, fmt.Errorf("invalid schedule: must specify Cron, Interval or At")
	}

	if err != nil {
		return 0, fmt.Errorf("add job failed: %w", err)
	}

	wrapper.entry.ID = id
	c.jobs[id] = wrapper.entry

	// 如果有延迟，设置下次执行时间
	if schedule.Delay > 0 {
		wrapper.entry.NextRun = time.Now().Add(schedule.Delay)
	}

	mlog.Info("job added",
		zap.String("name", name),
		zap.String("cron", schedule.Cron),
		zap.Duration("interval", schedule.Interval),
	)

	return id, nil
}

// AddFunc 添加函数任务
func (c *Cron) AddFunc(name string, schedule Schedule, fn JobFunc) (cron.EntryID, error) {
	return c.AddJob(name, schedule, NewFuncJob(name, fn))
}

// Remove 移除任务
func (c *Cron) Remove(id cron.EntryID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.jobs[id]; ok {
		c.cron.Remove(id)
		delete(c.jobs, id)
		mlog.Info("job removed", zap.String("name", entry.Name))
	}
}

// GetEntry 获取任务条目
func (c *Cron) GetEntry(id cron.EntryID) *Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jobs[id]
}

// ListEntries 列出所有任务
func (c *Cron) ListEntries() []*Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]*Entry, 0, len(c.jobs))
	for _, entry := range c.jobs {
		entries = append(entries, entry)
	}
	return entries
}

// RunOnce 立即执行一次任务（阻塞）
func (c *Cron) RunOnce(id cron.EntryID) error {
	c.mu.RLock()
	entry := c.jobs[id]
	c.mu.RUnlock()

	if entry == nil {
		return fmt.Errorf("job not found: %d", id)
	}

	return c.executeJob(entry)
}

// RunOnceAsync 立即异步执行一次任务
func (c *Cron) RunOnceAsync(id cron.EntryID) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.RunOnce(id); err != nil {
			mlog.Error("run once async failed", zap.Error(err))
		}
	}()
}

// executeJob 执行具体任务
func (c *Cron) executeJob(entry *Entry) (err error) {
	start := time.Now()
	entry.PrevRun = start
	entry.RunCount++

	// 创建任务上下文
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Minute)
	defer cancel()

	// 添加 request_id 便于追踪
	ctx = context.WithValue(ctx, "request_id", fmt.Sprintf("cron_%s_%d", entry.Name, start.Unix()))

	// 记录开始日志
	mlog.Info("job started",
		zap.String("name", entry.Name),
		zap.Int64("run_count", entry.RunCount),
	)

	// 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, string(debug.Stack()))
			entry.ErrorCount++
			entry.LastError = err
			mlog.Error("job panic",
				zap.String("name", entry.Name),
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())),
			)
		}
	}()

	// 执行任务
	err = entry.Job.Run(ctx)

	duration := time.Since(start)

	if err != nil {
		entry.ErrorCount++
		entry.LastError = err
		mlog.Error("job failed",
			zap.String("name", entry.Name),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
	} else {
		mlog.Info("job completed",
			zap.String("name", entry.Name),
			zap.Duration("duration", duration),
		)
	}

	return err
}

// jobWrapper 任务包装器
type jobWrapper struct {
	cron  *Cron
	entry *Entry
}

// Run 实现 cron.Job 接口
func (w *jobWrapper) Run() {
	w.cron.executeJob(w.entry)
}

// cronLogger 适配 cron 的日志接口
type cronLogger struct {
	logger *zap.Logger
}

func (l *cronLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Sugar().Infow(msg, keysAndValues...)
}

func (l *cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fields := []interface{}{"error", err}
	fields = append(fields, keysAndValues...)
	l.logger.Sugar().Errorw(msg, fields...)
}

// ==================== 便捷函数 ====================

// Every 创建固定间隔调度
func Every(interval time.Duration) Schedule {
	return Schedule{Interval: interval}
}

// At 创建固定时间点调度
func At(timeStr string) Schedule {
	return Schedule{At: timeStr}
}

// CronExpr 创建 cron 表达式调度
func CronExpr(expr string) Schedule {
	return Schedule{Cron: expr}
}

// Delay 创建带延迟的调度
func Delay(delay time.Duration, schedule Schedule) Schedule {
	schedule.Delay = delay
	return schedule
}

// ==================== 常用 Cron 表达式 ====================

// Predefined 预定义的常用调度
var Predefined = struct {
	// EverySecond 每秒
	EverySecond Schedule
	// EveryMinute 每分钟
	EveryMinute Schedule
	// EveryHour 每小时
	EveryHour Schedule
	// EveryDay 每天零点
	EveryDay Schedule
	// EveryWeek 每周一零点
	EveryWeek Schedule
	// EveryMonth 每月1号零点
	EveryMonth Schedule
}{
	EverySecond: Schedule{Cron: "* * * * * *"},
	EveryMinute: Schedule{Cron: "0 * * * * *"},
	EveryHour:   Schedule{Cron: "0 0 * * * *"},
	EveryDay:    Schedule{Cron: "0 0 0 * * *"},
	EveryWeek:   Schedule{Cron: "0 0 0 * * 1"},
	EveryMonth:  Schedule{Cron: "0 0 0 1 * *"},
}
