package pythonu

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

type Float322DNumpyArray [][]float32

func (f Float322DNumpyArray) MarshalJSON() ([]byte, error) {
	l1 := len(f)
	l2 := 0
	if l1 > 0 {
		l2 = len(f[0])
	}
	byteSlice := make([]byte, l1*l2*4)

	for i, row := range f {
		for j, v := range row {
			offset := (i*l2 + j) * 4
			binary.LittleEndian.PutUint32(byteSlice[offset:offset+4], math.Float32bits(v))
		}
	}
	encoded := base64.StdEncoding.EncodeToString(byteSlice)
	result := map[string]interface{}{
		"_elementType": "float32",
		"_shape":       [2]int{l1, l2},
		"_data":        encoded,
	}
	return json.Marshal(result)
}

func (f *Float322DNumpyArray) UnmarshalJSON(data []byte) error {
	var temp struct {
		ElementType string `json:"_elementType"`
		Shape       [2]int `json:"_shape"`
		Data        string `json:"_data"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if temp.ElementType != "float32" {
		return fmt.Errorf("unexpected element type: %s", temp.ElementType)
	}

	byteSlice, err := base64.StdEncoding.DecodeString(temp.Data)
	if err != nil {
		return err
	}

	l1, l2 := temp.Shape[0], temp.Shape[1]
	*f = make(Float322DNumpyArray, l1)
	for i := range *f {
		(*f)[i] = make([]float32, l2)
		for j := range (*f)[i] {
			offset := (i*l2 + j) * 4
			(*f)[i][j] = math.Float32frombits(binary.LittleEndian.Uint32(byteSlice[offset : offset+4]))
		}
	}

	return nil
}

type Float642DNumpyArray [][]float64

func (f Float642DNumpyArray) MarshalJSON() ([]byte, error) {
	l1 := len(f)
	l2 := 0
	if l1 > 0 {
		l2 = len(f[0])
	}
	byteSlice := make([]byte, l1*l2*8)

	for i, row := range f {
		for j, v := range row {
			offset := (i*l2 + j) * 8
			binary.LittleEndian.PutUint64(byteSlice[offset:offset+8], math.Float64bits(v))
		}
	}
	encoded := base64.StdEncoding.EncodeToString(byteSlice)
	result := map[string]interface{}{
		"_elementType": "float64",
		"_shape":       [2]int{l1, l2},
		"_data":        encoded,
	}
	return json.Marshal(result)
}

// UnmarshalJSON implements the custom JSON unmarshaling for Float642DNumpyArray
func (f *Float642DNumpyArray) UnmarshalJSON(data []byte) error {
	var temp struct {
		ElementType string `json:"_elementType"`
		Shape       [2]int `json:"_shape"`
		Data        string `json:"_data"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if temp.ElementType != "float64" {
		return fmt.Errorf("unexpected element type: %s", temp.ElementType)
	}

	byteSlice, err := base64.StdEncoding.DecodeString(temp.Data)
	if err != nil {
		return err
	}

	l1, l2 := temp.Shape[0], temp.Shape[1]
	*f = make(Float642DNumpyArray, l1)
	for i := range *f {
		(*f)[i] = make([]float64, l2)
		for j := range (*f)[i] {
			offset := (i*l2 + j) * 8
			(*f)[i][j] = math.Float64frombits(binary.LittleEndian.Uint64(byteSlice[offset : offset+8]))
		}
	}

	return nil
}

type Float64NumpyArray []float64

func (f Float64NumpyArray) MarshalJSON() ([]byte, error) {
	byteSlice := make([]byte, len(f)*8)
	for i, floatVal := range f {
		binary.LittleEndian.PutUint64(byteSlice[i*8:(i+1)*8], math.Float64bits(floatVal))
	}
	encoded := base64.StdEncoding.EncodeToString(byteSlice)
	result := map[string]string{
		"_elementType": "float64",
		"_data":        encoded,
	}
	return json.Marshal(result)
}

// UnmarshalJSON implements the custom JSON unmarshaling for Float64NumpyArray
func (f *Float64NumpyArray) UnmarshalJSON(data []byte) error {
	var temp struct {
		ElementType string `json:"_elementType"`
		Data        string `json:"_data"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if temp.ElementType != "float64" {
		return fmt.Errorf("unexpected element type: %s", temp.ElementType)
	}

	byteSlice, err := base64.StdEncoding.DecodeString(temp.Data)
	if err != nil {
		return err
	}

	*f = make(Float64NumpyArray, len(byteSlice)/8)
	for i := range *f {
		offset := i * 8
		(*f)[i] = math.Float64frombits(binary.LittleEndian.Uint64(byteSlice[offset : offset+8]))
	}

	return nil
}
