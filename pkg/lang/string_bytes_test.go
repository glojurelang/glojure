package lang

import (
	"reflect"
	"testing"
)

func TestStringCollectionOperationsUseUnicodeCodePoints(t *testing.T) {
	s := "aé"
	want := []any{
		NewChar('a'),
		NewChar('é'),
	}

	if got := Count(s); got != len(want) {
		t.Fatalf("Count(%q) = %d, want %d", s, got, len(want))
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
