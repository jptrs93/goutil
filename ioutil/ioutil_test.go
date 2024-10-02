package ioutil

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestLineStreamer(t *testing.T) {
	tests := []struct {
		input       string
		expected    [][]byte
		expectError bool
	}{
		{
			input:       "Hello, World!\nThis is a test.\nLast line.\n",
			expected:    [][]byte{[]byte("Hello, World!\n"), []byte("This is a test.\n"), []byte("Last line.\n"), []byte{}},
			expectError: false,
		},
		{
			input:       "Hello, World!\nThis is a test.\nLast line.",
			expected:    [][]byte{[]byte("Hello, World!\n"), []byte("This is a test.\n"), []byte("Last line.")},
			expectError: false,
		},
		{
			input:       "",
			expected:    [][]byte{[]byte("")},
			expectError: false, // Expect EOF error when there's no input
		},
	}

	for _, test := range tests {
		// Create a new LineStreamer
		input := bytes.NewBufferString(test.input)
		streamer := NewLineStreamer(input)

		var results [][]byte

		for {
			line, err := streamer.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				t.Fatalf("unexpected error: %v", err)
			}
			results = append(results, line)
		}

		// Check results against expected
		if test.expectError && len(results) > 0 {
			t.Fatalf("expected error but got results: %v", results)
		}

		if !test.expectError && len(results) != len(test.expected) {
			t.Fatalf("expected %d lines but got %d", len(test.expected), len(results))
		}

		for i, expectedLine := range test.expected {
			if i < len(results) {
				if !bytes.Equal(results[i], expectedLine) {
					t.Errorf("for input %q: expected %q but got %q", test.input, expectedLine, results[i])
				}
			}
		}
	}
}
