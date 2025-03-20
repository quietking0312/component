package mchat

import (
	"context"
	"fmt"
	"martial/component/mredis"
	"time"
)

const (
	channel = "chat_test_1"
)

type Chat struct {
	cli mredis.Client
}

func NewChat() (*Chat, error) {
	cli, err := mredis.NewClient(mredis.SetAddrs([]string{"127.0.0.1:6379"}), mredis.SetPoolSize(20))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &Chat{
		cli: cli,
	}, nil
}

func (chat *Chat) Sub(x int) {
	sub := chat.cli.Subscribe(context.Background(), channel, "chat_test_2")
	defer sub.Close()
	for msg := range sub.Channel() {
		fmt.Println(x, msg.Payload)
	}
}

func (chat *Chat) Publish() {
	i := 0
	for {
		i += 1
		err := chat.cli.Publish(context.Background(), channel, fmt.Sprintf("hello %d", i)).Err()
		if err != nil {
			fmt.Println(err)
			return
		}
		err = chat.cli.Publish(context.Background(), "chat_test_2", fmt.Sprintf("world %d", i)).Err()
		if err != nil {
			fmt.Println(err)
			return
		}
		time.Sleep(time.Second * 3)
	}
}
