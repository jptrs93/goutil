package mathutil

import (
	"math"
	"sort"
)

func AbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func AbsInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func ApproxEqual(x, y int64, tolerance float64) bool {
	if x == y {
		return true
	}
	diff := math.Abs(float64(x - y))
	maxVal := math.Max(math.Abs(float64(x)), math.Abs(float64(y)))
	relDiff := diff / maxVal
	return relDiff <= tolerance
}

func ContainsInvalid(x []float64) bool {
	for _, v := range x {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return true
		}
	}
	return false
}

func Median(values []float64) float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func Mean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}