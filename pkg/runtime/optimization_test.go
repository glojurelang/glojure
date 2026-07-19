package runtime

import (
	"math"
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
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
	if got := testing.AllocsPerRun(1_000, func() {
		_, _ = lang.FieldOrMethod(RT, "nth")
	}); got != 0 {
		t.Fatalf("cached RT.Nth resolution allocated %v objects per call, want 0", got)
	}
}

func TestNumericLoopRegionSpecializesFloat64AndFallsBack(t *testing.T) {
	x := lang.NewSymbol("x")
	local := &ast.Node{Op: ast.OpLocal, Sub: &ast.LocalNode{Name: x}}
	constant := &ast.Node{Op: ast.OpConst, Sub: &ast.ConstNode{Value: float64(1.5)}}
	call := &ast.HostCallNode{
		Target: &ast.Node{
			Op:  ast.OpConst,
			Sub: &ast.ConstNode{Value: lang.Numbers},
		},
		Method:         lang.NewSymbol("add"),
		Args:           []*ast.Node{local, constant},
		ResolvedMethod: lang.FnFunc2(func(a, b any) any { return lang.Numbers.Add(a, b) }),
	}
	compiler := threadedEvalCompiler{
		typedLoop: true,
		localSlots: map[*lang.Symbol]localSlot{
			x: {
				index:       0,
				kind:        loopLocalSlot,
				numericKind: float64NumericKind,
			},
		},
	}
	evaluator := compiler.compileNumericRegion(call, func(*environment) (interface{}, error) {
		return "fallback", nil
	})
	frame := loopFrame{}
	env := &environment{loopFrame: &frame}

	frame.args[0] = float64(2.5)
	if got, err := evaluator(env); err != nil || got != float64(4) {
		t.Fatalf("float region result = (%v, %v), want (4, nil)", got, err)
	}

	frame.args[0] = int64(2)
	if got, err := evaluator(env); err != nil || got != "fallback" {
		t.Fatalf("mismatched region result = (%v, %v), want (fallback, nil)", got, err)
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

func TestNativeCoreApplyPreservesLeadingArguments(t *testing.T) {
	capture := lang.FnFunc(func(args ...any) any {
		return lang.NewVector(args...)
	})
	apply := nativeCoreApply{}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{
			name: "no leading arguments",
			got:  apply.Invoke2(capture, lang.NewVector(int64(1), int64(2))),
			want: lang.NewVector(int64(1), int64(2)),
		},
		{
			name: "one leading argument",
			got:  apply.Invoke3(capture, int64(1), lang.NewVector(int64(2), int64(3))),
			want: lang.NewVector(int64(1), int64(2), int64(3)),
		},
		{
			name: "two leading arguments",
			got:  apply.Invoke4(capture, int64(1), int64(2), lang.NewVector(int64(3), int64(4))),
			want: lang.NewVector(int64(1), int64(2), int64(3), int64(4)),
		},
		{
			name: "variadic leading arguments",
			got: apply.Invoke(
				capture,
				int64(1),
				int64(2),
				int64(3),
				int64(4),
				lang.NewVector(int64(5), int64(6)),
			),
			want: lang.NewVector(int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)),
		},
		{
			name: "apply-to",
			got: apply.ApplyTo(lang.NewList(
				capture,
				int64(1),
				int64(2),
				lang.NewVector(int64(3), int64(4)),
			)),
			want: lang.NewVector(int64(1), int64(2), int64(3), int64(4)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !lang.Equals(test.got, test.want) {
				t.Fatalf("apply result = %v, want %v", test.got, test.want)
			}
		})
	}
}

func TestNativeCoreUpdateInPreservesCollectionAndVarSemantics(t *testing.T) {
	testNS := lang.FindOrCreateNamespace(lang.NewSymbol("runtime.native-core-test"))
	assocCalls := 0
	applyCalls := 0
	assocVar := lang.NewVarWithRoot(
		testNS,
		lang.NewSymbol("assoc"),
		lang.FnFunc3(func(m, key, value any) any {
			assocCalls++
			return lang.Assoc(m, key, value)
		}),
	)
	applyVar := lang.NewVarWithRoot(
		testNS,
		lang.NewSymbol("apply"),
		lang.FnFunc3(func(fn, value, args any) any {
			applyCalls++
			return lang.ApplySeq(fn, lang.NewCons(value, lang.Seq(args)))
		}),
	)
	updateIn := nativeCoreUpdateIn{assoc: assocVar, apply: applyVar}

	outer := lang.NewKeyword("outer")
	inner := lang.NewKeyword("inner")
	original := lang.NewMap(outer, lang.NewMap(inner, int64(2)))
	add := lang.FnFunc2(func(a, b any) any {
		return lang.Numbers.Add(a, b)
	})

	updated := updateIn.Invoke4(
		original,
		lang.NewVector(outer, inner),
		add,
		int64(3),
	)
	if got := lang.Get(lang.Get(updated, outer), inner); got != int64(5) {
		t.Fatalf("updated value = %v, want 5", got)
	}
	if got := lang.Get(lang.Get(original, outer), inner); got != int64(2) {
		t.Fatalf("original value changed to %v", got)
	}
	if assocCalls != 2 {
		t.Fatalf("assoc Var called %d times, want 2", assocCalls)
	}
	if applyCalls != 1 {
		t.Fatalf("apply Var called %d times, want 1", applyCalls)
	}

	created := updateIn.Invoke3(
		nil,
		lang.NewList(outer, inner),
		lang.FnFunc1(func(value any) any {
			if value != nil {
				t.Fatalf("missing leaf value = %v, want nil", value)
			}
			return int64(7)
		}),
	)
	if got := lang.Get(lang.Get(created, outer), inner); got != int64(7) {
		t.Fatalf("created value = %v, want 7", got)
	}
}
