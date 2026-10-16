//go:build !glj_aot_runtime

package runtime

import (
	"sync"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestCompiledFunctionCapturesLexicalSlots(t *testing.T) {
	for _, code := range []string{
		`(fn [value] (fn [] value))`,
		`(fn [arg] (let [value arg] (fn [] value)))`,
		`(fn [value] (let [value (identity value)] (fn [] value)))`,
	} {
		fn := ReadEval(code).(*Fn)
		if fn.singleEval == nil {
			t.Fatalf("function did not compile: %s", code)
		}
		for _, value := range []any{int64(42), false, nil} {
			closure := fn.Invoke1(value).(lang.IFn)
			fn.Invoke1("other")
			if got := closure.Invoke(); !lang.Equals(got, value) {
				t.Fatalf("%s captured %v, want %v", code, got, value)
			}
		}
	}
}

func TestCompiledLoopCapturesEachIteration(t *testing.T) {
	got := ReadEval(`
      (let [collect (fn [limit]
                      (loop [i 0 result []]
                        (if (< i limit)
                          (recur (inc i) (conj result (fn [] i)))
                          result)))
            closures (collect 4)]
        (mapv (fn [f] (f)) closures))`)
	want := lang.NewVector(int64(0), int64(1), int64(2), int64(3))
	if !lang.Equals(got, want) {
		t.Fatalf("captured loop values = %v, want %v", got, want)
	}
}

func TestCompiledNestedLoopsKeepOuterSlots(t *testing.T) {
	fn := ReadEval(`
      (fn [limit]
        (loop [i 0 total 0]
          (if (< i limit)
            (recur (inc i)
              (+ total
                (loop [j 0 sum 0]
                  (if (< j i)
                    (recur (inc j) (+ sum j))
                    sum))))
            total)))`).(*Fn)
	if fn.singleEval == nil {
		t.Fatal("nested-loop function did not compile")
	}
	if got := fn.Invoke1(int64(5)); got != int64(10) {
		t.Fatalf("nested loops = %v, want 10", got)
	}
}

func TestCompiledVectorPreservesEvaluationOrder(t *testing.T) {
	fn := ReadEval(`(fn [a] [(swap! a conj 1) (swap! a conj 2)])`).(*Fn)
	if fn.singleEval == nil {
		t.Fatal("vector function did not compile")
	}
	value := lang.NewAtom(lang.NewVector())
	got := fn.Invoke1(value)
	want := lang.NewVector(lang.NewVector(int64(1)), lang.NewVector(int64(1), int64(2)))
	if !lang.Equals(got, want) {
		t.Fatalf("vector = %v, want %v", got, want)
	}
}

func TestCompiledFunctionsKeepConcurrentArguments(t *testing.T) {
	makeClosure := ReadEval(`(fn [value] (fn [] value))`).(*Fn)
	add := ReadEval(`(fn [x y] (+ x y))`).(*Fn)
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for i := 0; i < 100; i++ {
				value := int64(worker*100 + i)
				closure := makeClosure.Invoke1(value).(lang.IFn)
				if got := add.Invoke2(value, int64(1)); got != value+1 {
					t.Errorf("sum = %v, want %d", got, value+1)
				}
				if got := closure.Invoke(); got != value {
					t.Errorf("captured value = %v, want %d", got, value)
				}
			}
		}(worker)
	}
	workers.Wait()
}
