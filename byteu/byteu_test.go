package byteu

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBytesToInt32Arr(t *testing.T) {
	tests := []struct {
		arr []int32
	}{
		{
			arr: []int32{1, 0, -5, 35_000_000, 52_349_999, 2_147_483_647, -2_147_483_648},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test case %v", i), func(t *testing.T) {
			b := Int32ArrToBytes(tt.arr)
			arr := BytesToInt32Arr(b)
			if !reflect.DeepEqual(arr, tt.arr) {
				t.Errorf("BytesToInt32Array() = %v, want %v", arr, tt.arr)
			}
			b = Int32ArrToBytes(tt.arr)
			arr = BytesToInt32Arr(b)
			if !reflect.DeepEqual(arr, tt.arr) {
				t.Errorf("Int32ArrToBytesUnsafe() = %v, want %v", arr, tt.arr)
			}
		})
	}
}
