package lang

import (
	"testing"
)

// TestVectorPopReturnsVector verifies that Vector.Pop returns a *Vector,
// not a *SubVector.
func TestVectorPopReturnsVector(t *testing.T) {
	v := NewVector(1, 2, 3)
	popped := v.Pop()
	if _, ok := popped.(*Vector); !ok {
		t.Errorf("Pop() returned %T, want *Vector", popped)
	}
}

// TestVectorPopPreservesElements verifies the popped vector has the right
// elements.
func TestVectorPopPreservesElements(t *testing.T) {
	v := NewVector(10, 20, 30)
	popped := v.Pop().(*Vector)
	if popped.Count() != 2 {
		t.Errorf("popped.Count() = %d, want 2", popped.Count())
	}
	if popped.Nth(0) != 10 || popped.Nth(1) != 20 {
		t.Errorf("popped elements = [%v, %v], want [10, 20]",
			popped.Nth(0), popped.Nth(1))
	}
}

// TestVectorPopConjRoundtrip verifies that popping and conj-ing restores the
// original vector contents.
func TestVectorPopConjRoundtrip(t *testing.T) {
	v := NewVector(1, 2, 3)
	popped := v.Pop().(*Vector)
	restored := popped.Cons(3).(*Vector)
	if restored.Count() != v.Count() {
		t.Errorf("restored.Count() = %d, want %d", restored.Count(), v.Count())
	}
	for i := 0; i < v.Count(); i++ {
		if restored.Nth(i) != v.Nth(i) {
			t.Errorf("element %d: got %v, want %v", i, restored.Nth(i), v.Nth(i))
		}
	}
}

// TestVectorPopSingleElement verifies that popping a single-element vector
// returns emptyVector.
func TestVectorPopSingleElement(t *testing.T) {
	v := NewVector(42)
	popped := v.Pop()
	if popped.(*Vector).Count() != 0 {
		t.Errorf("popping single-element vector: Count() = %d, want 0",
			popped.(*Vector).Count())
	}
}

func TestEmptyVectorConstructionDoesNotAllocate(t *testing.T) {
	if got := testing.AllocsPerRun(1_000, func() {
		if NewVector().Count() != 0 {
			panic("non-empty vector")
		}
	}); got != 0 {
		t.Fatalf("NewVector() allocated %v objects, want 0", got)
	}
}

// TestSubVectorPopNoNesting verifies that repeated SubVector.Pop() does not
// increase nesting depth — the result wraps the underlying vector, not self.
func TestSubVectorPopNoNesting(t *testing.T) {
	base := NewVector(1, 2, 3, 4, 5)
	// Create a SubVector manually
	sv := NewSubVector(nil, base, 0, 5) // [1 2 3 4 5]
	// Pop once → should wrap `base` at depth 1, not `sv`
	popped1 := sv.Pop().(*SubVector)
	if popped1.v != base {
		t.Errorf("after first pop: inner vector is %T, want *Vector (base)", popped1.v)
	}
	// Pop again → should still wrap `base`, not `popped1`
	popped2 := popped1.Pop().(*SubVector)
	if popped2.v != base {
		t.Errorf("after second pop: inner vector is %T, want *Vector (base)", popped2.v)
	}
	// Verify correct element counts
	if popped1.Count() != 4 {
		t.Errorf("after 1 pop: Count() = %d, want 4", popped1.Count())
	}
	if popped2.Count() != 3 {
		t.Errorf("after 2 pops: Count() = %d, want 3", popped2.Count())
	}
}

// TestNthStringASCII verifies Nth on ASCII strings returns correct chars.
func TestNthStringASCII(t *testing.T) {
	s := "hello"
	for i, want := range s {
		got, ok := Nth(s, i)
		if !ok {
			t.Errorf("Nth(%q, %d): ok=false, want true", s, i)
			continue
		}
		if got != NewChar(want) {
			t.Errorf("Nth(%q, %d) = %v, want %v", s, i, got, NewChar(want))
		}
	}
}

// TestNthStringMultibyte verifies Nth on multi-byte UTF-8 strings returns
// runes (not bytes).
func TestNthStringMultibyte(t *testing.T) {
	s := "héllo" // 'é' is 2 bytes (U+00E9)
	wantRunes := []rune(s)
	for i, want := range wantRunes {
		got, ok := Nth(s, i)
		if !ok {
			t.Errorf("Nth(%q, %d): ok=false, want true", s, i)
			continue
		}
		if got != NewChar(want) {
			t.Errorf("Nth(%q, %d) = %v, want %v", s, i, got, NewChar(want))
		}
	}
}

// TestNthStringOutOfBounds verifies Nth returns (nil, false) for out-of-range
// indices.
func TestNthStringOutOfBounds(t *testing.T) {
	s := "hi"
	cases := []int{-1, 2, 100}
	for _, n := range cases {
		got, ok := Nth(s, n)
		if ok || got != nil {
			t.Errorf("Nth(%q, %d) = (%v, %v), want (nil, false)", s, n, got, ok)
		}
	}
}

// TestCharAtASCII verifies CharAt on ASCII strings.
func TestCharAtASCII(t *testing.T) {
	s := "world"
	for i, want := range s {
		got := CharAt(s, i)
		if got != NewChar(want) {
			t.Errorf("CharAt(%q, %d) = %v, want %v", s, i, got, NewChar(want))
		}
	}
}

// TestCharAtMultibyte verifies CharAt on multi-byte UTF-8 strings.
func TestCharAtMultibyte(t *testing.T) {
	s := "café"
	wantRunes := []rune(s)
	for i, want := range wantRunes {
		got := CharAt(s, i)
		if got != NewChar(want) {
			t.Errorf("CharAt(%q, %d) = %v, want %v", s, i, got, NewChar(want))
		}
	}
}

// TestCharAtOutOfBoundsPanics verifies CharAt panics for out-of-range index.
func TestCharAtOutOfBoundsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("CharAt out-of-bounds did not panic")
		}
	}()
	CharAt("hi", 5)
}

// TestGoSliceStringOneArg verifies GoSlice string fast-path with one index.
func TestGoSliceStringOneArg(t *testing.T) {
	s := "hello"
	got := GoSlice(s, 2)
	if got != "llo" {
		t.Errorf("GoSlice(%q, 2) = %q, want %q", s, got, "llo")
	}
}

// TestGoSliceStringTwoArgs verifies GoSlice string fast-path with two indices.
func TestGoSliceStringTwoArgs(t *testing.T) {
	s := "hello"
	got := GoSlice(s, 1, 4)
	if got != "ell" {
		t.Errorf("GoSlice(%q, 1, 4) = %q, want %q", s, got, "ell")
	}
}

// TestGoSliceStringNilIndices verifies GoSlice handles nil indices as
// 0 / len(s).
func TestGoSliceStringNilIndices(t *testing.T) {
	s := "hello"
	got := GoSlice(s, nil, nil)
	if got != "hello" {
		t.Errorf("GoSlice(%q, nil, nil) = %q, want %q", s, got, "hello")
	}
}

// TestGoSliceStringMatchesReflect verifies the string fast-path produces the
// same result as the reflect path would for a []byte with same content.
func TestGoSliceStringMatchesReflect(t *testing.T) {
	cases := []struct {
		s    string
		i, j int
	}{
		{"abcdef", 0, 6},
		{"abcdef", 2, 5},
		{"abcdef", 0, 0},
		{"abcdef", 3, 3},
	}
	for _, c := range cases {
		got := GoSlice(c.s, c.i, c.j)
		want := c.s[c.i:c.j]
		if got != want {
			t.Errorf("GoSlice(%q, %d, %d) = %q, want %q", c.s, c.i, c.j, got, want)
		}
	}
}
