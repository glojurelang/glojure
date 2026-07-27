//go:build !glj_aot_runtime

package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func TestAnalyzeLetGoMapFilterPipeline(t *testing.T) {
	ReadEval(`
		(defn test-map-filter-pipeline-analysis []
		  (reduce + 0
		    (take 100
		      (filter even?
		        (map #(* % %) (range 10000))))))`)
	fn := lang.GlobalEnv.CurrentNamespace().
		FindInternedVar(lang.NewSymbol("test-map-filter-pipeline-analysis")).
		Get().(*Fn)
	body := fn.ASTNode().Sub.(*ast.FnNode).
		Methods[0].Sub.(*ast.FnMethodNode).Body
	reduce := body.Sub.(*ast.DoNode).Ret
	plan := compiler.AnalyzePipeline(reduce)
	if plan == nil || plan.Lowering != compiler.IRPipelineReduceInt64 {
		t.Fatal("let-go map/filter workload was not recognized")
	}
	if plan.TakeLimit != 100 {
		t.Fatalf("take limit = %d, want 100", plan.TakeLimit)
	}
	want := []ReducePipelineTransformKind{
		ReducePipelineMapSquare,
		ReducePipelineFilterEven,
	}
	var transforms []ReducePipelineTransformKind
	for _, stage := range plan.Stages {
		if stage.Kind == compiler.IRPipelineMap ||
			stage.Kind == compiler.IRPipelineFilter {
			transforms = append(transforms, stage.Primitive)
		}
	}
	if len(transforms) != len(want) {
		t.Fatalf("transform count = %d, want %d", len(transforms), len(want))
	}
	for i, transform := range transforms {
		if transform != want[i] {
			t.Fatalf("transform %d = %v, want %v", i, transform, want[i])
		}
	}
}

func TestReducePipelineFallsBackAfterTakeRedefinition(t *testing.T) {
	ReadEval(`
		(defn test-map-filter-take-guard []
		  (reduce + 0
		    (take 100
		      (filter even?
		        (map #(* % %) (range 10000))))))`)
	run := lang.GlobalEnv.CurrentNamespace().
		FindInternedVar(lang.NewSymbol("test-map-filter-take-guard")).
		Get().(lang.IFn)
	take := lang.NSCore.FindInternedVar(lang.NewSymbol("take"))
	original := take.Get()
	take.BindRoot(lang.FnFunc2(func(_, _ any) any {
		return lang.NewVector(int64(7))
	}))
	defer take.BindRoot(original)

	got := run.Invoke()
	if got != int64(7) {
		t.Fatalf("result with redefined take = %v, want 7", got)
	}
}

func TestReducePipelineRejectsTakeBelowFilter(t *testing.T) {
	got := ReadEval(`
		(defn test-map-filter-inner-take []
		  (reduce + 0
		    (filter odd?
		      (take 2
		        (map inc (range 10))))))
		(test-map-filter-inner-take)`)
	if got != int64(1) {
		t.Fatalf("inner take pipeline = %v, want 1", got)
	}
	fn := lang.GlobalEnv.CurrentNamespace().
		FindInternedVar(lang.NewSymbol("test-map-filter-inner-take")).
		Get().(*Fn)
	body := fn.ASTNode().Sub.(*ast.FnNode).
		Methods[0].Sub.(*ast.FnMethodNode).Body
	reduce := body.Sub.(*ast.DoNode).Ret
	if plan := compiler.AnalyzePipeline(reduce); plan != nil &&
		plan.Lowering == compiler.IRPipelineReduceInt64 {
		t.Fatal("pipeline moved an inner take past an outer filter")
	}
}
