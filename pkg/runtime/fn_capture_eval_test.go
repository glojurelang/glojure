//go:build !glj_aot_runtime

package runtime

import "testing"

func TestFixedArityFunctionPreservesCapturedParameter(t *testing.T) {
	got := ReadEval(`
		(let [make-closure (fn [value] (fn [] value))
		      closure (make-closure 42)]
		  (closure))`)
	if got != int64(42) {
		t.Fatalf("captured parameter = %v, want 42", got)
	}
}
