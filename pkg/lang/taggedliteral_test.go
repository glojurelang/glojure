package lang

import "testing"

func TestTaggedLiteralLookupAndEquality(t *testing.T) {
	tag := NewSymbol("demo/value")
	literal := NewTaggedLiteral(tag, 42).(*TaggedLiteral)
	if got := literal.ValAt(NewKeyword("tag")); got != tag {
		t.Fatalf(":tag = %v, want %v", got, tag)
	}
	if got := literal.ValAt(NewKeyword("form")); got != 42 {
		t.Fatalf(":form = %v, want 42", got)
	}
	if !literal.Equals(NewTaggedLiteral(tag, 42)) {
		t.Fatal("equal tagged literals did not compare equal")
	}
}
