package interpreterbench

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

const (
	letGoMapFilterExpression = `
		(reduce + 0
		  (take 100
		    (filter even?
		      (map #(* % %) (range 10000)))))`
	letGoMapFilterSource = `
		(defn benchmark-let-go-map-filter-run []
		  ` + letGoMapFilterExpression + `)`
)

func BenchmarkLetGoMapFilter(b *testing.B) {
	runtime.ReadEval(letGoMapFilterSource)
	run := benchmarkFn(b, "benchmark-let-go-map-filter-run")
	if got := run.Invoke(); got != int64(1_313_400) {
		b.Fatalf("run = %v, want 1313400", got)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkResult = run.Invoke()
	}
}

func BenchmarkLoadLetGoMapFilter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkResult = runtime.ReadEval(letGoMapFilterExpression)
	}
}

func BenchmarkLetGoTak(b *testing.B) {
	runtime.ReadEval(`
		(defn benchmark-let-go-tak [x y z]
		  (if (< y x)
		    (benchmark-let-go-tak
		      (benchmark-let-go-tak (dec x) y z)
		      (benchmark-let-go-tak (dec y) z x)
		      (benchmark-let-go-tak (dec z) x y))
		    z))

		(defn benchmark-let-go-tak-run []
		  (benchmark-let-go-tak 30 22 12))`)
	run := benchmarkFn(b, "benchmark-let-go-tak-run")
	if got := run.Invoke(); got != int64(13) {
		b.Fatalf("run = %v, want 13", got)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkResult = run.Invoke()
	}
}

func benchmarkFn(tb testing.TB, name string) lang.IFn {
	tb.Helper()
	return lang.GlobalEnv.CurrentNamespace().
		FindInternedVar(lang.NewSymbol(name)).
		Get().(lang.IFn)
}
