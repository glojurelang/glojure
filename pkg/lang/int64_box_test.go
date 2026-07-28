package lang

import (
	"math"
	"testing"
)

var (
	boxedArithmeticLeft  any = BoxInt64(1200)
	boxedArithmeticRight any = BoxInt64(2300)
	boxedMultiplyLeft    any = BoxInt64(40)
	boxedMultiplyRight   any = BoxInt64(100)
	boxedArithmeticSink  any
)

func TestBoxInt64PreservesValues(t *testing.T) {
	for _, value := range []int64{
		math.MinInt64,
		minCachedInt64 - 1,
		minCachedInt64,
		0,
		maxCachedInt64,
		maxCachedInt64 + 1,
		math.MaxInt64,
	} {
		if got := BoxInt64(value); got != value {
			t.Errorf("BoxInt64(%d) = %v", value, got)
		}
	}
}

func TestBoxIntPreservesValues(t *testing.T) {
	for _, value := range []int{-129, -128, 0, 4096, 4097} {
		if got := BoxInt(value); got != value {
			t.Errorf("BoxInt(%d) = %v", value, got)
		}
	}
}

func TestBoxIntCachesCommonValues(t *testing.T) {
	var boxed any
	allocs := testing.AllocsPerRun(1000, func() {
		boxed = BoxInt(42)
	})
	if boxed != 42 {
		t.Fatalf("BoxInt(42) = %v", boxed)
	}
	if allocs != 0 {
		t.Fatalf("BoxInt(42) allocated %v times, want zero", allocs)
	}
}

func TestCommonInt64ArithmeticDoesNotAllocate(t *testing.T) {
	if got := testing.AllocsPerRun(1_000, func() {
		boxedArithmeticSink = Numbers.Add(
			boxedArithmeticLeft,
			boxedArithmeticRight,
		)
	}); got != 0 {
		t.Fatalf("cached int64 addition allocated %v objects per call", got)
	}
	if got := testing.AllocsPerRun(1_000, func() {
		boxedArithmeticSink = Numbers.Multiply(
			boxedMultiplyLeft,
			boxedMultiplyRight,
		)
	}); got != 0 {
		t.Fatalf("cached int64 multiplication allocated %v objects per call", got)
	}
}

func BenchmarkCommonInt64Arithmetic(b *testing.B) {
	left := make([]any, 1024)
	right := make([]any, 1024)
	for i := range left {
		left[i] = BoxInt64(int64(i))
		right[i] = BoxInt64(int64(1023 - i))
	}
	var result any
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = Numbers.Add(left[i&1023], right[(i>>10)&1023])
	}
	boxedArithmeticSink = result
}
