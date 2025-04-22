package sliceu

import (
	"cmp"
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

func MapMapValues[K comparable, T, U any](m map[K]T, transform func(T) U) map[K]U {
	result := make(map[K]U, len(m))
	for k, v := range m {
		result[k] = transform(v)
	}
	return result
}

// Map applies a transformation function to each object in the slice
// and returns a new slice containing the transformed objects.
func Map[T, U any](slice []T, transform func(T) U) []U {
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

func AnyMatch[T any](items []T, f func(j T) bool) bool {
	for _, i := range items {
		if f(i) {
			return true
		}
	}
	return false
}

func CreateBatches[T any](s []T, batchSize int) [][]T {
	res := make([][]T, 0, (len(s)+batchSize-1)/batchSize)
	for i := 0; i < len(s); i += batchSize {
		res = append(res, s[i:min(i+batchSize, len(s))])
	}
	return res
}

func Flatten[T any](data [][]T) []T {
	if len(data) == 0 {
		return []T{}
	}
	buffer := make([]T, 0, len(data)*len(data[0]))
	for _, b := range data {
		buffer = append(buffer, b...)
	}
	return buffer
}

func Unflatten[T any](data []T, rowLength int) [][]T {
	result := make([][]T, 0, len(data)/rowLength)
	for i := 0; i < len(data); i += rowLength {
		end := i + rowLength
		if end > len(data) {
			end = len(data)
		}
		result = append(result, data[i:end])
	}
	return result
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

// Deduplicate removes duplicates from a slice, will overwrite existing slice - input slice must be pre sorted
func Deduplicate[T any](s []T, cmp func(a, b T) int) []T {
	if len(s) <= 1 {
		return s
	}

	writePos := 1
	for readPos := 1; readPos < len(s); readPos++ {
		// If current element is different from the previous unique element
		if cmp(s[readPos], s[writePos-1]) != 0 {
			// Only copy if write position is different from read position
			if writePos != readPos {
				s[writePos] = s[readPos]
			}
			writePos++
		}
	}
	return s[:writePos]
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

func SetMap[T comparable](s []T) map[T]struct{} {
	m := make(map[T]struct{}, len(s))
	for _, obj := range s {
		m[obj] = struct{}{}
	}
	return m
}

// -----------------------------------------------------------------
// binary search functions:
// - the cmp function should return negative for items to the left of the insertion point

func BisectFunc[T any](s []T, cmp func(T) int) int {
	low, high := 0, len(s)
	for low < high {
		mid := (low + high) / 2
		if cmp(s[mid]) < 0 {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < len(s) && cmp(s[low]) == 0 {
		return low
	}
	// no exact match, return -(insertion point + 1)
	return -(low + 1)
}

func InsortFunc[T any](s []T, value T, cmp func(a T) int) []T {
	pos := BisectFunc(s, cmp)
	pos = max(pos, -pos-1)
	s = append(s, value)
	copy(s[pos+1:], s[pos:])
	s[pos] = value
	return s
}

func BisectFilterFunc[T any](items []T, startCmp, endCmp func(T) int) []T {
	end := BisectFunc(items, endCmp)
	end = max(end, -end-1)
	if end == 0 {
		return nil
	}
	start := BisectFunc(items, startCmp)
	start = max(start, -start-1)
	if start >= end {
		return nil
	}
	return items[start:end]
}

func Bisect[T cmp.Ordered](s []T, v T) int {
	return BisectFunc(s, func(t T) int { return cmp.Compare(t, v) })
}

func BisectRight[T cmp.Ordered](s []T, v T) int {
	return BisectFunc(s, func(t T) int {
		if t <= v {
			return -1
		}
		return 1
	})
}

func BisectFilter[T cmp.Ordered](s []T, startInclusive, endExclusive T) []T {
	return BisectFilterFunc(s, func(t T) int { return cmp.Compare(t, startInclusive) }, func(t T) int { return cmp.Compare(t, endExclusive) })
}

func Insort[T cmp.Ordered](s []T, v T) []T {
	return InsortFunc(s, v, func(t T) int { return cmp.Compare(t, v) })
}

func InsortRight[T cmp.Ordered](s []T, v T) []T {
	return InsortFunc(s, v, func(t T) int {
		if t <= v {
			return -1
		}
		return 1
	})
}
