package syncu

import (
	"context"
)

type ChanLock chan struct{}

func NewChanLock(n int) ChanLock {
	return make(chan struct{}, n)
}

// Lock acquires the lock with context support
func (l ChanLock) Lock(ctx context.Context) error {
	select {
	case l <- struct{}{}: // Acquire lock
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l ChanLock) Unlock() {
	select {
	case <-l:
	default:
		panic("unlock of unlocked channel")
	}
}

func (l ChanLock) TryLock() bool {
	select {
	case l <- struct{}{}:
		return true
	default:
		return false
	}
}
