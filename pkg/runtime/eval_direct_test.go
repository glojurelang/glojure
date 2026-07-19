package runtime

import (
	"errors"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestEvalDirectInvokePreservesVarAndQuoteSemantics(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("runtime.direct-eval-test"))
	calls := 0
	vr := lang.InternVar(
		ns,
		lang.NewSymbol("target"),
		lang.FnFunc1(func(value interface{}) interface{} {
			calls++
			return value
		}),
		true,
	)
	form := lang.NewList(
		lang.NewSymbol("target"),
		lang.NewList(lang.NewSymbol("quote"), lang.NewSymbol("value")),
	)
	env := newEnvironment(t.Context(), nil, nil)

	got, ok, err := env.evalDirectInvoke(form, ns)
	if err != nil || !ok {
		t.Fatalf("direct invoke = %v, %v, %v; want success", got, ok, err)
	}
	if want := lang.NewSymbol("value"); !lang.Equals(got, want) {
		t.Fatalf("direct invoke = %v, want %v", got, want)
	}
	if calls != 1 {
		t.Fatalf("direct invoke calls = %d, want 1", calls)
	}

	vr.SetMacro()
	if _, ok, err := env.evalDirectInvoke(form, ns); ok || err != nil {
		t.Fatalf("macro direct invoke = ok %v, err %v; want fallback", ok, err)
	}
}

func TestEvalReturnsSelfEvaluatingValueDirectly(t *testing.T) {
	env := newEnvironment(t.Context(), nil, nil)
	for _, value := range []interface{}{
		nil,
		true,
		int64(42),
		"literal",
		lang.NewKeyword("literal"),
	} {
		got, err := env.Eval(value)
		if err != nil || !lang.Equals(got, value) {
			t.Fatalf("Eval(%v) = %v, %v", value, got, err)
		}
	}
}

func TestEvalDirectInvokeBuildsRuntimeErrorFrame(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("runtime.direct-eval-error-test"))
	lang.InternVar(
		ns,
		lang.NewSymbol("fail"),
		lang.FnFunc0(func() interface{} {
			panic(errors.New("boom"))
		}),
		true,
	)
	form := lang.NewList(lang.NewSymbol("fail"))
	env := newEnvironment(t.Context(), nil, nil)

	_, ok, err := env.evalDirectInvoke(form, ns)
	if !ok {
		t.Fatal("direct invoke unexpectedly fell back")
	}
	var evalErr *RTEvalError
	if !errors.As(err, &evalErr) {
		t.Fatalf("direct invoke error = %T %v, want RTEvalError", err, err)
	}
	if len(evalErr.GLJStack) != 1 {
		t.Fatalf("GLJ stack has %d frames, want 1", len(evalErr.GLJStack))
	}
}
