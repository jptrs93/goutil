package sliceutil

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBisectFilter(t *testing.T) {

	type testCase struct {
		items     []int
		startLess func(item int) bool
		endLess   func(item int) bool
		want      []int
	}
	tests := []testCase{
		{
			items:     []int{1, 2, 2, 2, 3, 3, 3, 4, 4, 4, 5},
			startLess: func(item int) bool { return item < 2 },
			endLess:   func(item int) bool { return item < 4 },
			want:      []int{2, 2, 2, 3, 3, 3},
		},
		{
			items:     []int{1, 2, 3, 4, 5},
			startLess: func(item int) bool { return item < 1 },
			endLess:   func(item int) bool { return item < 4 },
			want:      []int{1, 2, 3},
		},
		{
			items:     []int{1, 2, 3, 4, 5},
			startLess: func(item int) bool { return item < 1 },
			endLess:   func(item int) bool { return item < 6 },
			want:      []int{1, 2, 3, 4, 5},
		},
		{
			items:     []int{1, 2, 3, 4, 5},
			startLess: func(item int) bool { return item < 1 },
			endLess:   func(item int) bool { return item < 5 },
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
		items []int
		less  func(item int) bool
		want  int
	}
	tests := []testCase{
		{
			items: []int{1, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) bool { return item < 2 },
			want:  1,
		},
		{
			items: []int{1, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) bool { return item <= 2 },
			want:  5,
		},
		{
			items: []int{2, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) bool { return item < 2 },
			want:  0,
		},
		{
			items: []int{2, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) bool { return item < 5 },
			want:  7,
		},
		{
			items: []int{2, 2, 2, 2, 2, 3, 4, 5},
			less:  func(item int) bool { return item < 6 },
			want:  8,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test case %v", i+1), func(t *testing.T) {
			if got := BisectFunc(tt.items, tt.less); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BisectFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
