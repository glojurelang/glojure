package lang

import (
	"reflect"
	"testing"
)

func TestMappedSeqIsLazyAndCachesValues(t *testing.T) {
	calls := 0
	mapped := NewMappedSeq(FnFunc1(func(value any) any {
		calls++
		return value.(int) * 2
	}), NewList(1, 2, 3))

	if calls != 0 {
		t.Fatalf("mapping ran during construction: %d calls", calls)
	}
	if mapped.(*mappedSeq).IsRealized() {
		t.Fatal("mapped sequence reported realized before access")
	}
	if got := mapped.First(); got != 2 {
		t.Fatalf("first mapped value = %v, want 2", got)
	}
	if got := mapped.First(); got != 2 {
		t.Fatalf("cached first mapped value = %v, want 2", got)
	}
	if calls != 1 {
		t.Fatalf("first mapped value ran %d times, want 1", calls)
	}

	next := mapped.Next()
	if calls != 1 {
		t.Fatalf("next mapped value was realized eagerly: %d calls", calls)
	}
	if got := next.First(); got != 4 {
		t.Fatalf("second mapped value = %v, want 4", got)
	}
	if calls != 2 {
		t.Fatalf("second mapped value ran %d total times, want 2", calls)
	}
}

func TestMappedSeqConvertsDirectlyToStringSlice(t *testing.T) {
	calls := 0
	mapped := NewMappedSeq(FnFunc1(func(value any) any {
		calls++
		return value.(string) + "!"
	}), NewVector("a", "b", "c").Seq())

	got := asStringSlice(mapped)
	if want := []string{"a!", "b!", "c!"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mapped strings = %v, want %v", got, want)
	}
	if calls != 3 {
		t.Fatalf("mapping ran %d times, want 3", calls)
	}
	if got := mapped.First(); got != "a!" {
		t.Fatalf("cached first mapped value = %v, want a!", got)
	}
	if calls != 3 {
		t.Fatalf("cached first value ran mapping again: %d calls", calls)
	}
}
