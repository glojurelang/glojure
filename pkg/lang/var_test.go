package lang

import (
	"testing"
)

func TestIsMacroFalseForNonMacroVar(t *testing.T) {
	ns := FindOrCreateNamespace(NewSymbol("test.var"))
	v := InternVarReplaceRoot(ns, NewSymbol("not-a-macro"), "value")
	if v.IsMacro() {
		t.Error("IsMacro() = true for non-macro var")
	}
	// Second call should use cache.
	if v.IsMacro() {
		t.Error("IsMacro() = true on cached call")
	}
}

func TestIsMacroTrueForMacroVar(t *testing.T) {
	ns := FindOrCreateNamespace(NewSymbol("test.var"))
	v := InternVarReplaceRoot(ns, NewSymbol("is-a-macro"), "value")
	v.SetMacro()
	if !v.IsMacro() {
		t.Error("IsMacro() = false after SetMacro()")
	}
	// Second call should use cache.
	if !v.IsMacro() {
		t.Error("IsMacro() = false on cached call after SetMacro()")
	}
}

func TestIsMacroCacheInvalidatedBySetMeta(t *testing.T) {
	ns := FindOrCreateNamespace(NewSymbol("test.var"))
	v := InternVarReplaceRoot(ns, NewSymbol("meta-test"), "value")

	// Prime the cache as non-macro.
	if v.IsMacro() {
		t.Fatal("unexpected macro")
	}

	// Set macro via SetMeta — should invalidate cache.
	v.SetMeta(v.Meta().Assoc(KWMacro, true).(IPersistentMap))
	if !v.IsMacro() {
		t.Error("IsMacro() = false after SetMeta with :macro true")
	}
}

func TestIsMacroCacheInvalidatedByAlterMeta(t *testing.T) {
	ns := FindOrCreateNamespace(NewSymbol("test.var"))
	v := InternVarReplaceRoot(ns, NewSymbol("alter-meta-test"), "value")

	// Prime cache as non-macro.
	if v.IsMacro() {
		t.Fatal("unexpected macro")
	}

	// AlterMeta calls SetMeta, which should invalidate cache.
	assocFn := FnFunc(func(args ...any) any {
		m := args[0].(IPersistentMap)
		return m.Assoc(args[1], args[2])
	})
	v.AlterMeta(assocFn, NewCons(KWMacro, NewCons(true, nil)))
	if !v.IsMacro() {
		t.Error("IsMacro() = false after AlterMeta with :macro true")
	}
}
