package syncu

import (
	"sync"
)

type Lockable interface {
	uint | uint32 | uint64 | int | int32 | int64 | string
}

type paddedMutex struct {
	sync.Mutex
	_ [56]byte
}

type StripedLock[T Lockable] struct {
	mutexes []paddedMutex
	mask    uint32
}

func NewStripedLock[T Lockable](count int) *StripedLock[T] {
	n := 1
	for n < count {
		n <<= 1
	}
	return &StripedLock[T]{
		mutexes: make([]paddedMutex, n),
		mask:    uint32(n - 1),
	}
}

func (s *StripedLock[T]) hash(id T) uint32 {
	switch v := any(id).(type) {
	case string:
		return fnv1a32String(v)
	case int:
		return fnv1a32Uint64(uint64(v))
	case int32:
		return fnv1a32Uint64(uint64(v))
	case int64:
		return fnv1a32Uint64(uint64(v))
	case uint:
		return fnv1a32Uint64(uint64(v))
	case uint32:
		return fnv1a32Uint64(uint64(v))
	case uint64:
		return fnv1a32Uint64(v)
	default:
		panic("unreachable")
	}
}

func (s *StripedLock[T]) Lock(id T) {
	s.mutexes[s.hash(id)&s.mask].Lock()
}

func (s *StripedLock[T]) Unlock(id T) {
	s.mutexes[s.hash(id)&s.mask].Unlock()
}

func fnv1a32String(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func fnv1a32Uint64(v uint64) uint32 {
	// Process all 8 bytes
	h := uint32(2166136261)
	h ^= uint32(v & 0xFF)
	h *= 16777619
	h ^= uint32((v >> 8) & 0xFF)
	h *= 16777619
	h ^= uint32((v >> 16) & 0xFF)
	h *= 16777619
	h ^= uint32((v >> 24) & 0xFF)
	h *= 16777619
	h ^= uint32((v >> 32) & 0xFF)
	h *= 16777619
	h ^= uint32((v >> 40) & 0xFF)
	h *= 16777619
	h ^= uint32((v >> 48) & 0xFF)
	h *= 16777619
	h ^= uint32((v >> 56) & 0xFF)
	h *= 16777619
	return h
}
