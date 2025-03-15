package syncu

import (
	"hash"
	"hash/fnv"
	"sync"
	"unsafe"
)

type StripedIDLock[T comparable] struct {
	mutexes []sync.Mutex
	count   int
	pool    sync.Pool
}

func NewStripedIDLock[T comparable](count int) *StripedIDLock[T] {
	return &StripedIDLock[T]{
		mutexes: make([]sync.Mutex, count),
		count:   count,
		pool: sync.Pool{
			New: func() interface{} {
				return fnv.New32a()
			},
		},
	}
}

func (s *StripedIDLock[T]) hash(id T) int {
	hasher := s.pool.Get().(hash.Hash32)
	defer s.pool.Put(hasher)
	hasher.Reset()
	valBytes := unsafe.Slice((*byte)(unsafe.Pointer(&id)), unsafe.Sizeof(id))
	hasher.Write(valBytes)
	return int(hasher.Sum32())
}

func (s *StripedIDLock[T]) Lock(id T) {
	s.mutexes[s.hash(id)%s.count].Lock()
}

func (s *StripedIDLock[T]) Unlock(id T) {
	s.mutexes[s.hash(id)%s.count].Unlock()
}
