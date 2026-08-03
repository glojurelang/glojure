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
