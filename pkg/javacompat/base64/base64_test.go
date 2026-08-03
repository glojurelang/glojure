package base64

import "testing"

func TestBase64RoundTrip(t *testing.T) {
	encoded := GetEncoder().EncodeToString([]int8{'h', 'i'})
	if encoded != "aGk=" {
		t.Fatalf("encoded = %q, want aGk=", encoded)
	}
	if decoded := string(GetDecoder().Decode(encoded)); decoded != "hi" {
		t.Fatalf("decoded = %q, want hi", decoded)
	}
}
