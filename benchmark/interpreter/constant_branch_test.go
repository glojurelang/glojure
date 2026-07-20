package interpreterbench

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

func BenchmarkConstantBranch(b *testing.B) {
	runtime.ReadEval(`
		(defn benchmark-interpreter-constant-branch-run []
		  (loop [i 0
		         total 0]
		    (if (< i 10000000)
		      (recur (inc i)
		             (if (< (+ 1 2) 4)
		               (inc total)
		               (+ total 2)))
		      total)))`)
	ns := lang.GlobalEnv.CurrentNamespace()
	run := ns.FindInternedVar(
		lang.NewSymbol("benchmark-interpreter-constant-branch-run"),
	).Get().(lang.IFn)
	if got := run.Invoke(); got != int64(10000000) {
		b.Fatalf("run = %v, want 10000000", got)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkResult = run.Invoke()
	}
}
