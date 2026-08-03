package lang

import "testing"

func TestUncheckedIntOperations(t *testing.T) {
	if got := Numbers.Unchecked_int_dec(int64(4)); got != int32(3) {
		t.Fatalf("unchecked int dec = %v, want 3", got)
	}
	if got := Numbers.Unchecked_int_add(int64(2), int64(5)); got != int32(7) {
		t.Fatalf("unchecked int add = %v, want 7", got)
	}
	if got := Numbers.Unchecked_int_inc(int64(2147483647)); got != int32(-2147483648) {
		t.Fatalf("unchecked int overflow = %v, want -2147483648", got)
	}
}
