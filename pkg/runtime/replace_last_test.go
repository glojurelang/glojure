package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestReplaceLastPlanPreservesPopBeforeValueEvaluation(t *testing.T) {
	valueEvaluated := false
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("empty vector did not reject pop")
			}
		}()
		plan := PrepareReplaceLast(lang.NewVector())
		valueEvaluated = true
		_ = plan.Finish(int64(1))
	}()
	if valueEvaluated {
		t.Fatal("replacement value was evaluated before pop failed")
	}
}

func TestReplaceLastPlanPreservesNonVectorStackSemantics(t *testing.T) {
	original := lang.NewList(int64(1), int64(2))
	plan := PrepareReplaceLast(original)
	result := plan.Finish(int64(3))
	if got := lang.PrintString(result); got != "(3 2)" {
		t.Fatalf("fused list result = %s, want (3 2)", got)
	}
}
