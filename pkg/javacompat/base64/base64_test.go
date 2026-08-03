package base64

import "testing"

func TestBase64RoundTrip(t *testing.T) {
	encoded := GetEncoder().EncodeToString([]int8{'h', 'i'})
	if encoded != "aGk=" {
		t.Fatalf("encoded = %q, want aGk=", encoded)
	}
	if decoded := GetDecoder().Decode(encoded); !EqualsBytes(decoded, []int8{'h', 'i'}) {
		t.Fatalf("decoded = %v, want hi", decoded)
	}
}

func TestBase64EncoderEncodeReturnsJavaBytes(t *testing.T) {
	got := GetEncoder().Encode([]int8{1, 2, 3})
	want := []int8{'A', 'Q', 'I', 'D'}
	if !EqualsBytes(got, want) {
		t.Fatalf("Encode = %v, want %v", got, want)
	}
}

func EqualsBytes(left, right []int8) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
