package interpreterbench

import (
	"testing"

	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

var benchmarkResult any

func BenchmarkConstantArithmetic(b *testing.B) {
	runtime.ReadEval(`
		(defn benchmark-interpreter-configured-step [value]
		  (+ value
		     (* (+ 2 3)
		        (- 11 4))))

		(defn benchmark-interpreter-constant-arithmetic-run []
		  (loop [i 0
		         total 0]
		    (if (= i 1000000)
		      total
		      (recur (inc i)
		             (benchmark-interpreter-configured-step total)))))`)
	ns := lang.GlobalEnv.CurrentNamespace()
	run := ns.FindInternedVar(
		lang.NewSymbol("benchmark-interpreter-constant-arithmetic-run"),
	).Get().(lang.IFn)
	if got := run.Invoke(); got != int64(35000000) {
		b.Fatalf("run = %v, want 35000000", got)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkResult = run.Invoke()
	}
}
