package runtime

import (
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativeTransducersComposeInt64Steps(t *testing.T) {
	mapper := lang.NewInt64UnaryFn(
		lang.FnFunc1(func(value any) any {
			return lang.Numbers.Inc(value)
		}),
		func(value int64) int64 {
			return lang.CheckedAddInt64(value, 1)
		},
	)
	predicate := lang.NewInt64PredicateFn(
		lang.FnFunc1(func(value any) any {
			return value.(int64)&1 != 0
		}),
		func(value int64) bool {
			return value&1 != 0
		},
	)

	var reducing any = nativeCoreAdd{}
	reducing = lang.Apply1(NewTakeTransducer(int64(3)), reducing)
	reducing = lang.Apply1(NewFilterTransducer(predicate), reducing)
	reducing = lang.Apply1(NewMapTransducer(mapper), reducing)

	stepper, ok := reducing.(lang.Int64ReductionStepper)
	if !ok {
		t.Fatalf("composed reducer has type %T, want Int64ReductionStepper", reducing)
	}
	result := lang.NewLongRange(0, 100, 1).(lang.Int64StepReducible).
		ReduceInt64Steps(stepper, 0)
	if result != 9 {
		t.Fatalf("primitive transduction = %d, want 9", result)
	}
	if completed := lang.Apply1(reducing, result); completed != int64(9) {
		t.Fatalf("completed result = %v, want 9", completed)
	}
}

func TestNativeMapTransducerPreservesGenericAndVariadicCalls(t *testing.T) {
	mapper := lang.FnFunc(func(args ...any) any {
		parts := make([]string, len(args))
		for index, arg := range args {
			parts[index] = arg.(string)
		}
		return strings.Join(parts, ":")
	})
	appendValue := lang.FnFunc2(func(result, value any) any {
		return result.(string) + value.(string)
	})
	reducer := lang.Apply1(
		NewMapTransducer(mapper),
		appendValue,
	).(lang.IFn)

	if got := reducer.Invoke("", "a", "b", "c"); got != "a:b:c" {
		t.Fatalf("variadic mapped reduction = %v, want a:b:c", got)
	}
}

func TestNativeTakeTransducerRetainsNumericTowerFallback(t *testing.T) {
	reducer := lang.Apply1(
		NewTakeTransducer(float64(2.5)),
		nativeCoreAdd{},
	)
	if _, ok := reducer.(lang.Int64ReductionStepper); ok {
		t.Fatal("floating take limit selected primitive path")
	}
	result := lang.Apply2(reducer, int64(0), int64(1))
	if lang.IsReduced(result) {
		t.Fatal("floating take reduced after first value")
	}
	result = lang.Apply2(reducer, result, int64(2))
	if lang.IsReduced(result) {
		t.Fatal("floating take reduced after second value")
	}
	result = lang.Apply2(reducer, result, int64(3))
	if !lang.IsReduced(result) {
		t.Fatalf("third floating take step = %T(%v), want Reduced", result, result)
	}
	if got := result.(lang.IDeref).Deref(); got != int64(6) {
		t.Fatalf("floating take result = %v, want 6", got)
	}
}

func TestNativeInt64TransductionStepDoesNotAllocate(t *testing.T) {
	mapper := lang.NewInt64UnaryFn(
		lang.FnFunc1(func(value any) any { return value }),
		func(value int64) int64 { return value + 1 },
	)
	predicate := lang.NewInt64PredicateFn(
		lang.FnFunc1(func(value any) any {
			return value.(int64)&1 == 0
		}),
		func(value int64) bool { return value&1 == 0 },
	)
	var reducing any = nativeCoreAdd{}
	reducing = lang.Apply1(NewFilterTransducer(predicate), reducing)
	reducing = lang.Apply1(NewMapTransducer(mapper), reducing)
	stepper := reducing.(lang.Int64ReductionStepper)
	source := lang.NewLongRange(0, 1000, 1).(lang.Int64StepReducible)

	allocations := testing.AllocsPerRun(100, func() {
		if got := source.ReduceInt64Steps(stepper, 0); got != 250500 {
			panic(got)
		}
	})
	if allocations != 0 {
		t.Fatalf("primitive transduction allocated %.2f objects, want 0", allocations)
	}
}
