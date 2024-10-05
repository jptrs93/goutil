package iteru

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"iter"
	"testing"
)

func TestSeqToIOReader(t *testing.T) {
	tests := []struct {
		name    string
		in      iter.Seq2[[]byte, error]
		want    []byte
		wantErr error
	}{
		{
			name:    "Test case 1",
			in:      LineSeq(bytes.NewReader([]byte("apples\noranges\ntomatoes"))),
			want:    []byte("applesorangestomatoes"),
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := Seq2ErrToIOReader(tt.in)
			b, err := io.ReadAll(reader)
			if !bytes.Equal(b, tt.want) {
				t.Errorf("bytes '%s' != expected '%s'", b, tt.want)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error '%v' != expected '%v'", err, tt.wantErr)
			}
		})
	}
}

func TestLengthPrefixedBytesIter(t *testing.T) {
	readErr := errors.New("")

	tests := []struct {
		name           string
		expectedChunks [][]byte
		expectedErrs   []error
	}{
		{
			name:           "No empty chunks",
			expectedChunks: [][]byte{[]byte("data 1"), []byte("data 2 ......."), []byte("data 3 .."), []byte("data 4 ....")},
			expectedErrs:   []error{nil, nil, nil, nil},
		},
		{
			name:           "Empty chunks",
			expectedChunks: [][]byte{[]byte("data 1"), []byte(""), []byte("data 3 .."), []byte("")},
			expectedErrs:   []error{nil, nil, nil, nil},
		},
		{
			name:           "Reader errors",
			expectedChunks: [][]byte{[]byte("data 1"), nil},
			expectedErrs:   []error{nil, readErr},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := 0
			b := []byte{}
			for _, chunk := range tt.expectedChunks {
				b = append(b, make([]byte, 4)...)
				binary.LittleEndian.PutUint32(b[len(b)-4:], uint32(len(chunk)))
				b = append(b, chunk...)
			}
			var reader io.Reader
			if tt.expectedErrs[len(tt.expectedErrs)-1] != nil {
				reader = &MockReader{
					b:   b[:len(b)-1],
					err: tt.expectedErrs[len(tt.expectedErrs)-1],
				}
			} else {
				reader = bytes.NewBuffer(b)
			}
			for chunk, err := range LengthPrefixedBytesSeq(reader) {
				if !bytes.Equal(chunk, tt.expectedChunks[i]) {
					t.Errorf("%v item chunk '%s' != expected '%s'", i+1, chunk, tt.expectedChunks[i])
				}
				if !errors.Is(err, tt.expectedErrs[i]) {
					t.Errorf("%v error '%v' != expected '%v'", i+1, err, tt.expectedErrs[i])
				}
				i++
			}
		})
	}
}

func TestSSEIter(t *testing.T) {

	readErr := errors.New("")

	tests := []struct {
		name           string
		in             io.Reader
		expectedEvents [][]byte
		expectedErrs   []error
	}{
		{
			name:           "Events no error",
			in:             bytes.NewReader([]byte("data: event 1\n\ndata: \n\ndata: event 2\n\ndata: event 3")),
			expectedEvents: [][]byte{[]byte("event 1"), []byte(""), []byte("event 2"), []byte("event 3")},
			expectedErrs:   []error{nil, nil, nil, nil},
		},
		{
			name: "Events with error",
			in: &MockReader{
				b:   []byte("data: event 1\n\ndata: event 2\n\ndata: ev.."),
				err: readErr,
			},
			expectedEvents: [][]byte{[]byte("event 1"), []byte("event 2"), nil},
			expectedErrs:   []error{nil, nil, readErr},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := 0
			for line, err := range SSESeq(tt.in) {
				if !bytes.Equal(line, tt.expectedEvents[i]) {
					t.Errorf("%v item line '%s' != expected '%s'", i+1, line, tt.expectedEvents[i])
				}
				if !errors.Is(err, tt.expectedErrs[i]) {
					t.Errorf("%v error '%v' != expected '%v'", i+1, err, tt.expectedErrs[i])
				}
				i++
			}
		})
	}
}

func TestLineIter(t *testing.T) {

	readErr := errors.New("")

	tests := []struct {
		name          string
		in            io.Reader
		expectedLines []string
		expectedErrs  []error
	}{
		{
			name:          "Lines without new line at end",
			in:            bytes.NewReader([]byte("apples\noranges\ntomatoes")),
			expectedLines: []string{"apples", "oranges", "tomatoes"},
			expectedErrs:  []error{nil, nil, nil},
		},
		{
			name:          "Lines with empty line and new line at end",
			in:            bytes.NewReader([]byte("apples\noranges\n\ntomatoes\n")),
			expectedLines: []string{"apples", "oranges", "", "tomatoes", ""},
			expectedErrs:  []error{nil, nil, nil, nil, nil},
		},
		{
			name: "Lines with error mid read",
			in: &MockReader{
				b:   []byte("apples\noranges\ntom"),
				err: readErr,
			},
			expectedLines: []string{"apples", "oranges", "tom"},
			expectedErrs:  []error{nil, nil, readErr},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := 0
			for line, err := range LineSeq(tt.in) {
				if !bytes.Equal(line, []byte(tt.expectedLines[i])) {
					t.Errorf("%v item line '%s' != expected '%s'", i+1, line, tt.expectedLines[i])
				}
				if !errors.Is(err, tt.expectedErrs[i]) {
					t.Errorf("%v error '%v' != expected '%v'", i+1, err, tt.expectedErrs[i])
				}
				i++
			}
		})
	}
}

type MockReader struct {
	b   []byte
	err error
}

func (r *MockReader) Read(p []byte) (n int, err error) {
	n = copy(p, r.b)
	r.b = r.b[n:]
	if len(r.b) == 0 {
		return n, r.err
	}
	return n, nil
}
