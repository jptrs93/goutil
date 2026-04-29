package erru

import (
	"errors"
	"testing"
)

func TestMustReturnsValue(t *testing.T) {
	v := Must(123, nil)
	if v != 123 {
		t.Fatalf("Must() = %v, want %v", v, 123)
	}
}

func TestMustPanicsOnError(t *testing.T) {
	err := errors.New("boom")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Must() did not panic")
		}
		panicErr, ok := r.(error)
		if !ok {
			t.Fatalf("Must() panic = %T, want error", r)
		}
		if !errors.Is(panicErr, err) {
			t.Fatalf("Must() panic = %v, want %v", panicErr, err)
		}
	}()

	Must(0, err)
}
