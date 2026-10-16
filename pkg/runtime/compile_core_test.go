//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestCompiledCoreCallsPreserveCalleeEvaluationOrder(t *testing.T) {
	fn := ReadEval(`(fn [change x] (first (change x)))`).(*Fn)
	first := lang.NSCore.FindInternedVar(lang.NewSymbol("first"))
	original := first.Get()
	defer func() { first.BindRoot(original); recordOptimizableCoreRoots(lang.NSCore) }()
	calls := 0
	change := lang.FnFunc1(func(value any) any {
		calls++
		captured := calls
		first.BindRoot(lang.FnFunc1(func(any) any { return int64(captured) }))
		return value
	})
	if got := fn.Invoke2(change, lang.NewVector(int64(42))); got != int64(42) {
		t.Fatalf("callee changed while evaluating argument: %v", got)
	}
	if got := fn.Invoke2(change, lang.NewVector(int64(42))); got != int64(1) {
		t.Fatalf("fallback callee changed while evaluating argument: %v", got)
	}
}

func TestCompiledCoreEqualityPreservesRedefinition(t *testing.T) {
	fn := ReadEval(`(fn [x y] (= x y))`).(*Fn)
	equal := lang.NSCore.FindInternedVar(lang.NewSymbol("="))
	original := equal.Get()
	defer func() { equal.BindRoot(original); recordOptimizableCoreRoots(lang.NSCore) }()
	marker := lang.NewKeyword("redefined")
	equal.BindRoot(lang.FnFunc2(func(any, any) any { return marker }))
	if got := fn.Invoke2(int64(1), int64(1)); got != marker {
		t.Fatalf("redefined equality = %v, want %v", got, marker)
	}
}

func TestCompiledPositiveCountPreservesRedefinition(t *testing.T) {
	fn := ReadEval(`(fn [xs] (> (count xs) 0))`).(*Fn)
	for _, value := range []any{
		nil, []any{}, []any{int64(1)}, []int{1},
		lang.NewVector(), lang.NewVector(int64(1)),
	} {
		if got := fn.Invoke1(value); got != (lang.Count(value) > 0) {
			t.Fatalf("positive count of %T = %v", value, got)
		}
	}
	count := lang.NSCore.FindInternedVar(lang.NewSymbol("count"))
	original := count.Get()
	defer func() { count.BindRoot(original); recordOptimizableCoreRoots(lang.NSCore) }()
	count.BindRoot(lang.FnFunc1(func(any) any { return int64(0) }))
	if got := fn.Invoke1(lang.NewSliceSeq([]any{int64(1)})); got != false {
		t.Fatalf("positive count ignored redefined count: %v", got)
	}
}

func TestCompiledCoreSequenceCalls(t *testing.T) {
	fn := ReadEval(`(fn [xs] [(count xs) (seq xs) (first xs) (next xs) (rest xs)])`).(*Fn)
	for _, value := range []any{nil, lang.NewVector(), lang.NewVector(int64(1), int64(2)), []int{1, 2}} {
		want := lang.NewVector(lang.Count(value), lang.Seq(value), lang.First(value), lang.Next(value), lang.Rest(value))
		if got := fn.Invoke1(value); !lang.Equals(got, want) {
			t.Fatalf("sequence calls on %v = %v, want %v", value, got, want)
		}
	}
}

func TestCompiledCoreCallsRecoverInvalidSequence(t *testing.T) {
	for _, expression := range []string{
		`(fn [x] (first x))`, `(fn [x] (count x))`,
		`(fn [x] (first (identity x)))`,
	} {
		for _, value := range []any{int64(42), &lang.SliceSeq{}} {
			t.Run(fmt.Sprintf("%s/%T", expression, value), func(t *testing.T) {
				fn := ReadEval(expression, WithFilename("compiled-core.glj")).(*Fn)
				defer func() {
					failure := recover()
					if failure == nil || !strings.Contains(fmt.Sprint(failure), "compiled-core.glj") {
						t.Fatalf("missing source information in recovered error: %v", failure)
					}
				}()
				fn.Invoke1(value)
			})
		}
	}
}
