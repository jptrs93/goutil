package pubsubu

import (
	"log/slog"
	"sync"
)

type Sub[T any] struct {
	Filter func(T, T) bool
	Ch     chan T

	InitialValue      T
	InitialValueValid bool
	UnsubscribeFunc   func()
}

type PubSub[T any] struct {
	Subs           []*Sub[T]
	Mu             sync.Mutex
	LastValue      T
	LastValueValid bool
}

func (s *PubSub[T]) Subscribe(f func(T, T) bool) *Sub[T] {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	sub := &Sub[T]{
		Filter:            f,
		Ch:                make(chan T, 1_000),
		InitialValue:      s.LastValue,
		InitialValueValid: s.LastValueValid,
	}
	s.Subs = append(s.Subs, sub)

	sub.UnsubscribeFunc = func() {
		s.Mu.Lock()
		defer s.Mu.Unlock()
		for i, current := range s.Subs {
			if current == sub {
				s.Subs = append(s.Subs[:i], s.Subs[i+1:]...)
				close(sub.Ch)
				return
			}
		}
	}

	return sub
}

func (s *PubSub[T]) Value() T {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.LastValue
}

func (s *PubSub[T]) Notify(value T) {
	if s == nil {
		return
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	// Send under the lock: UnsubscribeFunc closes Ch under the same lock, and a send on a closed channel panics.
	for _, sub := range s.Subs {
		if sub.Filter != nil && !sub.Filter(s.LastValue, value) {
			continue
		}
		select {
		case sub.Ch <- value:
		default:
			slog.Warn("subscription channel full, dropping notification")
		}
	}
	s.LastValue = value
	s.LastValueValid = true
}
