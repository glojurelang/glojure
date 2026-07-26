package lang

import (
	"reflect"
	"testing"
)

func TestStringCollectionOperationsUseBytes(t *testing.T) {
	s := "aé"
	want := []any{
		NewChar('a'),
		NewChar(rune(0xc3)),
		NewChar(rune(0xa9)),
	}

	if got := Count(s); got != len(s) {
		t.Fatalf("Count(%q) = %d, want %d", s, got, len(s))
	}

	var fromSeq []any
	for seq := Seq(s); seq != nil; seq = seq.Next() {
		fromSeq = append(fromSeq, seq.First())
	}
	if !reflect.DeepEqual(fromSeq, want) {
		t.Fatalf("Seq(%q) = %v, want %v", s, fromSeq, want)
	}

	if got := ToSlice(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("ToSlice(%q) = %v, want %v", s, got, want)
	}
}
