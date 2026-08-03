package lang

import "testing"

func TestKeyword(t *testing.T) {
	kw1 := NewKeyword("foo")
	kw2 := NewKeyword("foo")

	if kw1 != kw2 {
		t.Errorf("NewKeyword(\"foo\") != NewKeyword(\"foo\")")
	}
	if !kw1.Equals(kw2) {
		t.Errorf("kw1.Equals(kw2) == false")
	}

	kw3 := NewKeyword("not-foo")
	if kw1 == kw3 {
		t.Errorf("kw1 == kw3")
	}
	if kw1.Equals(kw3) {
		t.Errorf("kw1.Equals(kw3) == true")
	}
}

func TestKeywordFixedArityLookup(t *testing.T) {
	kw := NewKeyword("answer")
	m := NewMap(kw, int64(42))

	if got := Apply1(kw, m); got != int64(42) {
		t.Fatalf("Apply1(keyword, map) = %v, want 42", got)
	}
	if got := Apply1(kw, nil); got != nil {
		t.Fatalf("Apply1(keyword, nil) = %v, want nil", got)
	}
	if got := Apply2(kw, nil, "missing"); got != "missing" {
		t.Fatalf("Apply2(keyword, nil, default) = %v, want missing", got)
	}

	hashMap := NewPersistentHashMap(kw, int64(42))
	if got := testing.AllocsPerRun(1_000, func() {
		if value := kw.Invoke1(hashMap); value != int64(42) {
			panic(value)
		}
	}); got != 0 {
		t.Fatalf("keyword hash-map lookup allocated %v objects, want 0", got)
	}

	if got := testing.AllocsPerRun(1_000, func() {
		if value := kw.Invoke1(m); value != int64(42) {
			panic(value)
		}
	}); got != 0 {
		t.Fatalf("keyword array-map lookup allocated %v objects, want 0", got)
	}
}

func TestKeywordInvokesPersistentSet(t *testing.T) {
	keyword := NewKeyword("at")
	set := NewSet(keyword)
	if got := keyword.Invoke1(set); !Equals(got, keyword) {
		t.Fatalf("keyword set lookup = %v, want %v", got, keyword)
	}
}
