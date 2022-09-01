package mredis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"sync"
	"testing"
	"time"
)

func TestNewRedisClient(t *testing.T) {
	_client, err := NewRedisClient()
	if err != nil {
		t.Fatal(err)
	}
	val, err := _client.Do(context.Background(), "keys", "*").StringSlice()
	if err != nil {
		if err == redis.Nil {
			fmt.Println("key ")
			return
		}
	}
	fmt.Println(val)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		err1 := _client.Watch(context.Background(), func(tx *redis.Tx) error {
			cmds, err := tx.TxPipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
				pipeResult := pipeliner.Set(context.Background(), "key", "11", 10*time.Second).Err()

				fmt.Println(pipeResult)
				return nil
			})
			if err != nil {
				fmt.Println("pipe err:", err.Error())
				return err
			}
			for _, cmd := range cmds {
				fmt.Println(cmd.String())
			}
			return nil
		}, "key")
		if err1 != nil {
			fmt.Println("err1", err1)
			t.Fatal(err1)
		}
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()
		err2 := _client.Watch(context.Background(), func(tx *redis.Tx) error {
			cmds, err := tx.TxPipelined(context.Background(), func(pipeliner redis.Pipeliner) error {
				pipeResult := pipeliner.Set(context.Background(), "key", "22", 10*time.Second).Err()
				fmt.Println(pipeResult)
				return nil
			})
			if err != nil {
				fmt.Println("pipe err:", err.Error())
				return err
			}
			for _, cmd := range cmds {
				fmt.Println(cmd.String())
			}
			return nil
		}, "key")
		if err2 != nil {
			fmt.Println("err2", err2)
		}
	}()
	wg.Wait()
	fmt.Println(_client.Get(context.Background(), "key").String())
}
