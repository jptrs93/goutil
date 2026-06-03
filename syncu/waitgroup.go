package syncu

import (
	"sync"
)

var m = sync.Map{}

func WaitGroupDone(wg *sync.WaitGroup) bool {
	ch, loaded := m.LoadOrStore(wg, make(chan struct{}))
	if !loaded {
		go func() {
			wg.Wait()
			close(ch.(chan struct{}))
			m.Delete(wg)
		}()
	}
	select {
	case <-ch.(chan struct{}):
		return true
	default:
		return false
	}
}
