//go:build !glj_aot_runtime

package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestUnaryFalseGuardPreservesBothBranches(t *testing.T) {
	for _, value := range []any{int64(3), int64(5), float64(3), int64(-3)} {
		fn := ReadEval(`(let [number 12] (fn [n] (if (= 0 (rem number n)) n :missing)))`).(*Fn)
		if fn.singleGuard == nil {
			t.Fatal("integer predicate was not guarded")
		}
		want := fn.Invoke(value)
		if got := fn.Invoke1(value); !lang.Equals(got, want) {
			t.Fatalf("argument %v: got %v, want %v", value, got, want)
		}
	}
}

func TestUnaryFalseGuardFallsBackOnZeroDivisor(t *testing.T) {
	fn := ReadEval(`(fn [n] (when (= 0 (rem 12 n)) n))`).(*Fn)
	if fn.singleGuard == nil {
		t.Fatal("integer predicate was not guarded")
	}
	if fn.singleGuard.reject(int64(0)) {
		t.Fatal("zero divisor took the fast path")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("zero divisor did not raise an error")
		}
	}()
	fn.Invoke1(int64(0))
}

func TestUnaryFalseGuardRetainsBranchEffects(t *testing.T) {
	got := ReadEval(`
      (let [calls (atom 0)
            f (fn [n] (if (= 0 (rem n 2)) (swap! calls inc) nil))]
        [(f 3) (f 4) @calls])`)
	want := lang.NewVector(nil, int64(1), int64(1))
	if !lang.Equals(got, want) {
		t.Fatalf("branch effects = %v, want %v", got, want)
	}
}

func TestUnaryFalseGuardPreservesRedefinedCoreFunctions(t *testing.T) {
	functions := []*Fn{
		ReadEval(`(fn [n] (if (= 0 (rem 12 n)) n :missing))`).(*Fn),
		ReadEval(`(fn [n] (if (= 0 (rem 12 n)) n :missing))`).(*Fn),
	}
	for i, name := range []string{"=", "rem"} {
		t.Run(name, func(t *testing.T) {
			fn := functions[i]
			if fn.singleGuard == nil {
				t.Fatal("integer predicate was not guarded")
			}
			variable := lang.NSCore.FindInternedVar(lang.NewSymbol(name))
			original := variable.Get()
			defer func() {
				variable.BindRoot(original)
				recordOptimizableCoreRoots(lang.NSCore)
			}()
			variable.BindRoot(lang.FnFunc2(func(a, b any) any {
				if name == "=" {
					return true
				}
				return int64(0)
			}))
			if got := fn.Invoke1(int64(5)); got != int64(5) {
				t.Fatalf("redefined %s returned %v, want 5", name, got)
			}
		})
	}
}

func TestNilCountGuardPreservesEmptyAndNonemptyInputs(t *testing.T) {
	fn := ReadEval(`(fn [value xs] (when (> (count xs) 0) value))`).(*Fn)
	if fn.singleNilGuard == nil {
		t.Fatal("empty-input branch was not guarded")
	}
	if got := fn.Invoke2(int64(42), nil); got != nil {
		t.Fatalf("nil input = %v", got)
	}
	if got := fn.Invoke2(int64(42), lang.NewList(1)); got != int64(42) {
		t.Fatalf("nonempty input = %v", got)
	}
	count := lang.NSCore.FindInternedVar(lang.NewSymbol("count"))
	original := count.Get()
	defer func() {
		count.BindRoot(original)
		recordOptimizableCoreRoots(lang.NSCore)
	}()
	calls := 0
	count.BindRoot(lang.FnFunc1(func(any) any { calls++; return int64(1) }))
	if got := fn.Invoke2(int64(42), nil); got != int64(42) || calls != 1 {
		t.Fatalf("redefined count: result %v, calls %d, want 42 and 1", got, calls)
	}
}
