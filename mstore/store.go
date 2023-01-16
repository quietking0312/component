package mstore

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Store struct {
	data   sync.Map
	filter map[string]func(key string)
	mu     sync.Mutex
}

func (s *Store) Get(Key string) (interface{}, error) {
	if v, ok := s.data.Load(Key); ok {
		return v, nil
	}
	return nil, fmt.Errorf("not found")
}

func (s *Store) Set(k string, v interface{}) error {
	oldValue, _ := s.data.Load(k)
	if oldValue != v {
		for _, fn := range s.filter {
			fn(k)
		}
	}
	s.data.Store(k, v)
	return nil
}

func (s *Store) Watch(ctx context.Context, key *string) {
	if s.filter == nil {
		s.filter = make(map[string]func(key string))
	}
	id := fmt.Sprintf("watch-%s", time.Now())
	ch := make(chan string, 10)
	s.mu.Lock()
	s.filter[id] = func(key string) {
		ch <- key
	}
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		return
	case k := <-ch:
		*key = k
		return
	}
}
