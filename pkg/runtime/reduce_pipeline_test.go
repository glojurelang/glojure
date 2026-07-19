package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestReducePipeline(t *testing.T) {
	result := ReduceInt64Pipeline(
		int64(0),
		lang.NewLongRange(0, 10, 1),
		[]ReducePipelineTransformKind{
			ReducePipelineFilterOdd,
			ReducePipelineMapInc,
		},
	)
	if result != int64(30) {
		t.Fatalf("ReducePipeline = %v, want 30", result)
	}
}

func TestDefaultCoreVarGuardTracksRedefinition(t *testing.T) {
	defaultCoreRoots.Lock()
	original := defaultCoreRoots.byVar
	defaultCoreRoots.Unlock()
	defer func() {
		defaultCoreRoots.Lock()
		defaultCoreRoots.byVar = original
		defaultCoreRoots.Unlock()
	}()

	core := lang.FindOrCreateNamespace(lang.NewSymbol("core-root-guard-test"))
	vr := core.InternWithValue(
		lang.NewSymbol("f"),
		lang.FnFunc0(func() interface{} { return nil }),
		true,
	)
	recordDefaultCoreRoots(core, "f")
	if !IsDefaultCoreVar(vr) {
		t.Fatal("freshly recorded core Var was not stable")
	}
	vr.BindRoot(lang.FnFunc0(func() interface{} { return int64(1) }))
	if IsDefaultCoreVar(vr) {
		t.Fatal("redefined core Var retained its default-root marker")
	}
}
