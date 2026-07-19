package runtime

import (
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
