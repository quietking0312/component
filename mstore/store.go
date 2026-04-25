package mstore

import (
	"fmt"
	"reflect"
	"sync"
)

type Store struct {
	data   sync.Map
	filter map[string][]WatchFunc
	mu     sync.Mutex
}

type WatchFunc func(oldV, newV any)

func NewStore() *Store {
	return &Store{
		filter: make(map[string][]WatchFunc),
	}
}

func (s *Store) Get(key string) (interface{}, error) {
	if v, ok := s.data.Load(key); ok {
		return v, nil
	}
	return nil, fmt.Errorf("not found")
}

func (s *Store) Set(k string, v interface{}) error {
	oldValue, _ := s.data.Load(k)
	s.data.Store(k, v)
	if !reflect.DeepEqual(oldValue, v) {
		s.mu.Lock()
		fcs := s.filter[k]
		s.mu.Unlock()
		for _, fc := range fcs {
			go func(fn WatchFunc) {
				defer func() {
					if r := recover(); r != nil {
						// 捕获 watch 回调中的 panic，避免崩溃
					}
				}()
				fn(oldValue, v)
			}(fc)
		}
	}
	return nil
}

func (s *Store) Register(key string, fc WatchFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.filter == nil {
		s.filter = make(map[string][]WatchFunc)
	}
	s.filter[key] = append(s.filter[key], fc)
}
