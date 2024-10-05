package syncu

import (
	"hash"
	"hash/fnv"
	"reflect"
	"sync"
	"unsafe"
)

type StripedStringIDLocks[T comparable] struct {
	mutexes []sync.Mutex
	count   int
	pool    sync.Pool
}

func NewStripedIDLocks[T comparable](count int) *StripedStringIDLocks[T] {
	return &StripedStringIDLocks[T]{
		mutexes: make([]sync.Mutex, count),
		count:   count,
		pool: sync.Pool{
			New: func() interface{} {
				return fnv.New32a()
			},
		},
	}
}

func (s *StripedStringIDLocks[T]) hash(id T) int {
	hasher := s.pool.Get().(hash.Hash32)
	defer s.pool.Put(hasher)
	hasher.Reset()
	valBytes := unsafe.Slice((*byte)(unsafe.Pointer(&id)), unsafe.Sizeof(id))
	hasher.Write(valBytes)
	return int(hasher.Sum32())
}

func (s *StripedStringIDLocks[T]) Lock(id T) {
	s.mutexes[s.hash(id)%s.count].Lock()
}

func (s *StripedStringIDLocks[T]) Unlock(id T) {
	s.mutexes[s.hash(id)%s.count].Unlock()
}

type IDLocks[T comparable] struct {
	m SyncMap[T, *sync.Mutex]
}

func (l *IDLocks[T]) Lock(id T) {
	mutex, _ := l.m.LoadOrStore(id, &sync.Mutex{})
	mutex.Lock()
}

func (l *IDLocks[T]) Unlock(id T) {
	if mutex, ok := l.m.Load(id); ok {
		mutex.Unlock()
	}
}

type RWIDLocks[T comparable] struct {
	m SyncMap[T, *sync.RWMutex] // Map of string ID to *sync.Mutex
}

func (l *RWIDLocks[T]) Lock(id T) {
	mutex, _ := l.m.LoadOrStore(id, &sync.RWMutex{})
	mutex.Lock()
}

func (l *RWIDLocks[T]) Unlock(id T) {
	if mutex, ok := l.m.Load(id); ok {
		mutex.Unlock()
	}
}

func (l *RWIDLocks[T]) RLock(id T) {
	mutex, _ := l.m.LoadOrStore(id, &sync.RWMutex{})
	mutex.RLock()
}

func (l *RWIDLocks[T]) RUnlock(id T) {
	if mutex, ok := l.m.Load(id); ok {
		mutex.RUnlock()
	}
}

type SyncMap[K comparable, V any] struct {
	M sync.Map
}

func (m *SyncMap[K, V]) Clear() {
	m.M.Clear()
}

func (m *SyncMap[K, V]) Store(key K, value V) {
	m.M.Store(key, value)
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	v, ok := m.M.Load(key)
	if ok {
		return v.(V), true
	}
	var zeroV V // Default zero value of type V
	return zeroV, false
}

func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.M.Range(func(k, v interface{}) bool {
		return f(k.(K), v.(V))
	})
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.M.Delete(key)
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	actual, loaded := m.M.LoadOrStore(key, value)
	if loaded {
		return actual.(V), true
	}
	return value, false
}

func (m *SyncMap[K, V]) Values() []V {
	var values []V
	m.M.Range(func(_, v interface{}) bool {
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

type Shared[T any] struct {
	mu    sync.RWMutex
	value T
}

func NewShared[T any](val T) *Shared[T] {
	return &Shared[T]{value: val}
}

func (v *Shared[T]) Set(val T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.value = val
}

func (v *Shared[T]) Value() T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.value
}

type CondObj[T any] struct {
	condObj T
	cond    *sync.Cond
	mu      *sync.Mutex
}

func NewCondObj[T any](obj T) *CondObj[T] {
	mu := &sync.Mutex{}
	return &CondObj[T]{
		condObj: obj,
		cond:    sync.NewCond(mu),
		mu:      mu,
	}
}

func (c *CondObj[T]) Set(v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.condObj = v
	c.cond.Broadcast()
}

func (c *CondObj[T]) SetFunc(updateFunc func(T) T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.condObj = updateFunc(c.condObj)
	c.cond.Broadcast()
}

func (c *CondObj[T]) WaitFor(v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for !reflect.DeepEqual(v, c.condObj) {
		c.cond.Wait()
	}
}

func (c *CondObj[T]) WaitForFunc(cond func(T) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for !cond(c.condObj) {
		c.cond.Wait()
	}
}

func (c *CondObj[T]) Get() T {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.condObj
}
