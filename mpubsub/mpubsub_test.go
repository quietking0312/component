package mpubsub

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

type Context[T any] struct {
	Id string
}

func NewContext[T any](Id string) WriteIface[T] {
	return &Context[T]{
		Id: Id,
	}
}
func (c *Context[T]) Write(t T) error {
	fmt.Println(fmt.Sprintf("频道%s 收到消息： %v", c.Id, t))
	return nil
}
func (c *Context[T]) Close() error {
	return nil
}
func (c *Context[T]) ID() string {
	return c.Id
}

func TestNewMPubSub(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	subFunc := func(ctx context.Context, key string) (<-chan []byte, error) {
		msgChan := make(chan []byte)
		go func() {
			defer close(msgChan)
			sub := rdb.Subscribe(context.Background(), key)
			defer sub.Close()
			ch := sub.Channel()
			for {
				select {
				case <-ctx.Done():
				case msg, ok := <-ch:
					if !ok {
						return
					}
					msgChan <- []byte(msg.Payload)
				}
			}
		}()
		return msgChan, nil
	}
	pubFunc := func(ctx context.Context, key string, m []byte) error {
		return rdb.Publish(ctx, key, m).Err()
	}
	subChannel := NewSubChannel[string, any]("1")
	subChannel.Register("100", NewContext[any]("100"))
	sub, err := NewMPubSub[any]([]string{"test_01"}, subFunc, pubFunc)
	if err != nil {
		t.Fatal(err)
	}
	sub.Start()
	sub.Register("1", subChannel)
	for i := 0; i < 200; i++ {
		sub.Publish(Message[any]{
			ChannelId: "1",
			Data:      "hello world",
		})
		time.Sleep(time.Second)
		if i > 50 {
			subChannel.Register("101", NewContext[any]("101"))
		}
		if i > 100 {
			subChannel.Register("102", NewContext[any]("102"))
		}
	}
	time.Sleep(5 * time.Minute)
}
