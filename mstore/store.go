package mstore

import (
	"fmt"
	"sync"
)

type Store struct {
	data   sync.Map
	filter map[string][]WatchFunc
	mu     sync.Mutex
}
type WatchFunc func(oldV, newV any)

func (s *Store) Get(Key string) (interface{}, error) {
	if v, ok := s.data.Load(Key); ok {
		return v, nil
	}
	return nil, fmt.Errorf("not found")
}

func (s *Store) Set(k string, v interface{}) error {
	oldValue, _ := s.data.Load(k)
	s.data.Store(k, v)
	if oldValue != v {
		fcs := s.filter[k]
		for _, fc := range fcs {
			go fc(oldValue, v)
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
	_, ok := s.filter[key]
	if !ok {
		s.filter[key] = make([]WatchFunc, 0)
	}
	s.filter[key] = append(s.filter[key], fc)

}
