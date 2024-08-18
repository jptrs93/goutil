package sliceutil

import (
	"cmp"
	"golang.org/x/exp/constraints"
	"time"
)

func GroupBy[K comparable, T any](objects []T, keyFunc func(T) K) map[K][]T {
	result := make(map[K][]T, len(objects))
	for _, obj := range objects {
		key := keyFunc(obj)
		result[key] = append(result[key], obj)
	}
	return result
}

func AsMap[K comparable, T any](objects []T, keyFunc func(T) K) map[K]T {
	result := make(map[K]T, len(objects))
	for _, obj := range objects {
		key := keyFunc(obj)
		result[key] = obj
	}
	return result
}

func MapKeys[K comparable, T any](m map[K]T) []K {
	result := make([]K, 0, len(m))
	for k, _ := range m {
		result = append(result, k)
	}
	return result
}

func MapValues[K comparable, T any](m map[K]T) []T {
	result := make([]T, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

func TransformedMapValues[K comparable, T, U any](m map[K]T, transform func(T) U) []U {
	result := make([]U, 0, len(m))
	for _, v := range m {
		result = append(result, transform(v))
	}
	return result
}

// Transform applies a transformation function to each object in the slice
// and returns a new slice containing the transformed objects.
func Transform[T, U any](slice []T, transform func(T) U) []U {
	if slice == nil {
		return nil
	}
	transformedSlice := make([]U, len(slice))
	for i, obj := range slice {
		transformedSlice[i] = transform(obj)
	}
	return transformedSlice
}

func Filter[T any](s []T, predicate func(T) bool) []T {
	filtered := make([]T, 0, len(s))
	for _, obj := range s {
		if predicate(obj) {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}

func CompareDesc[T cmp.Ordered](a, b T) int {
	if a > b {
		return -1
	}
	if a < b {
		return 1
	}
	return 0
}

func CompareAsc[T cmp.Ordered](a, b T) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}

func CompareTimeAsc(a, b time.Time) int {
	if a.Before(b) {
		return -1
	}
	if b.Before(a) {
		return 1
	}
	return 0
}

func CompareTimeDesc(a, b time.Time) int {
	if a.Before(b) {
		return 1
	}
	if b.Before(a) {
		return -1
	}
	return 0
}

func AnyMatch[T any](items []T, f func(j T) bool) bool {
	for _, i := range items {
		if f(i) {
			return true
		}
	}
	return false
}

func InsortFunc[T any](s []T, value T, less func(a T) bool) []T {
	pos := BisectFunc(s, less)
	s = append(s, value)
	copy(s[pos+1:], s[pos:])
	s[pos] = value
	return s
}

func BisectFilterFunc[T any](items []T, startLess, endLess func(item T) bool) []T {
	end := BisectFunc(items, endLess)
	if end == 0 {
		return nil
	}
	start := BisectFunc(items, startLess)
	if start >= end {
		return nil
	}
	return items[start:end]
}

// BisectFunc less should return true for items left of the insertion point
func BisectFunc[T any](s []T, less func(T) bool) int {
	low, high := 0, len(s)
	for low < high {
		mid := (low + high) / 2
		if less(s[mid]) {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low
}

func BisectFilter[T constraints.Ordered](s []T, startInclusive, endExclusive T) []T {
	return BisectFilterFunc(s, func(t T) bool { return t < startInclusive }, func(t T) bool { return t < endExclusive })
}

func Bisect[T constraints.Ordered](s []T, v T) int {
	return BisectFunc(s, func(t T) bool {
		return t < v
	})
}

func BisectRight[T constraints.Ordered](s []T, v T) int {
	return BisectFunc(s, func(t T) bool {
		return t <= v
	})
}
func Insort[T constraints.Ordered](s []T, v T) []T {
	return InsortFunc(s, v, func(t T) bool {
		return t < v
	})
}

func InsortRight[T constraints.Ordered](s []T, v T) []T {
	return InsortFunc(s, v, func(t T) bool {
		return t <= v
	})
}

func Copy[T any](original []T) []T {
	newSlice := make([]T, len(original))
	copy(newSlice, original)
	return newSlice
}

func RemoveDuplicates[T comparable](s []T) []T {
	seen := make(map[T]struct{})
	result := []T{}
	for _, item := range s {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func Generate[T any](n int, producer func() T) []T {
	res := make([]T, 0, n)
	for i := 0; i < n; i++ {
		res = append(res, producer())
	}
	return res
}

func Reverse[T any](s []T) []T {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func BoolMap[T comparable](s []T) map[T]bool {
	m := make(map[T]bool, len(s))
	for _, obj := range s {
		m[obj] = true
	}
	return m
}

func Flatten[T any](s [][]T) []T {
	res := make([]T, 0, len(s))
	for _, obj := range s {
		res = append(res, obj...)
	}
	return res
}

func CreateBatches[T any](s []T, batchSize int) [][]T {
	res := make([][]T, 0, (len(s)+batchSize-1)/batchSize)
	for i := 0; i < len(s); i += batchSize {
		res = append(res, s[i:min(i+batchSize, len(s))])
	}
	return res
}
