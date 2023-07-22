package lang

import "testing"

func TestKeyword(t *testing.T) {
	kw1 := NewKeyword("foo")
	kw2 := NewKeyword("foo")

	if kw1 != kw2 {
		t.Errorf("NewKeyword(\"foo\") != NewKeyword(\"foo\")")
	}
	if !kw1.Equal(kw2) {
		t.Errorf("kw1.Equal(kw2) == false")
	}

	kw3 := NewKeyword("not-foo")
	if kw1 == kw3 {
		t.Errorf("kw1 == kw3")
	}
	if kw1.Equal(kw3) {
		t.Errorf("kw1.Equal(kw3) == true")
	}
}
