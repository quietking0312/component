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

func NewContext[T any](Id string) HandlerIface[T] {
	return &Context[T]{
		Id: Id,
	}
}
func (c *Context[T]) Write(t T) error {
	fmt.Println(fmt.Sprintf("频道%s 收到消息： %v", c.Id, t))
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
	g := &Group[any]{}
	subChannel := NewSubGroup[any]("1", g)
	g.Set(NewContext[any]("100"))
	sub, err := NewMPubSub[any]([]string{"test_01"}, subFunc, pubFunc)
	if err != nil {
		t.Fatal(err)
	}
	sub.Start()
	sub.Register(subChannel)
	for i := 0; i < 200; i++ {
		sub.Publish(Message[any]{
			GroupId: "1",
			Data:    "hello world",
		})
		time.Sleep(time.Second)
		if i > 50 {
			g.Set(NewContext[any]("101"))
		}
		if i > 100 {
			g.Set(NewContext[any]("102"))
		}
	}
	time.Sleep(5 * time.Minute)
}
