//go:build !glj_aot_runtime

package runtime

import (
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
)

func TestAnalyzeDirectLetGoMapFilterPipeline(t *testing.T) {
	form := readDirectPipelineForm(t, `
		(reduce + 0
		  (take 100
		    (filter even?
		      (map #(* % %) (range 10000)))))`)
	plan, ok := analyzeDirectInt64ReducePipeline(
		form,
		lang.GlobalEnv.CurrentNamespace(),
	)
	if !ok {
		t.Fatal("let-go map/filter workload was not recognized before AST analysis")
	}
	if plan.takeLimit != 100 {
		t.Fatalf("take limit = %d, want 100", plan.takeLimit)
	}
	want := []ReducePipelineTransformKind{
		ReducePipelineMapSquare,
		ReducePipelineFilterEven,
	}
	if len(plan.transforms) != len(want) {
		t.Fatalf("transform count = %d, want %d", len(plan.transforms), len(want))
	}
	for i, transform := range plan.transforms {
		if transform != want[i] {
			t.Fatalf("transform %d = %v, want %v", i, transform, want[i])
		}
	}
}

func TestDirectReducePipelineFallsBackAfterTakeRedefinition(t *testing.T) {
	form := readDirectPipelineForm(t, `
		(reduce + 0
		  (take 100
		    (filter even?
		      (map #(* % %) (range 10000)))))`)
	take := lang.NSCore.FindInternedVar(lang.NewSymbol("take"))
	original := take.Get()
	take.BindRoot(lang.FnFunc2(func(_, _ any) any {
		return lang.NewVector(int64(7))
	}))
	defer take.BindRoot(original)

	if _, ok := analyzeDirectInt64ReducePipeline(
		form,
		lang.GlobalEnv.CurrentNamespace(),
	); ok {
		t.Fatal("pipeline with redefined take retained the direct path")
	}
	got, err := lang.GlobalEnv.Eval(form)
	if err != nil {
		t.Fatal(err)
	}
	if got != int64(7) {
		t.Fatalf("result with redefined take = %v, want 7", got)
	}
}

func TestDirectReducePipelineRejectsTakeBelowFilter(t *testing.T) {
	form := readDirectPipelineForm(t, `
		(reduce + 0
		  (filter odd?
		    (take 2
		      (map inc (range 10)))))`)
	if _, ok := analyzeDirectInt64ReducePipeline(
		form,
		lang.GlobalEnv.CurrentNamespace(),
	); ok {
		t.Fatal("pipeline moved an inner take past an outer filter")
	}
}

func readDirectPipelineForm(t *testing.T, source string) interface{} {
	t.Helper()
	rdr := reader.New(
		strings.NewReader(source),
		reader.WithGetCurrentNS(lang.GlobalEnv.CurrentNamespace),
	)
	form, err := rdr.ReadOne()
	if err != nil {
		t.Fatal(err)
	}
	return form
}
