package arrays

import "testing"

func TestEqualsByteRepresentations(t *testing.T) {
	if !Equals([]byte{0, 255}, []int8{0, -1}) {
		t.Fatal("signed and unsigned byte storage should compare equally")
	}
	if Equals([]byte{1}, []byte{2}) {
		t.Fatal("different arrays compared equal")
	}
}
