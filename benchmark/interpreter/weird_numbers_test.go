package interpreterbench

import (
	_ "embed"
	"math"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

//go:embed weird_numbers.glj
var weirdNumbersSource string

func loadWeirdNumbers(tb testing.TB) lang.IFn {
	tb.Helper()
	lang.PushThreadBindings(lang.NewMap(
		lang.VarCurrentNS, lang.GlobalEnv.CurrentNamespace(),
	))
	tb.Cleanup(lang.PopThreadBindings)
	runtime.ReadEval("(ns benchmark.weird-numbers)")
	ns := lang.GlobalEnv.CurrentNamespace()
	ns.InternWithValue(lang.NewSymbol("sqrt"), lang.FnFunc1(func(n any) any {
		return math.Sqrt(float64(n.(int64)))
	}), true)
	runtime.ReadEval(weirdNumbersSource)
	return benchmarkFn(tb, "weird-numbers")
}

func TestWeirdNumbers(t *testing.T) {
	run := loadWeirdNumbers(t)
	for _, tc := range []struct {
		limit int64
		want  []any
	}{
		{2, nil},
		{70, nil},
		{71, []any{int64(70)}},
		{1000, []any{int64(70), int64(836)}},
		{16500, []any{
			int64(70), int64(836), int64(4030), int64(5830), int64(7192),
			int64(7912), int64(9272), int64(10430), int64(10570), int64(10792),
			int64(10990), int64(11410), int64(11690), int64(12110), int64(12530),
			int64(12670), int64(13370), int64(13510), int64(13790), int64(13930),
			int64(14770), int64(15610), int64(15890), int64(16030), int64(16310),
		}},
	} {
		want := lang.NewVector(tc.want...)
		if got := lang.Apply1(run, tc.limit); !lang.Equals(got, want) {
			t.Fatalf("limit %d: got %v, want %v", tc.limit, got, want)
		}
	}
}

func BenchmarkWeirdNumbers(b *testing.B) {
	run := loadWeirdNumbers(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkResult = lang.Apply1(run, int64(16500))
	}
}

func BenchmarkLoadWeirdNumbers(b *testing.B) {
	loadWeirdNumbers(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkResult = runtime.ReadEval(weirdNumbersSource)
	}
}
