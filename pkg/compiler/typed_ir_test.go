package compiler

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func TestTypedIRFindsFixedGetInAndKeywordMapShapes(t *testing.T) {
	getIn := typedIRCoreVar("get-in")
	target := typedIRConst(nil)
	keyA := typedIRConst(lang.NewKeyword("a"))
	keyB := typedIRConst(lang.NewKeyword("b"))
	path := &ast.Node{
		Op: ast.OpVector,
		Sub: &ast.VectorNode{
			Items: []*ast.Node{keyA, keyB},
		},
	}
	invoke := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn:   typedIRVar(getIn),
			Args: []*ast.Node{target, path},
		},
	}
	m := &ast.Node{
		Op: ast.OpMap,
		Sub: &ast.MapNode{
			Keys: []*ast.Node{keyA, keyB},
			Vals: []*ast.Node{typedIRConst(1), typedIRConst(2)},
		},
	}
	root := &ast.Node{
		Op: ast.OpDo,
		Sub: &ast.DoNode{
			Statements: []*ast.Node{invoke},
			Ret:        m,
		},
	}

	ir := BuildTypedIR(root)
	getInFacts := ir.Facts(invoke)
	if getInFacts.GetIn == nil || len(getInFacts.GetIn.Keys) != 2 {
		t.Fatalf("get-in facts = %#v, want a two-key plan", getInFacts)
	}
	mapFacts := ir.Facts(m)
	if mapFacts.Shape.Kind != IRShapeKeywordMap ||
		mapFacts.Shape.Count != 2 ||
		len(mapFacts.Shape.Keywords) != 2 {
		t.Fatalf("map shape = %#v, want a two-key keyword map", mapFacts.Shape)
	}
}

func TestTypedIRMarksOnlySynchronousLocalAtomsNoEscape(t *testing.T) {
	atom := typedIRCoreVar("atom")
	deref := typedIRCoreVar("deref")
	name := lang.NewSymbol("state")
	initial := typedIRConst(int64(1))
	binding := &ast.Node{
		Op: ast.OpBinding,
		Sub: &ast.BindingNode{
			Name: name,
			Init: &ast.Node{
				Op: ast.OpInvoke,
				Sub: &ast.InvokeNode{
					Fn:   typedIRVar(atom),
					Args: []*ast.Node{initial},
				},
			},
		},
	}
	local := &ast.Node{
		Op:  ast.OpLocal,
		Sub: &ast.LocalNode{Name: name},
	}
	body := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn:   typedIRVar(deref),
			Args: []*ast.Node{local},
		},
	}
	root := &ast.Node{
		Op: ast.OpLet,
		Sub: &ast.LetNode{
			Bindings: []*ast.Node{binding},
			Body:     body,
		},
	}

	ir := BuildTypedIR(root)
	facts := ir.BindingFacts(binding)
	if facts.Escape != IRDoesNotEscape || facts.AtomInit != initial {
		t.Fatalf("binding facts = %#v, want scalar-replaceable atom", facts)
	}

	closure := &ast.Node{
		Op: ast.OpFn,
		Sub: &ast.FnNode{
			Methods: []*ast.Node{{
				Op: ast.OpFnMethod,
				Sub: &ast.FnMethodNode{
					Body: body,
				},
			}},
		},
	}
	root.Sub.(*ast.LetNode).Body = closure
	ir = BuildTypedIR(root)
	if got := ir.BindingFacts(binding).Escape; got != IREscapes {
		t.Fatalf("captured atom escape = %v, want IREscapes", got)
	}
}

func TestTypedIRDescribesGeneralCollectionPipelinesWithoutForcingLowering(
	t *testing.T,
) {
	xs := &ast.Node{
		Op:  ast.OpLocal,
		Sub: &ast.LocalNode{Name: lang.NewSymbol("xs")},
	}
	filter := typedIRInvoke(
		typedIRCoreVar("filter"),
		typedIRVar(typedIRCoreVar("odd?")),
		xs,
	)
	mapv := typedIRInvoke(
		typedIRCoreVar("mapv"),
		typedIRVar(typedIRCoreVar("inc")),
		filter,
	)

	plan := BuildTypedIR(mapv).Facts(mapv).Pipeline
	if plan == nil {
		t.Fatal("mapv/filter pipeline was not represented")
	}
	if plan.Consumer != IRPipelineMapv ||
		plan.Source != xs ||
		plan.Lowering != IRPipelineNoLowering {
		t.Fatalf("mapv plan = %#v", plan)
	}
	if len(plan.Stages) != 2 ||
		plan.Stages[0].Kind != IRPipelineFilter ||
		plan.Stages[0].Primitive != IRPipelineFilterOdd ||
		plan.Stages[1].Kind != IRPipelineMap ||
		plan.Stages[1].Primitive != IRPipelineMapInc {
		t.Fatalf("mapv stages = %#v", plan.Stages)
	}

	custom := &ast.Node{
		Op:  ast.OpLocal,
		Sub: &ast.LocalNode{Name: lang.NewSymbol("f")},
	}
	mapped := typedIRInvoke(typedIRCoreVar("map"), custom, xs)
	into := typedIRInvoke(
		typedIRCoreVar("into"),
		&ast.Node{Op: ast.OpVector, Sub: &ast.VectorNode{}},
		mapped,
	)
	plan = BuildTypedIR(into).Facts(into).Pipeline
	if plan == nil || plan.Consumer != IRPipelineInto ||
		len(plan.Stages) != 1 ||
		plan.Stages[0].Primitive != IRPipelinePrimitiveUnknown ||
		plan.Lowering != IRPipelineNoLowering {
		t.Fatalf("generic into/map plan = %#v", plan)
	}
}

func TestTypedIRPreservesPipelineStageOrder(t *testing.T) {
	source := typedIRInvoke(
		typedIRCoreVar("range"),
		typedIRConst(int64(10)),
	)
	mapped := typedIRInvoke(
		typedIRCoreVar("map"),
		typedIRVar(typedIRCoreVar("inc")),
		source,
	)
	taken := typedIRInvoke(
		typedIRCoreVar("take"),
		typedIRConst(int64(2)),
		mapped,
	)
	filtered := typedIRInvoke(
		typedIRCoreVar("filter"),
		typedIRVar(typedIRCoreVar("odd?")),
		taken,
	)
	reduce := typedIRInvoke(
		typedIRCoreVar("reduce"),
		typedIRVar(typedIRCoreVar("+")),
		typedIRConst(int64(0)),
		filtered,
	)

	plan := BuildTypedIR(reduce).Facts(reduce).Pipeline
	if plan == nil || len(plan.Stages) != 3 {
		t.Fatalf("reduce plan = %#v", plan)
	}
	want := []IRPipelineStageKind{
		IRPipelineMap,
		IRPipelineTake,
		IRPipelineFilter,
	}
	for i, kind := range want {
		if plan.Stages[i].Kind != kind {
			t.Fatalf("stage %d = %v, want %v", i, plan.Stages[i].Kind, kind)
		}
	}
	if plan.Lowering != IRPipelineNoLowering {
		t.Fatal("take below filter was incorrectly marked for lowering")
	}
}

func typedIRConst(value any) *ast.Node {
	return &ast.Node{
		Op:  ast.OpConst,
		Sub: &ast.ConstNode{Value: value},
	}
}

func typedIRInvoke(vr *lang.Var, args ...*ast.Node) *ast.Node {
	return &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn:   typedIRVar(vr),
			Args: args,
		},
	}
}

func typedIRVar(vr *lang.Var) *ast.Node {
	return &ast.Node{
		Op:  ast.OpVar,
		Sub: &ast.VarNode{Var: vr},
	}
}

func typedIRCoreVar(name string) *lang.Var {
	core := lang.FindOrCreateNamespace(lang.NewSymbol("clojure.core"))
	return core.Intern(lang.NewSymbol(name))
}
