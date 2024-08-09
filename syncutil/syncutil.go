package syncutil

import (
	"hash"
	"hash/fnv"
	"sync"
)

type StripedIntIDLocks struct {
	mutexes []sync.Mutex
	count   int
}

func NewStripedIntIDLocks(count int) *StripedIntIDLocks {
	return &StripedIntIDLocks{
		mutexes: make([]sync.Mutex, count),
		count:   count,
	}
}

func (s *StripedIntIDLocks) hash(id int) int {
	return id % s.count
}

func (s *StripedIntIDLocks) Lock(id int) {
	s.mutexes[s.hash(id)].Lock()
}

func (s *StripedIntIDLocks) Unlock(id int) {
	s.mutexes[s.hash(id)].Unlock()
}

type StripedStringIDLocks struct {
	mutexes []sync.Mutex
	count   int
	pool    sync.Pool
}

func NewStripedIDLocks(count int) *StripedStringIDLocks {
	return &StripedStringIDLocks{
		mutexes: make([]sync.Mutex, count),
		count:   count,
		pool: sync.Pool{
			New: func() interface{} {
				return fnv.New32a()
			},
		},
	}
}

func (s *StripedStringIDLocks) hash(id string) int {
	hasher := s.pool.Get().(hash.Hash32)
	defer s.pool.Put(hasher)
	hasher.Reset()
	hasher.Write([]byte(id))
	return int(hasher.Sum32())
}

func (s *StripedStringIDLocks) Lock(id string) {
	s.mutexes[s.hash(id)%s.count].Lock()
}

func (s *StripedStringIDLocks) Unlock(id string) {
	s.mutexes[s.hash(id)%s.count].Unlock()
}

type IDLocks struct {
	m SyncMap[string, *sync.Mutex]
}

func (l *IDLocks) Lock(id string) {
	mutex, _ := l.m.LoadOrStore(id, &sync.Mutex{})
	mutex.Lock()
}

func (l *IDLocks) Unlock(id string) {
	if mutex, ok := l.m.Load(id); ok {
		mutex.Unlock()
	}
}

type RWIDLocks struct {
	m SyncMap[string, *sync.RWMutex] // Map of string ID to *sync.Mutex
}

func (l *RWIDLocks) Lock(id string) {
	mutex, _ := l.m.LoadOrStore(id, &sync.RWMutex{})
	mutex.Lock()
}

func (l *RWIDLocks) Unlock(id string) {
	if mutex, ok := l.m.Load(id); ok {
		mutex.Unlock()
	}
}

func (l *RWIDLocks) RLock(id string) {
	mutex, _ := l.m.LoadOrStore(id, &sync.RWMutex{})
	mutex.RLock()
}

func (l *RWIDLocks) RUnlock(id string) {
	if mutex, ok := l.m.Load(id); ok {
		mutex.RUnlock()
	}
}

type SyncMap[K comparable, V any] struct {
	m sync.Map
}

func (m *SyncMap[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	v, ok := m.m.Load(key)
	if ok {
		return v.(V), true
	}
	var zeroV V // Default zero value of type V
	return zeroV, false
}

func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(k, v interface{}) bool {
		return f(k.(K), v.(V))
	})
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.m.Delete(key)
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	actual, loaded := m.m.LoadOrStore(key, value)
	if loaded {
		return actual.(V), true
	}
	return value, false
}

func (m *SyncMap[K, V]) Values() []V {
	var values []V
	m.m.Range(func(_, v interface{}) bool {
		values = append(values, v.(V))
		return true
	})
	return values
}

func (m *SyncMap[K, V]) ValuesMatching(predicate func(V) bool) []V {
	var values []V
	m.Range(func(_ K, v V) bool {
		if predicate(v) {
			values = append(values, v)
		}
		return true
	})
	return values
}

func (m *SyncMap[K, V]) FindValue(predicate func(V) bool) (V, bool) {
	var foundValue V
	var found bool
	m.Range(func(_ K, value V) bool {
		if predicate(value) {
			foundValue = value
			found = true
			return false // Stop the iteration
		}
		return true // Continue the iteration
	})
	return foundValue, found
}

func RunAllToCompletion(funcs []func()) {
	wg := sync.WaitGroup{}
	for _, f := range funcs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()
}

type Broadcaster struct {
	mu sync.Mutex
	c  *sync.Cond
	v  int
	wg sync.WaitGroup
	N  int
}

func NewBroadcaster(n int) *Broadcaster {
	b := &Broadcaster{
		N: n,
		v: -1,
	}
	b.c = sync.NewCond(&b.mu)
	return b
}

func (b *Broadcaster) WaitForBroadcast(broadcastID int) {
	b.mu.Lock()
	for b.v < broadcastID {
		b.c.Wait()
	}
	b.mu.Unlock()
}

func (b *Broadcaster) Broadcast() {
	b.mu.Lock()
	b.v = b.v + 1
	b.wg.Add(b.N)
	b.c.Broadcast()
	b.mu.Unlock()
}

func (b *Broadcaster) WorkerDone() {
	b.wg.Done()
}

func (b *Broadcaster) WaitForWorkersDone() {
	b.wg.Wait()
}

/*
Example usage:

func Consumer(ctx context.Context, b *Broadcaster) {
	var i int64 = 0
	for {
		b.WaitForBroadcast(i)
		select {
		case <-ctx.Done():
			return
		default:

			// do work
			b.WorkerDone()
		}
	}
}

func Run() {
	ctx, cancel := context.WithCancel(context.Background())

	consumers := 50

	b := NewBroadcaster(consumers)
	for i := 0; i < consumers; i++ {
		go Consumer(ctx, b)
	}

	for i := 0; i < 10; i++ {
		b.Broadcast()
		b.WaitForWorkersDone()
	}
	cancel()
	// final broadcast to exist consumers
	b.Broadcast()
}

*/
