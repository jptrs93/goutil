package contextu

import (
	"context"
	"sync"
	"time"
)

func OnCancel(ctx context.Context, cleanups ...func()) {
	go func() {
		<-ctx.Done()
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()
}

func ContextCauseWithCleanup(ctx context.Context, cleanups ...func()) (context.Context, context.CancelCauseFunc) {
	ctx, cancelCauseFunc := context.WithCancelCause(ctx)
	mu := sync.Mutex{}
	cancelCauseFuncWrapper := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		for _, cleanup := range cleanups {
			if cleanup != nil {
				cleanup()
			}
		}
		clear(cleanups)
		cancelCauseFunc(err)
	}
	return ctx, cancelCauseFuncWrapper
}

func ContextWithCleanup(ctx context.Context, cleanups ...func()) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	mu := sync.Mutex{}
	cancelCauseFuncWrapper := func() {
		mu.Lock()
		defer mu.Unlock()
		for _, cleanup := range cleanups {
			if cleanup != nil {
				cleanup()
			}
		}
		clear(cleanups)
		cancel()
	}
	return ctx, cancelCauseFuncWrapper
}

func WithTimeoutCancelCause(parent context.Context, timeout time.Duration) (context.Context, context.CancelCauseFunc) {
	child, cancel := context.WithCancelCause(parent)
	var once sync.Once
	var timer *time.Timer
	timer = time.AfterFunc(timeout, func() {
		once.Do(func() {
			cancel(context.DeadlineExceeded)
		})
	})
	return child, func(cause error) {
		once.Do(func() {
			timer.Stop()
			cancel(cause)
		})
	}
}
