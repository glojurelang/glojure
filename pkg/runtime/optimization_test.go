package runtime

import (
	"math"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestScopeDefineReplacesEquivalentSymbol(t *testing.T) {
	s := newScope()
	first := lang.NewSymbol("value")
	equivalent := lang.NewSymbol("value")

	s.define(first, int64(1))
	s.define(equivalent, int64(2))

	got, ok := s.lookup(first)
	if !ok || got != int64(2) {
		t.Fatalf("lookup = (%v, %v), want (2, true)", got, ok)
	}
}

func TestRTGetCompatibilityMethod(t *testing.T) {
	key := lang.NewKeyword("key")
	m := lang.NewMap(key, int64(42))
	get, ok := lang.FieldOrMethod(RT, "Get")
	if !ok {
		t.Fatal("RT.Get was not resolved")
	}

	if got := lang.Apply2(get, m, key); got != int64(42) {
		t.Fatalf("RT.Get existing key = %v, want 42", got)
	}
	missing := lang.NewKeyword("missing")
	if got := lang.Apply3(get, m, missing, int64(7)); got != int64(7) {
		t.Fatalf("RT.Get missing key = %v, want 7", got)
	}
}

func TestRTCollectionMethodsResolveDirectly(t *testing.T) {
	nth, ok := lang.FieldOrMethod(RT, "Nth")
	if !ok {
		t.Fatal("RT.Nth was not resolved")
	}
	if _, ok := nth.(lang.FnFunc2); !ok {
		t.Fatalf("RT.Nth resolved to %T, want lang.FnFunc2", nth)
	}
	if got := lang.Apply2(nth, lang.NewVector(int64(7)), int64(0)); got != int64(7) {
		t.Fatalf("RT.Nth result = %v, want 7", got)
	}
}

func TestNativeCoreAddApplyToReducibleSequence(t *testing.T) {
	args := lang.NewLongRange(0, 1_000, 1)

	if got := (nativeCoreAdd{}).ApplyTo(args); got != int64(499_500) {
		t.Fatalf("sum = %v, want 499500", got)
	}
}

func TestNativeCoreAddApplyToPreservesUnaryNegativeZero(t *testing.T) {
	negativeZero := math.Copysign(0, -1)
	args := lang.NewList(negativeZero)

	got := (nativeCoreAdd{}).ApplyTo(args).(float64)
	if !math.Signbit(got) {
		t.Fatalf("unary sum = %v, want negative zero", got)
	}
}

func TestNativeCoreSubtractApplyToReducibleSequence(t *testing.T) {
	args := lang.NewLongRange(0, 1_000, 1)

	if got := (nativeCoreSubtract{}).ApplyTo(args); got != int64(-499_500) {
		t.Fatalf("difference = %v, want -499500", got)
	}
}

func TestNativeCoreSubtractApplyToPreservesArities(t *testing.T) {
	fn := nativeCoreSubtract{}
	if got := fn.ApplyTo(lang.NewList(int64(5))); got != int64(-5) {
		t.Fatalf("unary difference = %v, want -5", got)
	}
	if got := fn.ApplyTo(lang.NewList(int64(9), int64(4))); got != int64(5) {
		t.Fatalf("binary difference = %v, want 5", got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("zero-arity subtraction did not panic")
		}
	}()
	fn.ApplyTo(nil)
}
