package mtimewheel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tw := New()
	assert.NotNil(t, tw)
	assert.Equal(t, 100*time.Millisecond, tw.tick)
	assert.Equal(t, 100, tw.wheelSize)
	assert.NotNil(t, tw.slots)
}

func TestNewWithOptions(t *testing.T) {
	tw := New(WithTick(50*time.Millisecond), WithWheelSize(200))
	assert.Equal(t, 50*time.Millisecond, tw.tick)
	assert.Equal(t, 200, tw.wheelSize)
}

func TestTimeWheel_AddTask(t *testing.T) {
	tw := New(WithTick(10 * time.Millisecond))
	tw.Start()
	defer tw.Stop()

	var executed bool
	err := tw.AddTask("test-task", 200*time.Millisecond, func(ctx context.Context) error {
		executed = true
		return nil
	})

	assert.NoError(t, err)

	// 等待任务执行（给足够的时间）
	time.Sleep(400 * time.Millisecond)
	assert.True(t, executed)
}

func TestTimeWheel_AddTaskMultiple(t *testing.T) {
	tw := New(WithTick(10 * time.Millisecond))
	tw.Start()
	defer tw.Stop()

	var counter int32
	var wg sync.WaitGroup

	// 添加多个任务
	for i := 0; i < 10; i++ {
		wg.Add(1)
		id := fmt.Sprintf("task-%d", i)
		tw.AddTask(id, 100*time.Millisecond, func(ctx context.Context) error {
			defer wg.Done()
			atomic.AddInt32(&counter, 1)
			return nil
		})
	}

	// 等待所有任务执行
	wg.Wait()
	assert.Equal(t, int32(10), atomic.LoadInt32(&counter))
}

func TestTimeWheel_CancelTask(t *testing.T) {
	tw := New(WithTick(10 * time.Millisecond))
	tw.Start()
	defer tw.Stop()

	tw.AddTask("cancelable-task", time.Second, func(ctx context.Context) error {
		return nil
	})

	// 等待异步添加完成
	time.Sleep(50 * time.Millisecond)

	// 取消任务
	cancelled := tw.CancelTask("cancelable-task")
	assert.True(t, cancelled)

	// 再次取消应该失败
	cancelled = tw.CancelTask("cancelable-task")
	assert.False(t, cancelled)
}

func TestTimeWheel_GetTask(t *testing.T) {
	tw := New()
	tw.Start()
	defer tw.Stop()

	tw.AddTask("gettable-task", time.Hour, func(ctx context.Context) error {
		return nil
	})

	// 等待异步添加完成
	time.Sleep(50 * time.Millisecond)

	task := tw.GetTask("gettable-task")
	assert.NotNil(t, task)
	assert.Equal(t, "gettable-task", task.ID)
	assert.Equal(t, time.Hour, task.Delay)
}

func TestTimeWheel_TaskCount(t *testing.T) {
	tw := New()
	tw.Start()
	defer tw.Stop()

	initialCount := tw.TaskCount()

	tw.AddTask("task-1", time.Hour, func(ctx context.Context) error { return nil })
	tw.AddTask("task-2", time.Hour, func(ctx context.Context) error { return nil })

	assert.Equal(t, int64(2), tw.TaskCount()-initialCount)
}

func TestTimeWheel_TaskPanic(t *testing.T) {
	tw := New(WithTick(10 * time.Millisecond))
	tw.Start()
	defer tw.Stop()

	tw.AddTask("panic-task", 100*time.Millisecond, func(ctx context.Context) error {
		panic("intentional panic")
	})

	// 等待任务执行（会被 panic，但应该被捕获）
	time.Sleep(200 * time.Millisecond)

	// 通过
	assert.True(t, true)
}

func TestTimeWheel_TaskError(t *testing.T) {
	tw := New(WithTick(10 * time.Millisecond))
	tw.Start()
	defer tw.Stop()

	tw.AddTask("error-task", 100*time.Millisecond, func(ctx context.Context) error {
		return errors.New("test error")
	})

	// 等待任务执行
	time.Sleep(200 * time.Millisecond)

	// 通过
	assert.True(t, true)
}

func TestTimeWheel_InvalidDelay(t *testing.T) {
	tw := New()
	tw.Start()
	defer tw.Stop()

	err := tw.AddTask("invalid-task", -1*time.Second, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delay must be non-negative")
}

func TestTimeWheel_NilCallback(t *testing.T) {
	tw := New()
	tw.Start()
	defer tw.Stop()

	err := tw.AddTask("nil-callback-task", time.Second, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "callback cannot be nil")
}

func TestTimeWheel_NotRunning(t *testing.T) {
	tw := New()
	// 不启动时间轮

	err := tw.AddTask("task", time.Second, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time wheel is not running")
}

func TestTimeWheel_ZeroDelay(t *testing.T) {
	tw := New()
	// 不启动时间轮

	var executed bool
	err := tw.AddTask("zero-delay-task", 0, func(ctx context.Context) error {
		executed = true
		return nil
	})

	// 延迟为0时，应该直接执行
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestDelay(t *testing.T) {
	tw := New(WithTick(10 * time.Millisecond))
	tw.Start()
	defer tw.Stop()

	done := make(chan bool)
	err := Delay(tw, "delay-task", 100*time.Millisecond, func(ctx context.Context) error {
		close(done)
		return nil
	})

	assert.NoError(t, err)

	select {
	case <-done:
		// 成功
	case <-time.After(300 * time.Millisecond):
		t.Fatal("task did not execute")
	}
}

func TestDefault(t *testing.T) {
	// 确保默认时间轮未初始化
	defaultTimeWheel = nil

	tw := Default()
	assert.NotNil(t, tw)
	assert.NotNil(t, defaultTimeWheel)

	StopDefault()
}

func TestAfter(t *testing.T) {
	// 先初始化默认时间轮
	defaultTimeWheel = nil
	InitDefault(WithTick(10 * time.Millisecond))
	defer StopDefault()

	done := make(chan bool)
	id := After(100*time.Millisecond, func() {
		close(done)
	})

	// id 可能为空（如果直接使用 time.AfterFunc）
	_ = id

	select {
	case <-done:
		// 成功
	case <-time.After(300 * time.Millisecond):
		t.Fatal("task did not execute")
	}
}

func TestAfterWithoutInit(t *testing.T) {
	// 确保默认时间轮未初始化
	StopDefault()
	defaultTimeWheel = nil

	var executed bool
	id := After(100*time.Millisecond, func() {
		executed = true
	})

	// 返回空ID，但任务仍然执行
	assert.Empty(t, id)

	time.Sleep(200 * time.Millisecond)
	assert.True(t, executed)
}

func TestHierarchicalTimeWheel(t *testing.T) {
	htw := NewHierarchical()
	htw.Start()
	defer htw.Stop()

	var counter int32

	// 添加不同延迟的任务
	htw.AddTask("ms-task", 100*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	htw.AddTask("sec-task", 2*time.Second, func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	// 等待毫秒级任务执行
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&counter))

	// 取消秒级任务
	cancelled := htw.CancelTask("sec-task")
	assert.True(t, cancelled)
}

func TestHierarchicalTimeWheel_CancelNotFound(t *testing.T) {
	htw := NewHierarchical()
	htw.Start()
	defer htw.Stop()

	cancelled := htw.CancelTask("non-existent-task")
	assert.False(t, cancelled)
}

func TestHierarchicalTimeWheel_InvalidDelay(t *testing.T) {
	htw := NewHierarchical()
	htw.Start()
	defer htw.Stop()

	err := htw.AddTask("invalid-task", -1*time.Second, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
}

func BenchmarkTimeWheel_AddTask(b *testing.B) {
	tw := New()
	tw.Start()
	defer tw.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.AddTask(fmt.Sprintf("task-%d", i), time.Hour, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkTimeWheel_Execute(b *testing.B) {
	tw := New(WithTick(10 * time.Millisecond))
	tw.Start()
	defer tw.Stop()

	var counter int32

	// 预添加任务
	for i := 0; i < 1000; i++ {
		tw.AddTask(fmt.Sprintf("task-%d", i), 50*time.Millisecond, func(ctx context.Context) error {
			atomic.AddInt32(&counter, 1)
			return nil
		})
	}

	b.ResetTimer()
	start := time.Now()

	// 等待所有任务执行
	for atomic.LoadInt32(&counter) < int32(b.N) {
		time.Sleep(10 * time.Millisecond)
		if time.Since(start) > 30*time.Second {
			break
		}
	}
}
