package contextutil

import (
	"context"
	"sync"
)

func ContextWithCleanup(ctx context.Context) (context.Context, context.CancelCauseFunc, func(func())) {
	ctx, cancelCauseFunc := context.WithCancelCause(context.Background())
	mu := sync.Mutex{}
	cleanups := []func(){}

	registerCleanup := func(cleanup func()) {
		mu.Lock()
		cleanups = append(cleanups, cleanup)
		mu.Unlock()
	}
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
	return ctx, cancelCauseFuncWrapper, registerCleanup
}
