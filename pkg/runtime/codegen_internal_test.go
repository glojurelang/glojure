//go:build !glj_aot_runtime

package runtime

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func TestDirectHostMethod(t *testing.T) {
	target := &ast.Node{
		Op: ast.OpConst,
		Sub: &ast.ConstNode{
			Value: lang.Numbers,
		},
	}

	if got, ok := directHostMethod(target, "multiply", 2); !ok || got != "Multiply" {
		t.Fatalf("directHostMethod(multiply) = %q, %v", got, ok)
	}
	if got, ok := directHostMethod(target, "UncheckedIntDivide", 2); ok {
		t.Fatalf("typed method unexpectedly resolved directly as %q", got)
	}
	if got, ok := directHostMethod(target, "multiply", 1); ok {
		t.Fatalf("wrong arity unexpectedly resolved directly as %q", got)
	}
}

func TestDirectHostCallConvertsIntegerArguments(t *testing.T) {
	target := &ast.Node{
		Op: ast.OpConst,
		Sub: &ast.ConstNode{
			Value: RT,
		},
	}

	method, args, ok := directHostCall(
		target,
		"Nth",
		[]string{"collection", "index"},
	)
	if !ok || method != "Nth" {
		t.Fatalf("directHostCall(Nth) = %q, %v, %v", method, args, ok)
	}
	if got, want := args[0], "collection"; got != want {
		t.Fatalf("collection argument = %q, want %q", got, want)
	}
	if got, want := args[1], "lang.IntCast(index)"; got != want {
		t.Fatalf("index argument = %q, want %q", got, want)
	}
}

func TestLoadedNamespacesUseFreshRuntimeState(t *testing.T) {
	core := lang.FindOrCreateNamespace(lang.NewSymbol("clojure.core"))
	loadedLibs := core.Intern(lang.NewSymbol("*loaded-libs*"))

	initializer, ok := runtimeStateInitializer(loadedLibs)
	if !ok {
		t.Fatal("*loaded-libs* does not have a runtime-state initializer")
	}
	if want := "lang.NewRef(lang.NewSet())"; initializer != want {
		t.Fatalf("*loaded-libs* initializer = %q, want %q", initializer, want)
	}
}

func TestRuntimeFunctionMeta(t *testing.T) {
	foo := lang.NewKeyword("foo")
	bar := lang.NewKeyword("bar")
	retTag := lang.NewKeyword("rettag")

	if got := runtimeFunctionMeta(nil); got != nil {
		t.Fatalf("nil metadata became %v", got)
	}
	if got := runtimeFunctionMeta(lang.NewMap(retTag, nil)); got != nil {
		t.Fatalf("compiler-only metadata became %v", got)
	}

	explicit := lang.NewMap(foo, bar)
	if got := runtimeFunctionMeta(explicit); !lang.Equals(got, explicit) {
		t.Fatalf("explicit metadata became %v, want %v", got, explicit)
	}
	if got := runtimeFunctionMeta(lang.NewMap(retTag, nil, foo, bar)); !lang.Equals(got, explicit) {
		t.Fatalf("mixed metadata became %v, want %v", got, explicit)
	}
}

func TestAnalyzeInt64AOTRecursiveFunction(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-analysis"))
	vr := ns.Intern(lang.NewSymbol("fib"))
	n := lang.NewSymbol("n")
	localN := aotTestLocal(n)

	body := aotTestIf(
		aotTestNumbersCall("Lte", localN, aotTestInt(1)),
		localN,
		aotTestNumbersCall(
			"Add",
			aotTestInvoke(vr, aotTestNumbersCall("Minus", localN, aotTestInt(1))),
			aotTestInvoke(vr, aotTestNumbersCall("Minus", localN, aotTestInt(2))),
		),
	)
	method := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body:       body,
	}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 recursive function was not specialized")
	}
	if !analysis.usesSelf {
		t.Fatal("recursive function did not request a root-version guard")
	}

	floatBody := aotTestIf(
		aotTestNumbersCall("Lte", localN, aotTestInt(1)),
		aotTestConst(float64(1)),
		localN,
	)
	method.Body = floatBody
	if analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	); analysis != nil {
		t.Fatal("mixed float function unexpectedly received an int64 specialization")
	}
}

func TestAnalyzeInt64AOTLoop(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-loop-analysis"))
	vr := ns.Intern(lang.NewSymbol("sum-loop"))
	i := lang.NewSymbol("i")
	sum := lang.NewSymbol("sum")
	loopID := lang.NewSymbol("loop-id")

	inc := aotTestNumbersCall("Inc", aotTestLocal(i))
	add := aotTestNumbersCall("Add", aotTestLocal(sum), aotTestLocal(i))
	recur := ast.MakeNode(ast.OpRecur, nil)
	recur.Sub = &ast.RecurNode{
		LoopID: loopID,
		Exprs: []*ast.Node{
			inc,
			add,
		},
	}
	loop := ast.MakeNode(ast.OpLoop, nil)
	loop.Sub = &ast.LetNode{
		LoopID: loopID,
		Bindings: []*ast.Node{
			aotTestBinding(i, aotTestInt(0)),
			aotTestBinding(sum, aotTestInt(0)),
		},
		Body: aotTestIf(
			aotTestNumbersCall("Lt", aotTestLocal(i), aotTestInt(10)),
			recur,
			aotTestLocal(sum),
		),
	}
	method := &ast.FnMethodNode{Body: loop}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 loop was not specialized")
	}
	if analysis.usesSelf {
		t.Fatal("non-recursive loop unnecessarily requested a root-version guard")
	}
	analysis.proveSafeOperations(method)
	if !analysis.uncheckedHostCalls[inc.Sub.(*ast.HostCallNode)] {
		t.Fatal("bounded induction increment retained an overflow check")
	}
	if !analysis.uncheckedHostCalls[add.Sub.(*ast.HostCallNode)] {
		t.Fatal("bounded accumulator addition retained an overflow check")
	}
}

func TestInt64AOTRangeProofRetainsPossibleOverflow(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-overflow-proof"))
	vr := ns.Intern(lang.NewSymbol("unsafe-loop"))
	i := lang.NewSymbol("i")
	sum := lang.NewSymbol("sum")
	loopID := lang.NewSymbol("loop-id")

	inc := aotTestNumbersCall("Inc", aotTestLocal(i))
	add := aotTestNumbersCall("Add", aotTestLocal(sum), aotTestLocal(i))
	recur := ast.MakeNode(ast.OpRecur, nil)
	recur.Sub = &ast.RecurNode{
		LoopID: loopID,
		Exprs:  []*ast.Node{inc, add},
	}
	loop := ast.MakeNode(ast.OpLoop, nil)
	loop.Sub = &ast.LetNode{
		LoopID: loopID,
		Bindings: []*ast.Node{
			aotTestBinding(i, aotTestInt(1)),
			aotTestBinding(sum, aotTestInt(math.MaxInt64)),
		},
		Body: aotTestIf(
			aotTestNumbersCall("Lt", aotTestLocal(i), aotTestInt(2)),
			recur,
			aotTestLocal(sum),
		),
	}
	method := &ast.FnMethodNode{Body: loop}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 loop was not specialized")
	}
	analysis.proveSafeOperations(method)
	if !analysis.uncheckedHostCalls[inc.Sub.(*ast.HostCallNode)] {
		t.Fatal("safe induction increment retained an overflow check")
	}
	if analysis.uncheckedHostCalls[add.Sub.(*ast.HostCallNode)] &&
		!analysis.guardInt32Loops[loop.Sub.(*ast.LetNode)] {
		t.Fatal("possibly overflowing accumulator addition lost its check without a range guard")
	}
}

func TestInt64AOTSpeculatesOnlyBehindParameterGuard(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-guarded-proof"))
	vr := ns.Intern(lang.NewSymbol("double"))
	n := lang.NewSymbol("n")
	add := aotTestNumbersCall("Add", aotTestLocal(n), aotTestLocal(n))
	method := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body:       add,
	}
	analysis := analyzeInt64AOTFunction(
		&aotSpecializationTarget{vr: vr},
		method,
		nil,
	)
	if analysis == nil {
		t.Fatal("int64 function was not specialized")
	}
	analysis.proveSafeOperations(method)
	if !analysis.uncheckedHostCalls[add.Sub.(*ast.HostCallNode)] {
		t.Fatal("signed-32 addition retained an overflow check")
	}
	if !analysis.guardInt32Params {
		t.Fatal("speculative addition omitted its parameter range guard")
	}
}

func TestInt64AOTFallbackGuardEmission(t *testing.T) {
	var generated bytes.Buffer
	generator := NewGenerator(&generated)
	generator.writeInt32AOTFallbackGuards([]string{"left", "right"})

	source := generated.String()
	for _, name := range []string{"left", "right"} {
		guard := "if " + name + " < -2147483647 || " +
			name + " > 2147483647 {"
		if !strings.Contains(source, guard) {
			t.Fatalf("generated source omitted %q:\n%s", guard, source)
		}
	}
	if got := strings.Count(source, "return 0, false"); got != 2 {
		t.Fatalf("generated source has %d fallbacks, want 2:\n%s", got, source)
	}
}

func TestSnapshotAOTReferencesUsesCompactExclusions(t *testing.T) {
	source := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.snapshot-source"))
	for _, name := range []string{"first", "second", "excluded"} {
		source.Intern(lang.NewSymbol(name))
	}
	refs := []aotReferredVar{
		{symName: "first", srcNS: source.Name().String(), srcSym: "first"},
		{symName: "second", srcNS: source.Name().String(), srcSym: "second"},
	}

	snapshot, exclusions := snapshotAOTReferences(source.Name().String(), refs)
	if !snapshot {
		t.Fatal("dense reference set did not use a shared snapshot")
	}
	if len(exclusions) != 1 || exclusions[0] != "excluded" {
		t.Fatalf("snapshot exclusions = %v, want [excluded]", exclusions)
	}
}

func TestAnalyzeInt64AOTAcrossVarCall(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.int64-cross-call"))
	calleeVar := ns.Intern(lang.NewSymbol("callee"))
	callerVar := ns.Intern(lang.NewSymbol("caller"))
	n := lang.NewSymbol("n")

	calleeTarget := &aotSpecializationTarget{vr: calleeVar, arity: 1}
	calleeMethod := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body: aotTestNumbersCall(
			"Add",
			aotTestLocal(n),
			aotTestInt(1),
		),
	}
	targets := map[*lang.Var]*aotSpecializationTarget{
		calleeVar: calleeTarget,
		callerVar: {vr: callerVar, arity: 1},
	}

	callerMethod := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(n, nil)},
		FixedArity: 1,
		Body:       aotTestInvoke(calleeVar, aotTestLocal(n)),
	}
	if analysis := analyzeInt64AOTFunction(
		targets[callerVar],
		callerMethod,
		targets,
	); analysis != nil {
		t.Fatal("caller specialized before its callee had a primitive path")
	}

	calleeTarget.int64Analysis = analyzeInt64AOTFunction(
		calleeTarget,
		calleeMethod,
		targets,
	)
	if calleeTarget.int64Analysis == nil {
		t.Fatal("callee did not receive a primitive path")
	}
	if analysis := analyzeInt64AOTFunction(
		targets[callerVar],
		callerMethod,
		targets,
	); analysis == nil {
		t.Fatal("caller did not specialize after its callee")
	}
}

func TestAnalyzeFloat64AOTMixedLoopAndCrossCall(t *testing.T) {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("codegen.float64-analysis"))
	polynomialVar := ns.Intern(lang.NewSymbol("polynomial"))
	runVar := ns.Intern(lang.NewSymbol("run"))
	x := lang.NewSymbol("x")
	i := lang.NewSymbol("i")
	total := lang.NewSymbol("total")
	loopID := lang.NewSymbol("float-loop")

	polynomialTarget := &aotSpecializationTarget{vr: polynomialVar, arity: 1}
	runTarget := &aotSpecializationTarget{vr: runVar, arity: 0}
	targets := map[*lang.Var]*aotSpecializationTarget{
		polynomialVar: polynomialTarget,
		runVar:        runTarget,
	}
	polynomialMethod := &ast.FnMethodNode{
		Params:     []*ast.Node{aotTestBinding(x, nil)},
		FixedArity: 1,
		Body: aotTestNumbersCall(
			"Add",
			aotTestNumbersCall("Multiply", aotTestLocal(x), aotTestLocal(x)),
			aotTestConst(float64(0.5)),
		),
	}
	polynomialTarget.float64Analysis = analyzeFloat64AOTFunction(
		polynomialTarget,
		polynomialMethod,
		targets,
	)
	if polynomialTarget.float64Analysis == nil {
		t.Fatal("float64 callee was not specialized")
	}

	recur := ast.MakeNode(ast.OpRecur, nil)
	recur.Sub = &ast.RecurNode{
		LoopID: loopID,
		Exprs: []*ast.Node{
			aotTestNumbersCall("Inc", aotTestLocal(i)),
			aotTestNumbersCall(
				"Add",
				aotTestLocal(total),
				aotTestInvoke(polynomialVar, aotTestConst(float64(1.5))),
			),
		},
	}
	loop := ast.MakeNode(ast.OpLoop, nil)
	loop.Sub = &ast.LetNode{
		LoopID: loopID,
		Bindings: []*ast.Node{
			aotTestBinding(i, aotTestInt(0)),
			aotTestBinding(total, aotTestConst(float64(0))),
		},
		Body: aotTestIf(
			aotTestInvoke(
				lang.NSCore.Intern(lang.NewSymbol("=")),
				aotTestLocal(i),
				aotTestInt(10),
			),
			aotTestLocal(total),
			recur,
		),
	}
	runMethod := &ast.FnMethodNode{Body: loop}
	if analysis := analyzeFloat64AOTFunction(
		runTarget,
		runMethod,
		targets,
	); analysis == nil {
		t.Fatal("mixed int64/float64 loop was not specialized")
	}

	analyzer := newFloat64AOTAnalyzer(
		&float64AOTAnalysis{target: runTarget},
		targets,
	)
	mixedEquality := aotTestInvoke(
		lang.NSCore.Intern(lang.NewSymbol("=")),
		aotTestInt(9007199254740993),
		aotTestConst(float64(9007199254740992)),
	)
	if typ := analyzer.exprType(mixedEquality, nil); typ != invalidAOTPrimitive {
		t.Fatalf("mixed numeric equality received unsafe primitive type %v", typ)
	}
}

func TestAnalyzeReducePipeline(t *testing.T) {
	core := lang.NSCore
	coreVar := func(name string) *lang.Var {
		return core.Intern(lang.NewSymbol(name))
	}
	rangeCall := aotTestInvoke(coreVar("range"), aotTestInt(100))
	filterCall := aotTestInvoke(
		coreVar("filter"),
		aotTestVar(coreVar("odd?")),
		rangeCall,
	)
	mapCall := aotTestInvoke(
		coreVar("map"),
		aotTestVar(coreVar("inc")),
		filterCall,
	)
	reduce := aotTestInvoke(
		coreVar("reduce"),
		aotTestVar(coreVar("+")),
		aotTestInt(0),
		mapCall,
	).Sub.(*ast.InvokeNode)

	plan := analyzeReducePipeline(reduce)
	if plan == nil {
		t.Fatal("safe integer range pipeline was not fused")
	}
	want := []ReducePipelineTransformKind{
		ReducePipelineFilterOdd,
		ReducePipelineMapInc,
	}
	if len(plan.transforms) != len(want) {
		t.Fatalf("transform count = %d, want %d", len(plan.transforms), len(want))
	}
	for i, transform := range plan.transforms {
		if transform.kind != want[i] {
			t.Fatalf("transform %d = %v, want %v", i, transform.kind, want[i])
		}
	}

	rangeCall.Sub.(*ast.InvokeNode).Args[0] = aotTestLocal(lang.NewSymbol("n"))
	if plan := analyzeReducePipeline(reduce); plan != nil {
		t.Fatal("pipeline with an unproven range bound was fused")
	}
}

func aotTestBinding(name *lang.Symbol, init *ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpBinding, nil)
	node.Sub = &ast.BindingNode{Name: name, Init: init}
	return node
}

func aotTestLocal(name *lang.Symbol) *ast.Node {
	node := ast.MakeNode(ast.OpLocal, nil)
	node.Sub = &ast.LocalNode{Name: name}
	return node
}

func aotTestConst(value any) *ast.Node {
	node := ast.MakeNode(ast.OpConst, nil)
	node.Sub = &ast.ConstNode{Value: value}
	return node
}

func aotTestInt(value int64) *ast.Node {
	return aotTestConst(value)
}

func aotTestNumbersCall(name string, args ...*ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpHostCall, nil)
	node.Sub = &ast.HostCallNode{
		Target:         aotTestConst(lang.Numbers),
		Method:         lang.NewSymbol(name),
		Args:           args,
		ResolvedMethod: true,
	}
	return node
}

func aotTestInvoke(vr *lang.Var, args ...*ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpInvoke, nil)
	node.Sub = &ast.InvokeNode{Fn: aotTestVar(vr), Args: args}
	return node
}

func aotTestVar(vr *lang.Var) *ast.Node {
	node := ast.MakeNode(ast.OpVar, nil)
	node.Sub = &ast.VarNode{Var: vr}
	return node
}

func aotTestIf(test, then, otherwise *ast.Node) *ast.Node {
	node := ast.MakeNode(ast.OpIf, nil)
	node.Sub = &ast.IfNode{Test: test, Then: then, Else: otherwise}
	return node
}
