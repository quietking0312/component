package mtool

import (
	"sync"
)

type FIFO[T any] struct {
	data  []T
	size  int
	head  int //
	tail  int
	count int
	mu    sync.RWMutex
}

func NewFIFO[T any](capacity int) *FIFO[T] {
	if capacity <= 0 {
		capacity = 10
	}
	return &FIFO[T]{
		data: make([]T, capacity),
		size: capacity,
		tail: -1,
	}
}

func (f *FIFO[T]) Push(value T) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.tail = (f.tail + 1) % f.size
	f.data[f.tail] = value

	if f.count < f.size {
		f.count++
	} else {
		f.head = (f.head + 1) % f.size
	}
}

func (f *FIFO[T]) Pop() (T, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var zero T
	if f.count == 0 {
		return zero, false
	}

	value := f.data[f.head]
	f.head = (f.head + 1) % f.size
	f.count--

	if f.count == 0 {
		f.head = 0
		f.tail = -1
	}

	return value, true
}

func (f *FIFO[T]) Data() []T {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make([]T, f.count)
	for i := 0; i < f.count; i++ {
		idx := (f.head + i) % f.size
		result[i] = f.data[idx]
	}
	return result
}

func (f *FIFO[T]) Len() int {
	return f.count
}
