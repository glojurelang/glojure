package lang

import "testing"

func TestInt64UnaryFnAdapterPreservesDynamicFallback(t *testing.T) {
	fn := NewInt64UnaryFn(
		FnFunc1(func(value any) any {
			if text, ok := value.(string); ok {
				return text + "!"
			}
			return value.(int64) + 1
		}),
		func(value int64) int64 { return value + 1 },
	)

	if got := fn.Invoke1("go"); got != "go!" {
		t.Fatalf("Invoke1 = %v, want go!", got)
	}
	if got := fn.InvokeInt64(41); got != 42 {
		t.Fatalf("InvokeInt64 = %v, want 42", got)
	}
}

func TestInt64PredicateFnAdapterPreservesMetadata(t *testing.T) {
	fn := NewInt64PredicateFn(
		FnFunc1(func(value any) any { return value.(int64)&1 != 0 }),
		func(value int64) bool { return value&1 != 0 },
	)
	meta := NewMap(NewKeyword("source"), "test")
	withMeta := fn.WithMeta(meta).(*Int64PredicateFnAdapter)

	if !withMeta.InvokeInt64Predicate(3) {
		t.Fatal("InvokeInt64Predicate(3) = false, want true")
	}
	if withMeta.Meta() != meta {
		t.Fatalf("Meta = %v, want %v", withMeta.Meta(), meta)
	}
	if fn.Meta() != nil {
		t.Fatalf("original Meta = %v, want nil", fn.Meta())
	}
}
