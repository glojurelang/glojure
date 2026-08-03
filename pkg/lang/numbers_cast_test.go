package lang

import (
	"reflect"
	"testing"
)

func TestBytesAcceptsUnsignedGoByteSlice(t *testing.T) {
	input := []byte{0, 127, 128, 255}
	got := Numbers.Bytes(input)
	want := []int8{0, 127, -128, -1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Bytes(%v) = %v, want %v", input, got, want)
	}
}

func TestShortCastRejectsFractionalValuesOutsideRange(t *testing.T) {
	for _, value := range []float64{-32768.000001, 32767.000001} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("ShortCast(%v) did not panic", value)
				}
			}()
			ShortCast(value)
		}()
	}
}
