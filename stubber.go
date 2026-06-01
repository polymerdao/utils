package utils

import (
	"context"
	"sync"
	"time"
)

// Stubber is a generic interface for a stubber that can hold items of type T and their associated errors.
type Stubber[T any] interface {
	// Append adds an item to the stubber
	Append(item T, err error)

	// AppendMany adds the item times times to the stubber
	AppendMany(item T, err error, times int)

	// Shift removes the first item from the stubber and returns it
	Shift() (item T, err error)

	// Wait waits for an item to be present in the stubber and returns it
	Wait(timeout time.Duration) (t T, found bool)

	// Reset clears the stubber
	Reset()

	// Length returns the number of items in the stubber
	Length() int
}

type stubber[T any] struct {
	lock  sync.Mutex
	items []T
	errs  []error
}

// NewStubber creates a new instance of Stubber for type T.
func NewStubber[T any]() Stubber[T] {
	return &stubber[T]{
		items: make([]T, 0),
		errs:  make([]error, 0),
	}
}

func (s *stubber[T]) Append(item T, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.append(item, err)
}

func (s *stubber[T]) AppendMany(item T, err error, times int) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for range times {
		s.append(item, err)
	}
}

func (s *stubber[T]) Shift() (T, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	t, _, err := s.shift()
	return t, err
}

func (s *stubber[T]) Wait(timeout time.Duration) (t T, found bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return t, false
		case <-ticker.C:
			s.lock.Lock()
			x, ok, _ := s.shift()
			s.lock.Unlock()
			if ok {
				return x, true
			}
		}
	}
}

func (s *stubber[T]) Reset() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items = make([]T, 0)
	s.errs = make([]error, 0)
}

func (s *stubber[T]) Length() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.items)
}

func (s *stubber[T]) append(item T, err error) {
	s.items = append(s.items, item)
	s.errs = append(s.errs, err)
}

func (s *stubber[T]) shift() (item T, found bool, err error) {
	if len(s.items) > 0 {
		item, s.items = s.items[0], s.items[1:]
		err, s.errs = s.errs[0], s.errs[1:]
		found = true
	}
	return item, found, err
}
