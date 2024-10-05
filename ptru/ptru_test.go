package ptru

import (
	"testing"
)

func TestNilRef(t *testing.T) {

	var x [][]string
	y := NilRef(x)
	if y != nil {
		t.Errorf("bad result")
	}
}
