package sliceu

import (
	"bytes"
	"cmp"
	"fmt"
	"reflect"
	"testing"
)

func TestBisectFilter(t *testing.T) {

	type testCase struct {
		items     []int
		startLess func(item int) int
		endLess   func(item int) int
		want      []int
	}
	tests := []testCase{
		{
			items:     []int{1, 2, 2, 2, 3, 3, 3, 4, 4, 4, 5},
			startLess: func(item int) int { return cmp.Compare(item, 2) },
			endLess:   func(item int) int { return cmp.Compare(item, 4) },
			want:      []int{2, 2, 2, 3, 3, 3},
		},
		{
			items:     []int{1, 2, 3, 4, 5},
			startLess: func(item int) int { return cmp.Compare(item, 1) },
			endLess:   func(item int) int { return cmp.Compare(item, 4) },
			want:      []int{1, 2, 3},
		},
		{
			items:     []int{1, 2, 3, 4, 5},
			startLess: func(item int) int { return cmp.Compare(item, 1) },
			endLess:   func(item int) int { return cmp.Compare(item, 6) },
			want:      []int{1, 2, 3, 4, 5},
		},
		{
			items:     []int{1, 2, 3, 4, 5},
			startLess: func(item int) int { return cmp.Compare(item, 1) },
			endLess:   func(item int) int { return cmp.Compare(item, 5) },
			want:      []int{1, 2, 3, 4},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test case %v", i+1), func(t *testing.T) {
			if got := BisectFilterFunc(tt.items, tt.startLess, tt.endLess); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BisectFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBisectRight(t *testing.T) {

	type testCase struct {
		name  string
		items []int
		less  func(item int) int
		want  int
	}
	tests := []testCase{
		{
			name:  "Test case 1",
			items: []int{1, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) int { return cmp.Compare(item, 2) },
			want:  1,
		},
		{
			name:  "Test case 2",
			items: []int{1, 2, 2, 2, 2, 3, 4, 5},
			less: func(t int) int {
				if t <= 2 {
					return -1
				}
				return 1
			},
			want: 5,
		},
		{
			name:  "Test case 3",
			items: []int{2, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) int { return cmp.Compare(item, 2) },
			want:  0,
		},
		{
			name:  "Test case 4",
			items: []int{2, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) int { return cmp.Compare(item, 5) },
			want:  7,
		},
		{
			name:  "Test case 5",
			items: []int{2, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) int { return cmp.Compare(item, 6) },
			want:  8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BisectFunc(tt.items, tt.less)
			got = max(got, -got-1)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BisectFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	n := 6 * 4
	size := 100 * n
	randomBytes := make([]byte, size)

	unflat := Unflatten(randomBytes, n)

	if len(unflat) != 100 {
		t.Errorf("Unflatten() returned wrong length: got %v, want 100", len(unflat))
	}

	flat := Flatten(unflat)

	if !bytes.Equal(flat, randomBytes) {
		t.Errorf("Flatten() = %v, want %v", flat, randomBytes)
	}
}
