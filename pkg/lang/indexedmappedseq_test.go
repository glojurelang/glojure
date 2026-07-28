package lang

import (
	"strings"
	"testing"
)

func TestIndexedMappedSeqIsLazyAndCachesValues(t *testing.T) {
	calls := 0
	mapped := NewIndexedMappedSeq(
		FnFunc2(func(index, value any) any {
			calls++
			return []any{index, value}
		}),
		NewList("a", "b", "c"),
	)

	if calls != 0 {
		t.Fatalf("mapping ran during construction: %d calls", calls)
	}
	if got := mapped.First().([]any); got[0] != int64(0) || got[1] != "a" {
		t.Fatalf("first indexed value = %v, want [0 a]", got)
	}
	if got := mapped.First().([]any); got[0] != int64(0) || got[1] != "a" {
		t.Fatalf("cached first indexed value = %v, want [0 a]", got)
	}
	if calls != 1 {
		t.Fatalf("first indexed value ran %d times, want 1", calls)
	}

	next := mapped.Next()
	if calls != 1 {
		t.Fatalf("next indexed value was realized eagerly: %d calls", calls)
	}
	if got := next.First().([]any); got[0] != int64(1) || got[1] != "b" {
		t.Fatalf("second indexed value = %v, want [1 b]", got)
	}
	if calls != 2 {
		t.Fatalf("second indexed value ran %d total times, want 2", calls)
	}
	if mapped.Next() != next {
		t.Fatal("indexed mapped sequence did not cache its successor")
	}
}

func TestIndexedMappedSeqAppendsStringsWithoutSequenceNodes(t *testing.T) {
	calls := 0
	mapped := NewIndexedMappedSeq(
		FnFunc2(func(index, value any) any {
			calls++
			return ToString(index) + value.(string)
		}),
		NewList("a", "b", "c"),
	)

	var builder strings.Builder
	mapped.(interface {
		AppendStrings(*strings.Builder)
	}).AppendStrings(&builder)

	if got := builder.String(); got != "0a1b2c" {
		t.Fatalf("appended strings = %q, want 0a1b2c", got)
	}
	if calls != 3 {
		t.Fatalf("mapping ran %d times, want 3", calls)
	}
	if got := mapped.First(); got != "0a" {
		t.Fatalf("cached first value = %v, want 0a", got)
	}
	if calls != 3 {
		t.Fatalf("cached first value ran mapping again: %d calls", calls)
	}
}
