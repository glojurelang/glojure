package uuid

import "testing"

func TestIsUUID(t *testing.T) {
	if !IsUUID(RandomUUID()) {
		t.Fatal("RandomUUID result is not recognized as a UUID")
	}
	if IsUUID("not-a-uuid") {
		t.Fatal("string was recognized as a UUID")
	}
}
