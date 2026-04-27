package mevent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	b := New()
	require.NotNil(t, b)
	assert.Empty(t, b.Subscribers())
}

func TestSubscribeAndPublish(t *testing.T) {
	b := New()
	var called int32

	b.Subscribe("order.created", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		assert.Equal(t, "order.created", e.Name)
		assert.Equal(t, 42, e.Data)
		return nil
	})

	err := b.Publish(context.Background(), Event{Name: "order.created", Data: 42})
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestPublish_NoSubscribers(t *testing.T) {
	b := New()
	err := b.Publish(context.Background(), Event{Name: "nothing", Data: nil})
	require.NoError(t, err)
}

func TestPublish_MultipleHandlers(t *testing.T) {
	b := New()
	var sum int32

	b.Subscribe("count", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&sum, 1)
		return nil
	})
	b.Subscribe("count", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&sum, 2)
		return nil
	})

	err := b.Publish(context.Background(), Event{Name: "count"})
	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&sum))
}

func TestPublish_StopOnError(t *testing.T) {
	b := New()
	var secondCalled int32

	b.Subscribe("fail", func(ctx context.Context, e Event) error {
		return errors.New("first failed")
	})
	b.Subscribe("fail", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&secondCalled, 1)
		return nil
	})

	err := b.Publish(context.Background(), Event{Name: "fail"})
	require.Error(t, err)
	assert.Equal(t, "first failed", err.Error())
	assert.Equal(t, int32(0), atomic.LoadInt32(&secondCalled))
}

func TestPublish_ContinueOnError(t *testing.T) {
	b := New(WithErrorStrategy(ContinueOnError))
	var secondCalled int32

	b.Subscribe("fail", func(ctx context.Context, e Event) error {
		return errors.New("first failed")
	})
	b.Subscribe("fail", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&secondCalled, 1)
		return nil
	})

	err := b.Publish(context.Background(), Event{Name: "fail"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1/2 handlers failed")
	assert.Equal(t, int32(1), atomic.LoadInt32(&secondCalled))
}

func TestPublishAsync(t *testing.T) {
	b := New()
	var called int32

	b.Subscribe("async", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	b.PublishAsync(context.Background(), Event{Name: "async"})
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestHasSubscribers(t *testing.T) {
	b := New()
	assert.False(t, b.HasSubscribers("x"))
	b.Subscribe("x", func(ctx context.Context, e Event) error { return nil })
	assert.True(t, b.HasSubscribers("x"))
}

func TestMiddleware_Recovery(t *testing.T) {
	b := New()
	b.Use(Recovery(nil))
	var called int32

	b.Subscribe("panic", func(ctx context.Context, e Event) error {
		panic("boom")
	})
	b.Subscribe("panic", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	err := b.Publish(context.Background(), Event{Name: "panic"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
	// second handler should NOT be called because first panicked and StopOnError is default
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestMiddleware_Filter(t *testing.T) {
	b := New()
	b.Use(Filter(func(e Event) bool {
		return e.Data != nil
	}))
	var called int32

	b.Subscribe("filtered", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	err := b.Publish(context.Background(), Event{Name: "filtered", Data: nil})
	require.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))

	err = b.Publish(context.Background(), Event{Name: "filtered", Data: "ok"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestMiddleware_Order(t *testing.T) {
	b := New()
	var order []string

	b.Use(func(next Handler) Handler {
		return func(ctx context.Context, e Event) error {
			order = append(order, "mw1-before")
			err := next(ctx, e)
			order = append(order, "mw1-after")
			return err
		}
	})
	b.Use(func(next Handler) Handler {
		return func(ctx context.Context, e Event) error {
			order = append(order, "mw2-before")
			err := next(ctx, e)
			order = append(order, "mw2-after")
			return err
		}
	})

	b.Subscribe("order", func(ctx context.Context, e Event) error {
		order = append(order, "handler")
		return nil
	})

	err := b.Publish(context.Background(), Event{Name: "order"})
	require.NoError(t, err)
	assert.Equal(t, []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}, order)
}
