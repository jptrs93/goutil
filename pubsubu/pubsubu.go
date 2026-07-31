package pubsubu

import (
	"log/slog"
	"sync"
	"sync/atomic"
)

const DefaultChanBuffer = 1_000

type Sub[T any] struct {
	Filter func(T, T) bool
	Ch     chan T

	InitialValue      T
	InitialValueValid bool

	unsubscribe func()
}

func (s *Sub[T]) Unsubscribe() {
	s.unsubscribe()
}

type PubSub[T any] struct {
	mu         sync.Mutex
	subs       []*Sub[T]
	last       atomic.Pointer[T]
	chanBuffer int
}

func NewPubSub[T any](existing T, chanBuffer int) *PubSub[T] {
	s := &PubSub[T]{chanBuffer: chanBuffer}
	s.last.Store(&existing)
	return s
}

func (s *PubSub[T]) Subscribe(f func(T, T) bool) *Sub[T] {
	s.mu.Lock()
	defer s.mu.Unlock()

	buffer := s.chanBuffer
	if buffer <= 0 {
		buffer = DefaultChanBuffer
	}

	sub := &Sub[T]{
		Filter: f,
		Ch:     make(chan T, buffer),
	}
	if p := s.last.Load(); p != nil {
		sub.InitialValue = *p
		sub.InitialValueValid = true
	}
	s.subs = append(s.subs, sub)

	sub.unsubscribe = func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, current := range s.subs {
			if current == sub {
				s.subs = append(s.subs[:i], s.subs[i+1:]...)
				close(sub.Ch)
				return
			}
		}
	}

	return sub
}

func (s *PubSub[T]) Value() T {
	if p := s.last.Load(); p != nil {
		return *p
	}
	var zero T
	return zero
}

func (s *PubSub[T]) ValueOK() (T, bool) {
	if p := s.last.Load(); p != nil {
		return *p, true
	}
	var zero T
	return zero, false
}

func (s *PubSub[T]) Notify(value T) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.notifyLocked(value)
}

func (s *PubSub[T]) UpdateAndNotify(update func(existing T) T) T {
	if s == nil {
		var zero T
		return zero
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var existing T
	if p := s.last.Load(); p != nil {
		existing = *p
	}
	value := update(existing)
	s.notifyLocked(value)

	return value
}

func (s *PubSub[T]) notifyLocked(value T) {
	var existing T
	if p := s.last.Load(); p != nil {
		existing = *p
	}

	for _, sub := range s.subs {
		if sub.Filter != nil && !sub.Filter(existing, value) {
			continue
		}
		select {
		case sub.Ch <- value:
		default:
			slog.Warn("subscription channel full, dropping notification")
		}
	}
	s.last.Store(&value)
}
