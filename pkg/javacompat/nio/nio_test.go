package nio

import "testing"

func TestUTF8Decoder(t *testing.T) {
	decoder := UTF8.NewDecoder().OnMalformedInput(REPORT)
	if got := decoder.Decode(Wrap([]int8{-61, -68})); got != "ü" {
		t.Fatalf("decoded = %q, want ü", got)
	}
	defer func() {
		if _, ok := recover().(*CharacterCodingException); !ok {
			t.Fatal("invalid UTF-8 did not raise CharacterCodingException")
		}
	}()
	decoder.Decode(Wrap([]byte{0xff}))
}

func TestByteBufferPositionLimitAndSlice(t *testing.T) {
	buffer := Wrap([]byte("Duty Now For The Future"))
	if got := buffer.Remaining(); got != 23 {
		t.Fatalf("remaining = %d, want 23", got)
	}
	if got := buffer.Position(int64(5)); got != buffer {
		t.Fatalf("position setter returned %T, want buffer", got)
	}
	buffer.Limit(int64(9))
	if got := buffer.Remaining(); got != 4 {
		t.Fatalf("remaining = %d, want 4", got)
	}
	if got := buffer.Get(int64(5)); got != int8('N') {
		t.Fatalf("get(5) = %d, want %d", got, 'N')
	}
	slice := buffer.Slice()
	if got := slice.Remaining(); got != 4 {
		t.Fatalf("slice remaining = %d, want 4", got)
	}
	if got := slice.Get(int64(0)); got != int8('N') {
		t.Fatalf("slice get(0) = %d, want %d", got, 'N')
	}
}
