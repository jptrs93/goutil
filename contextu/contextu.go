package contextu

import (
	"context"
	"sync"
)

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
