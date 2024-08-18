package byteutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"
)

func LoadNx3Int32Unsafe(data []byte) [][3]int32 {
	// todo: untested
	n := len(data)
	if n%(3*4) != 0 {
		panic(fmt.Sprintf("expected 3 int32's per row but file has %v bytes which is not divisiable by 12", n))
	}
	numRows := n / (3 * 4)
	int32Slice := *(*[][3]int32)(unsafe.Pointer(&data))
	reshapedSlice := int32Slice[:numRows]

	return reshapedSlice
}

func LoadNx6Int32Unsafe(data []byte) [][6]int32 {
	// todo: untested
	n := len(data)
	if n%(6*4) != 0 {
		panic(fmt.Sprintf("expected 6 int32's per row but file has %v bytes which is not divisiable by 12", n))

	}
	numRows := n / (6 * 4)
	int32Slice := *(*[][6]int32)(unsafe.Pointer(&data))
	reshapedSlice := int32Slice[:numRows]

	return reshapedSlice
}
func Equal2DBytes(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func BytesToInt32ArrUnsafe(arr []byte) []int32 {
	return *(*[]int32)(unsafe.Pointer(&arr))
}

func BytesToInt32Arr(arr []byte) []int32 {
	if len(arr)%4 != 0 {
		panic("input byte slice length must be a multiple of 4")
	}
	ints := make([]int32, len(arr)/4)
	for i := 0; i < len(ints); i++ {
		ints[i] = int32(binary.LittleEndian.Uint32(arr[i*4 : i*4+4]))
	}
	return ints
}

func Int32ArrToBytes(arr []int32) []byte {
	b := make([]byte, 4*len(arr))
	for i, v := range arr {
		binary.LittleEndian.PutUint32(b[i*4:], uint32(v))
	}
	return b
}
