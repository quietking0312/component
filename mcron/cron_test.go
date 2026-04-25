package mcron

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	cron := New()
	assert.NotNil(t, cron)
	assert.NotNil(t, cron.cron)
	assert.NotNil(t, cron.jobs)
}

func TestCron_AddFunc(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	var executed bool
	id, err := cron.AddFunc("test-job", Every(time.Second), func(ctx context.Context) error {
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.NotZero(t, id)

	// 等待任务执行
	time.Sleep(1100 * time.Millisecond)
	assert.True(t, executed)
}

func TestCron_AddJob(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	job := NewFuncJob("test-job", func(ctx context.Context) error {
		return nil
	})

	id, err := cron.AddJob("test-job", Every(100*time.Millisecond), job)
	assert.NoError(t, err)
	assert.NotZero(t, id)

	entry := cron.GetEntry(id)
	assert.NotNil(t, entry)
	assert.Equal(t, "test-job", entry.Name)
}

func TestCron_CronExpression(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	var counter int32
	// 每秒执行一次
	id, err := cron.AddFunc("cron-job", CronExpr("* * * * * *"), func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	assert.NoError(t, err)
	assert.NotZero(t, id)

	// 等待 2.5 秒
	time.Sleep(2500 * time.Millisecond)

	count := atomic.LoadInt32(&counter)
	assert.True(t, count >= 2 && count <= 4, "expected 2-4 executions, got %d", count)
}

func TestCron_Remove(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	done := make(chan bool, 1)
	id, _ := cron.AddFunc("removable-job", Every(100*time.Millisecond), func(ctx context.Context) error {
		select {
		case done <- true:
		default:
		}
		return nil
	})

	// 等待执行一次
	<-done

	// 移除任务
	cron.Remove(id)

	// 再等待一段时间
	time.Sleep(300 * time.Millisecond)

	// 应该不会再收到信号
	select {
	case <-done:
		t.Fatal("job should have been removed")
	case <-time.After(100 * time.Millisecond):
		// 成功
	}
}

func TestCron_RunOnce(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	var executed bool
	id, _ := cron.AddFunc("once-job", Every(time.Hour), func(ctx context.Context) error {
		executed = true
		return nil
	})

	// 立即执行一次
	err := cron.RunOnce(id)
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestCron_JobPanic(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	id, _ := cron.AddFunc("panic-job", Every(time.Second), func(ctx context.Context) error {
		panic("intentional panic")
	})

	// 给 panic 恢复一些时间
	time.Sleep(1200 * time.Millisecond)

	// 即使 panic，任务条目仍然存在
	entry := cron.GetEntry(id)
	assert.NotNil(t, entry)
}

func TestCron_JobError(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	id, _ := cron.AddFunc("error-job", Every(time.Second), func(ctx context.Context) error {
		return errors.New("test error")
	})

	// 等待执行
	time.Sleep(1100 * time.Millisecond)

	entry := cron.GetEntry(id)
	assert.NotNil(t, entry)
}

func TestCron_ListEntries(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	cron.AddFunc("job-1", Every(time.Hour), func(ctx context.Context) error { return nil })
	cron.AddFunc("job-2", Every(time.Hour), func(ctx context.Context) error { return nil })
	cron.AddFunc("job-3", Every(time.Hour), func(ctx context.Context) error { return nil })

	entries := cron.ListEntries()
	assert.Len(t, entries, 3)
}

func TestScheduleHelpers(t *testing.T) {
	assert.Equal(t, time.Hour, Every(time.Hour).Interval)
	assert.Equal(t, "0 9 * * *", At("0 9 * * *").At)
	assert.Equal(t, "0 0 * * * *", CronExpr("0 0 * * * *").Cron)

	delayed := Delay(5*time.Minute, Every(time.Hour))
	assert.Equal(t, 5*time.Minute, delayed.Delay)
	assert.Equal(t, time.Hour, delayed.Interval)
}

func TestPredefinedSchedules(t *testing.T) {
	assert.Equal(t, "* * * * * *", Predefined.EverySecond.Cron)
	assert.Equal(t, "0 * * * * *", Predefined.EveryMinute.Cron)
	assert.Equal(t, "0 0 * * * *", Predefined.EveryHour.Cron)
	assert.Equal(t, "0 0 0 * * *", Predefined.EveryDay.Cron)
	assert.Equal(t, "0 0 0 * * 1", Predefined.EveryWeek.Cron)
	assert.Equal(t, "0 0 0 1 * *", Predefined.EveryMonth.Cron)
}

func TestCron_InvalidSchedule(t *testing.T) {
	cron := New()

	_, err := cron.AddFunc("invalid-job", Schedule{}, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schedule")
}

func TestCron_NotRunning(t *testing.T) {
	cron := New()
	// 不启动调度器

	_, err := cron.AddFunc("job", Every(time.Hour), func(ctx context.Context) error {
		return nil
	})

	// 应该可以添加任务，只是不会执行
	assert.NoError(t, err)
}

func TestFuncJob(t *testing.T) {
	var executed bool
	job := NewFuncJob("test", func(ctx context.Context) error {
		executed = true
		return nil
	})

	assert.Equal(t, "test", job.Name())
	err := job.Run(context.Background())
	assert.NoError(t, err)
	assert.True(t, executed)
}
