package pythonu

import (
	"context"
	"embed"
	"math/rand"
	"reflect"
	"testing"
)

//go:embed test-scripts/*
var scriptsFS embed.FS

type AddInput struct {
	A int `json:"a,omitempty"`
	B int `json:"b,omitempty"`
}

type AddResult struct {
	Result int `json:"result,omitempty"`
}

type AddArraysInput struct {
	A Float64NumpyArray `json:"a,omitempty"`
	B Float64NumpyArray `json:"b,omitempty"`
}

type ArrayWrapper struct {
	Arr2D Float642DNumpyArray `json:"arr2D,omitempty"`
	Arr1D Float64NumpyArray   `json:"arr1D,omitempty"`
}

func TestCall(t *testing.T) {
	pythonEnv := "/Users/josspeters/repo/jptrs93/coflip-server/venv/bin/python"
	pp := NewPool(context.Background(), scriptsFS, pythonEnv, "test_script.py", 1)
	defer pp.Close()

	// build a test array
	columns := 10 + rand.Intn(20)
	var testArr Float642DNumpyArray
	for i := 0; i < 10+rand.Intn(20); i++ {
		var row []float64
		for c := 0; c < columns; c++ {
			row = append(row, rand.Float64())
		}
		testArr = append(testArr, row)
	}

	type testCase struct {
		name     string
		exec     func() (any, error)
		want     any
		wantType reflect.Type
		wantErr  bool
	}

	tests := []testCase{
		{
			name: "add with dict response",
			exec: func() (any, error) {
				return CallPool[AddResult](pp, "add", AddInput{5, 6})
			},
			want: AddResult{11},
		},
		{
			name: "add with scalar response",
			exec: func() (any, error) {
				return CallPool[int](pp, "add_scalar_output", AddInput{5, 6})
			},
			want: 11,
		},
		{
			name: "add two numpy arrays",
			exec: func() (any, error) {
				return CallPool[Float64NumpyArray](pp, "add_numpy_arrays", AddArraysInput{Float64NumpyArray{5.5, 3.5}, Float64NumpyArray{1, 2.1}})
			},
			want: Float64NumpyArray{6.5, 5.6},
		},
		{
			name: "unknown function Call",
			exec: func() (any, error) {
				return CallPool[AddResult](pp, "add_scalar_output_blaba", AddInput{5, 6})
			},
			wantErr: true,
		},
		{
			name: "verify numpy serialise float64 1d array",
			exec: func() (any, error) {
				return CallPool[ArrayWrapper](pp, "verify_1d_array", ArrayWrapper{Arr1D: Float64NumpyArray{1.2, 3.2, 99.1, -14.1}})
			},
			wantErr: false,
			want:    ArrayWrapper{Arr1D: Float64NumpyArray{1.2, 3.2, 99.1, -14.1}},
		},
		{
			name: "verify numpy serialise float64 2d array",
			exec: func() (any, error) {
				return CallPool[ArrayWrapper](pp, "verify_2d_array", ArrayWrapper{Arr2D: Float642DNumpyArray{{1.2, 3.2}, {99.1, -14.1}}})
			},
			wantErr: false,
			want:    ArrayWrapper{Arr2D: Float642DNumpyArray{{1.2, 3.2}, {99.1, -14.1}}},
		},
		{
			name: "numpy serialise/deserialise random array",
			exec: func() (any, error) {
				return CallPool[ArrayWrapper](pp, "identity", ArrayWrapper{Arr2D: testArr, Arr1D: testArr[0]})
			},
			wantErr: false,
			want:    ArrayWrapper{Arr2D: testArr, Arr1D: testArr[0]},
		},
		{
			name: "numpy serialise/deserialise basic",
			exec: func() (any, error) {
				return CallPool[ArrayWrapper](pp, "identity", ArrayWrapper{Arr2D: Float642DNumpyArray{{2.5, 1.34}, {1.1, 99.9}}})
			},
			wantErr: false,
			want:    ArrayWrapper{Arr2D: Float642DNumpyArray{{2.5, 1.34}, {1.1, 99.9}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.exec()
			if (err != nil) != tt.wantErr {
				t.Errorf("CallPool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CallPool() got = %v, want %v", got, tt.want)
			}
		})
	}
}
