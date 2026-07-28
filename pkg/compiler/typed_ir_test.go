package compiler

import (
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func TestTypedIRRecordsResolvedCallRepresentation(t *testing.T) {
	call := typedIRConst(func(any) int { return 1 })
	invoke := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn:   call,
			Args: []*ast.Node{typedIRConst(int64(42))},
		},
	}

	facts := BuildTypedIR(invoke).Facts(invoke)
	if facts.Type.Kind != IRInt || facts.Type.GoType != reflect.TypeFor[int]() {
		t.Fatalf("resolved call type = %#v, want concrete Go int", facts.Type)
	}
}

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

func TestTypedIRMarksLiteralIndexedCollectionCallbacksForInlining(
	t *testing.T,
) {
	accumulator := lang.NewSymbol("accumulator")
	value := lang.NewSymbol("value")
	reducer := &ast.Node{
		Op: ast.OpFn,
		Sub: &ast.FnNode{
			Methods: []*ast.Node{{
				Op: ast.OpFnMethod,
				Sub: &ast.FnMethodNode{
					Params: []*ast.Node{
						typedIRBinding(accumulator, nil),
						typedIRBinding(value, nil),
					},
					FixedArity: 2,
					Body: typedIRNumbersCall(
						"Add",
						typedIRLocal(accumulator),
						typedIRLocal(value),
					),
				},
			}},
		},
	}
	reduce := typedIRInvoke(
		typedIRCoreVar("reduce"),
		reducer,
		typedIRConst(int64(0)),
		typedIRConst(nil),
	)
	plan := BuildTypedIR(reduce).Facts(reduce).Pipeline
	if plan == nil || plan.Consumer != IRPipelineReduce ||
		plan.Lowering != IRPipelineInlineIndexed {
		t.Fatalf("literal reduce plan = %#v", plan)
	}

	item := lang.NewSymbol("item")
	callback := &ast.Node{
		Op: ast.OpFn,
		Sub: &ast.FnNode{
			Methods: []*ast.Node{{
				Op: ast.OpFnMethod,
				Sub: &ast.FnMethodNode{
					Params:     []*ast.Node{typedIRBinding(item, nil)},
					FixedArity: 1,
					Body:       typedIRLocal(item),
				},
			}},
		},
	}
	mapv := typedIRInvoke(
		typedIRCoreVar("mapv"),
		callback,
		typedIRConst(nil),
	)
	plan = BuildTypedIR(mapv).Facts(mapv).Pipeline
	if plan == nil || plan.Consumer != IRPipelineMapv ||
		plan.Lowering != IRPipelineInlineIndexed {
		t.Fatalf("literal mapv plan = %#v", plan)
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

func TestTypedIRProvesStableLoopBindingTypes(t *testing.T) {
	index := lang.NewSymbol("index")
	loopID := lang.NewSymbol("stable-loop")
	binding := typedIRBinding(index, typedIRConst(int64(0)))
	next := typedIRNumbersCall("Inc", typedIRLocal(index))
	recur := &ast.Node{
		Op: ast.OpRecur,
		Sub: &ast.RecurNode{
			LoopID: loopID,
			Exprs:  []*ast.Node{next},
		},
	}
	loop := &ast.Node{
		Op: ast.OpLoop,
		Sub: &ast.LetNode{
			Bindings: []*ast.Node{binding},
			Body:     recur,
			LoopID:   loopID,
		},
	}

	ir := BuildTypedIR(loop)
	if got := ir.Facts(next).Type.Kind; got != IRInt {
		t.Fatalf("increment type = %v, want IRInt", got)
	}
	if got := ir.BindingFacts(binding).StableType.Kind; got != IRInt {
		t.Fatalf("stable binding type = %v, want IRInt", got)
	}

	recur.Sub.(*ast.RecurNode).Exprs[0] = typedIRConst("changed")
	ir = BuildTypedIR(loop)
	if got := ir.BindingFacts(binding).StableType.Kind; got != IRDynamic {
		t.Fatalf("type-changing binding = %v, want IRDynamic", got)
	}
}

func TestTypedIRJoinsUniformCaseResultTypes(t *testing.T) {
	caseNode := &ast.Node{
		Op: ast.OpCase,
		Sub: &ast.CaseNode{
			Test:    typedIRConst(int64(0)),
			Default: typedIRConst(int64(3)),
			Entries: []ast.CaseEntry{
				{Key: 0, ResultExpr: typedIRConst(int64(1))},
				{Key: 1, ResultExpr: typedIRConst(int64(2))},
			},
		},
	}
	if got := BuildTypedIR(caseNode).Facts(caseNode).Type.Kind; got != IRInt {
		t.Fatalf("uniform case result type = %v, want int", got)
	}

	throwNode := &ast.Node{
		Op: ast.OpThrow,
		Sub: &ast.ThrowNode{
			Exception: typedIRConst("no matching clause"),
		},
	}
	caseNode.Sub.(*ast.CaseNode).Default = throwNode
	ir := BuildTypedIR(caseNode)
	if got := ir.Facts(caseNode).Type.Kind; got != IRInt {
		t.Fatalf("case with throwing default type = %v, want int", got)
	}
	if !ir.Facts(throwNode).NeverReturns {
		t.Fatal("throw was not marked as never returning")
	}

	caseNode.Sub.(*ast.CaseNode).Default = typedIRConst("mixed")
	if got := BuildTypedIR(caseNode).Facts(caseNode).Type.Kind; got != IRDynamic {
		t.Fatalf("mixed case result type = %v, want dynamic", got)
	}
}

func TestTypedIRProvesGeneralLoopCarriedMapOwnership(t *testing.T) {
	index := lang.NewSymbol("index")
	counts := lang.NewSymbol("counts")
	key := lang.NewSymbol("key")
	loopID := lang.NewSymbol("loop")

	indexBinding := typedIRBinding(index, typedIRConst(int64(0)))
	mapBinding := typedIRBinding(counts, &ast.Node{
		Op:  ast.OpMap,
		Sub: &ast.MapNode{},
	})
	keyBinding := typedIRBinding(key, typedIRLocal(index))
	get := &ast.Node{
		Op: ast.OpHostCall,
		Sub: &ast.HostCallNode{
			Target: &ast.Node{
				Op: ast.OpConst,
				Sub: &ast.ConstNode{
					HostSymbol: lang.NewSymbol(
						"github.com:glojurelang:glojure:pkg:runtime.RT",
					),
				},
			},
			Method: lang.NewSymbol("Get"),
			Args: []*ast.Node{
				typedIRLocal(counts),
				typedIRLocal(key),
				typedIRConst(int64(0)),
			},
		},
	}
	update := &ast.Node{
		Op: ast.OpAssoc,
		Sub: &ast.AssocNode{
			Target: typedIRLocal(counts),
			Entries: []ast.AssocEntry{{
				Key: typedIRLocal(key),
				Val: get,
			}},
		},
	}
	recur := &ast.Node{
		Op: ast.OpRecur,
		Sub: &ast.RecurNode{
			LoopID: loopID,
			Exprs: []*ast.Node{
				typedIRLocal(index),
				update,
			},
		},
	}
	body := &ast.Node{
		Op: ast.OpIf,
		Sub: &ast.IfNode{
			Test: typedIRConst(true),
			Then: typedIRLocal(counts),
			Else: &ast.Node{
				Op: ast.OpLet,
				Sub: &ast.LetNode{
					Bindings: []*ast.Node{keyBinding},
					Body:     recur,
				},
			},
		},
	}
	loop := &ast.Node{
		Op: ast.OpLoop,
		Sub: &ast.LetNode{
			Bindings: []*ast.Node{indexBinding, mapBinding},
			Body:     body,
			LoopID:   loopID,
		},
	}

	ir := BuildTypedIR(loop)
	if facts := ir.BindingFacts(mapBinding); !facts.OwnedMap ||
		facts.Escape != IRDoesNotEscape ||
		facts.OwnedMapMode != IROwnedMapTransient {
		t.Fatalf("map binding facts = %#v, want owned non-escaping map", facts)
	}
	if !ir.Facts(update).OwnedMapAssoc {
		t.Fatal("loop-carried assoc was not marked as an owned update")
	}
	if !ir.Facts(get).OwnedMapGet {
		t.Fatal("owned map lookup was not marked")
	}
	exit := body.Sub.(*ast.IfNode).Then
	if !ir.Facts(exit).PersistOwnedMap {
		t.Fatal("terminal map result was not marked for persistence")
	}

	// A dynamically typed initializer uses the guarded copy-on-write
	// representation instead of assuming that the value is a map.
	mapBinding.Sub.(*ast.BindingNode).Init =
		typedIRLocal(lang.NewSymbol("input"))
	ir = BuildTypedIR(loop)
	if facts := ir.BindingFacts(mapBinding); !facts.OwnedMap ||
		facts.OwnedMapMode != IROwnedMapAdaptive {
		t.Fatalf("dynamic map binding facts = %#v, want adaptive ownership", facts)
	}
	if got := ir.Facts(update).OwnedMapMode; got != IROwnedMapAdaptive {
		t.Fatalf("dynamic assoc mode = %v, want adaptive", got)
	}
	if got := ir.Facts(exit).OwnedMapMode; got != IROwnedMapAdaptive {
		t.Fatalf("dynamic exit mode = %v, want adaptive", got)
	}

	// Passing the map to an unknown operation makes its identity observable and
	// must disable the ownership optimization.
	unknown := lang.FindOrCreateNamespace(lang.NewSymbol("typed-ir-test")).
		Intern(lang.NewSymbol("observe"))
	body.Sub.(*ast.IfNode).Test = typedIRInvoke(unknown, typedIRLocal(counts))
	ir = BuildTypedIR(loop)
	if facts := ir.BindingFacts(mapBinding); facts.OwnedMap {
		t.Fatalf("escaping map was marked owned: %#v", facts)
	}

	// Failed analyses must not leave operation facts behind when a safe read
	// precedes the escaping use.
	body.Sub.(*ast.IfNode).Test = typedIRConst(true)
	elseLet := body.Sub.(*ast.IfNode).Else.Sub.(*ast.LetNode)
	elseLet.Body = &ast.Node{
		Op: ast.OpDo,
		Sub: &ast.DoNode{
			Statements: []*ast.Node{
				typedIRInvoke(unknown, typedIRLocal(counts)),
			},
			Ret: recur,
		},
	}
	ir = BuildTypedIR(loop)
	if facts := ir.BindingFacts(mapBinding); facts.OwnedMap {
		t.Fatalf("late-escaping map was marked owned: %#v", facts)
	}
	if ir.Facts(get).OwnedMapGet {
		t.Fatal("failed owned-map analysis left a lookup fact behind")
	}
}

func TestTypedIRProvesOwnedLoopStringParts(t *testing.T) {
	remaining := lang.NewSymbol("remaining")
	parts := lang.NewSymbol("parts")
	loopID := lang.NewSymbol("string-parts-loop")
	partsBinding := typedIRBinding(
		parts,
		&ast.Node{Op: ast.OpVector, Sub: &ast.VectorNode{}},
	)
	appendPart := typedIRInvoke(
		typedIRCoreVar("conj"),
		typedIRLocal(parts),
		typedIRConst("x"),
	)
	recur := &ast.Node{
		Op: ast.OpRecur,
		Sub: &ast.RecurNode{
			LoopID: loopID,
			Exprs: []*ast.Node{
				typedIRNumbersCall(
					"Dec",
					typedIRLocal(remaining),
				),
				appendPart,
			},
		},
	}
	finish := typedIRInvoke(
		typedIRCoreVar("apply"),
		typedIRVar(typedIRCoreVar("str")),
		typedIRLocal(parts),
	)
	body := &ast.Node{
		Op: ast.OpIf,
		Sub: &ast.IfNode{
			Test: typedIRConst(true),
			Then: finish,
			Else: recur,
		},
	}
	loop := &ast.Node{
		Op: ast.OpLoop,
		Sub: &ast.LetNode{
			Bindings: []*ast.Node{
				typedIRBinding(remaining, typedIRConst(int64(2))),
				partsBinding,
			},
			Body:   body,
			LoopID: loopID,
		},
	}

	ir := BuildTypedIR(loop)
	facts := ir.BindingFacts(partsBinding)
	if !facts.OwnedStringParts || facts.Escape != IRDoesNotEscape {
		t.Fatalf("string-parts binding facts = %#v, want owned parts", facts)
	}
	if plan := ir.Facts(appendPart).StringPartsAppend; plan == nil ||
		plan.Parts != parts ||
		plan.Value != appendPart.Sub.(*ast.InvokeNode).Args[1] {
		t.Fatalf("string-parts append plan = %#v", plan)
	}
	if plan := ir.Facts(finish).StringPartsFinish; plan == nil ||
		plan.Parts != parts {
		t.Fatalf("string-parts finish plan = %#v", plan)
	}
	if got := ir.Facts(finish).Type.Kind; got != IRString {
		t.Fatalf("string-parts result type = %v, want string", got)
	}

	// Any observation other than the proven append and apply-str boundary
	// exposes vector semantics and must retain the persistent representation.
	body.Sub.(*ast.IfNode).Test = typedIRInvoke(
		typedIRCoreVar("count"),
		typedIRLocal(parts),
	)
	ir = BuildTypedIR(loop)
	if facts := ir.BindingFacts(partsBinding); facts.OwnedStringParts {
		t.Fatalf("observable parts were marked owned: %#v", facts)
	}
	if ir.Facts(appendPart).StringPartsAppend != nil ||
		ir.Facts(finish).StringPartsFinish != nil {
		t.Fatal("failed string-parts analysis left operation facts behind")
	}
}

func TestTypedIRProvesOwnedMapReduceUpdateChain(t *testing.T) {
	totals := lang.NewSymbol("totals")
	value := lang.NewSymbol("value")
	pathA := &ast.Node{
		Op: ast.OpVector,
		Sub: &ast.VectorNode{
			Items: []*ast.Node{typedIRConst(lang.NewKeyword("a"))},
		},
	}
	pathB := &ast.Node{
		Op: ast.OpVector,
		Sub: &ast.VectorNode{
			Items: []*ast.Node{typedIRConst(lang.NewKeyword("b"))},
		},
	}
	first := typedIRInvoke(
		typedIRCoreVar("update-in"),
		typedIRLocal(totals),
		pathA,
		typedIRVar(typedIRCoreVar("inc")),
	)
	second := typedIRInvoke(
		typedIRCoreVar("update-in"),
		first,
		pathB,
		typedIRVar(typedIRCoreVar("+")),
		typedIRLocal(value),
	)
	reducer := &ast.Node{
		Op: ast.OpFn,
		Sub: &ast.FnNode{
			MaxFixedArity: 2,
			Methods: []*ast.Node{{
				Op: ast.OpFnMethod,
				Sub: &ast.FnMethodNode{
					FixedArity: 2,
					Params: []*ast.Node{
						typedIRBinding(totals, nil),
						typedIRBinding(value, nil),
					},
					Body: second,
				},
			}},
		},
	}
	reduce := typedIRInvoke(
		typedIRCoreVar("reduce"),
		reducer,
		&ast.Node{Op: ast.OpMap, Sub: &ast.MapNode{}},
		&ast.Node{
			Op:  ast.OpLocal,
			Sub: &ast.LocalNode{Name: lang.NewSymbol("values")},
		},
	)

	ir := BuildTypedIR(reduce)
	if plan := ir.Facts(reduce).OwnedMapReduce; plan == nil {
		t.Fatal("owned map reduction was not represented")
	}
	if !ir.Facts(first).OwnedMapUpdateIn ||
		!ir.Facts(second).OwnedMapUpdateIn {
		t.Fatal("owned update-in chain was not marked")
	}

	observer := lang.FindOrCreateNamespace(lang.NewSymbol("typed-ir-test")).
		Intern(lang.NewSymbol("observe-map"))
	method := reducer.Sub.(*ast.FnNode).
		Methods[0].Sub.(*ast.FnMethodNode)
	method.Body = typedIRInvoke(observer, typedIRLocal(totals))
	ir = BuildTypedIR(reduce)
	if plan := ir.Facts(reduce).OwnedMapReduce; plan != nil {
		t.Fatalf("escaping reducer accumulator was marked owned: %#v", plan)
	}
}

func TestTypedIRAppliesGuardedCallSignatures(t *testing.T) {
	vr := lang.FindOrCreateNamespace(lang.NewSymbol("typed-ir-test")).
		Intern(lang.NewSymbol("combine"))
	invoke := typedIRInvoke(
		vr,
		typedIRConst(nil),
		typedIRConst(float64(1.25)),
		typedIRConst(float64(2.5)),
	)
	signature := IRCallSignature{
		Params: []IRType{
			{Kind: IRNil},
			{Kind: IRFloat},
			{Kind: IRFloat},
		},
		Result: IRType{Kind: IRFloat},
	}
	ir := BuildTypedIRWithOptions(invoke, TypedIROptions{
		CallSignatures: map[*lang.Var][]IRCallSignature{
			vr: {signature},
		},
	})
	facts := ir.Facts(invoke)
	if facts.Type.Kind != IRFloat || facts.Signature == nil {
		t.Fatalf("guarded call facts = %#v, want float signature", facts)
	}
	if got := ir.ResolvedCallVars(); len(got) != 1 || got[0] != vr {
		t.Fatalf("resolved call vars = %v, want [%v]", got, vr)
	}
	if got := ir.ResolvedCallCount(); got != 1 {
		t.Fatalf("resolved call count = %d, want 1", got)
	}
}

func TestTypedIRSeedsGuardedParameterTypes(t *testing.T) {
	name := lang.NewSymbol("value")
	param := typedIRBinding(name, nil)
	body := typedIRLocal(name)
	fn := &ast.Node{
		Op: ast.OpFn,
		Sub: &ast.FnNode{
			Methods: []*ast.Node{{
				Op: ast.OpFnMethod,
				Sub: &ast.FnMethodNode{
					Params:     []*ast.Node{param},
					FixedArity: 1,
					Body:       body,
				},
			}},
		},
	}

	if got := BuildTypedIR(fn).Facts(body).Type.Kind; got != IRDynamic {
		t.Fatalf("unguarded parameter type = %v, want dynamic", got)
	}
	ir := BuildTypedIRWithOptions(fn, TypedIROptions{
		ParameterTypes: map[*lang.Symbol]IRType{
			name: {Kind: IRInt},
		},
	})
	if got := ir.Facts(body).Type.Kind; got != IRInt {
		t.Fatalf("guarded parameter type = %v, want int", got)
	}
}

func TestTypedIRReportsCallArgumentTypesAndRepresentationScore(t *testing.T) {
	target := lang.NewVar(
		lang.FindOrCreateNamespace(lang.NewSymbol("typed-ir-test")),
		lang.NewSymbol("target"),
	)
	parameter := lang.NewSymbol("value")
	local := typedIRLocal(parameter)
	one := typedIRConst(int64(1))
	add := &ast.Node{
		Op: ast.OpHostCall,
		Sub: &ast.HostCallNode{
			Target: typedIRConst(lang.Numbers),
			Method: lang.NewSymbol("Add"),
			Args:   []*ast.Node{local, one},
		},
	}
	invoke := &ast.Node{
		Op: ast.OpInvoke,
		Sub: &ast.InvokeNode{
			Fn: &ast.Node{
				Op:  ast.OpVar,
				Sub: &ast.VarNode{Var: target},
			},
			Args: []*ast.Node{add},
		},
	}
	root := &ast.Node{
		Op: ast.OpFn,
		Sub: &ast.FnNode{
			Methods: []*ast.Node{{
				Op: ast.OpFnMethod,
				Sub: &ast.FnMethodNode{
					Params:     []*ast.Node{typedIRBinding(parameter, nil)},
					FixedArity: 1,
					Body:       invoke,
				},
			}},
		},
	}

	base := BuildTypedIR(root)
	optimized := BuildTypedIRWithOptions(root, TypedIROptions{
		ParameterTypes: map[*lang.Symbol]IRType{
			parameter: {Kind: IRInt},
		},
	})
	if optimized.RepresentationScore() <= base.RepresentationScore() {
		t.Fatalf(
			"typed parameter score = %d, want greater than base %d",
			optimized.RepresentationScore(),
			base.RepresentationScore(),
		)
	}
	sites := optimized.DirectCallSites()
	if len(sites) != 1 || sites[0].Var != target ||
		len(sites[0].ArgumentTypes) != 1 ||
		sites[0].ArgumentTypes[0].Kind != IRInt {
		t.Fatalf("direct call sites = %#v, want one int call", sites)
	}
}

func TestAnalyzeFixedVectorResultRequiresParameterLocalComponents(t *testing.T) {
	left := lang.NewSymbol("left")
	right := lang.NewSymbol("right")
	vector := &ast.Node{
		Op: ast.OpVector,
		Sub: &ast.VectorNode{Items: []*ast.Node{
			typedIRLocal(left),
			typedIRLocal(right),
		}},
	}
	root := &ast.Node{
		Op: ast.OpFn,
		Sub: &ast.FnNode{Methods: []*ast.Node{{
			Op: ast.OpFnMethod,
			Sub: &ast.FnMethodNode{
				Params: []*ast.Node{
					typedIRBinding(left, nil),
					typedIRBinding(right, nil),
				},
				FixedArity: 2,
				Body:       vector,
			},
		}}},
	}
	plan := AnalyzeFixedVectorResult(root)
	if plan == nil || plan.Method.FixedArity != 2 ||
		len(plan.Components) != 2 {
		t.Fatalf("fixed vector result plan = %#v", plan)
	}

	vector.Sub.(*ast.VectorNode).Items[1] =
		typedIRLocal(lang.NewSymbol("captured"))
	if got := AnalyzeFixedVectorResult(root); got != nil {
		t.Fatalf("free-local vector result plan = %#v, want nil", got)
	}
}

func typedIRConst(value any) *ast.Node {
	return &ast.Node{
		Op:  ast.OpConst,
		Sub: &ast.ConstNode{Value: value},
	}
}

func typedIRBinding(name *lang.Symbol, init *ast.Node) *ast.Node {
	return &ast.Node{
		Op: ast.OpBinding,
		Sub: &ast.BindingNode{
			Name: name,
			Init: init,
		},
	}
}

func typedIRLocal(name *lang.Symbol) *ast.Node {
	return &ast.Node{
		Op:  ast.OpLocal,
		Sub: &ast.LocalNode{Name: name},
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

func typedIRNumbersCall(method string, args ...*ast.Node) *ast.Node {
	return &ast.Node{
		Op: ast.OpHostCall,
		Sub: &ast.HostCallNode{
			Target: &ast.Node{
				Op: ast.OpConst,
				Sub: &ast.ConstNode{
					Value:      lang.Numbers,
					HostSymbol: lang.NewSymbol("github.com:glojurelang:glojure:pkg:lang.Numbers"),
				},
			},
			Method:         lang.NewSymbol(method),
			Args:           args,
			ResolvedMethod: true,
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
