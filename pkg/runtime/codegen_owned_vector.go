//go:build !glj_aot_runtime

package runtime

import (
	"math/bits"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

// ownedVectorAOTAnalysis describes a synchronous update region whose vector
// identity cannot escape. The generated wrapper validates and recursively
// copies the required vector shape, while direct-linked helpers carry that
// private representation through calls and collection callbacks.
type ownedVectorAOTAnalysis struct {
	target        *aotSpecializationTarget
	paramDepths   []int
	indexedParams uint32
	result        ownedVectorAOTValue
	values        map[*ast.Node]ownedVectorAOTValue
	nths          map[*ast.Node]int
	assocs        map[*ast.Node]bool
	assocIns      map[*ast.Node]ownedVectorAOTAssocIn
	calls         map[*ast.Node]*aotSpecializationTarget
	reduces       map[*ast.Node]*ownedVectorAOTReduce
	mapvs         map[*ast.Node]*ownedVectorAOTMapv
	mutated       uint32
}

type ownedVectorAOTValue struct {
	depth         int
	origins       uint32
	parameters    uint32
	linearIndexed bool
	control       bool
	forbidden     bool
}

type ownedVectorAOTAssocIn struct {
	path []*ast.Node
	copy bool
}

type ownedVectorAOTReduce struct {
	method *ast.FnMethodNode
}

type ownedVectorAOTMapv struct {
	method *ast.FnMethodNode
}

type ownedVectorAOTAnalyzer struct {
	analysis *ownedVectorAOTAnalysis
	targets  map[*lang.Var]*aotSpecializationTarget
	valid    bool
}

func analyzeOwnedVectorAOTFunction(
	target *aotSpecializationTarget,
	method *ast.FnMethodNode,
	targets map[*lang.Var]*aotSpecializationTarget,
) *ownedVectorAOTAnalysis {
	if target == nil || method == nil || method.IsVariadic ||
		method.FixedArity < 1 || method.FixedArity > 31 {
		return nil
	}

	// Depth zero is dynamic. Candidate vector depths are derived from the
	// deepest vector operation or already-proved callee signature in the body,
	// rather than from a workload-specific nesting limit.
	maxDepth := ownedVectorAOTMaxDepth(method.Body, targets)
	if maxDepth == 0 {
		return nil
	}
	combinations := 1
	for range method.FixedArity {
		combinations *= maxDepth + 1
		if combinations > 1<<16 {
			return nil
		}
	}
	var best *ownedVectorAOTAnalysis
	for encoded := 1; encoded < combinations; encoded++ {
		paramDepths := make([]int, method.FixedArity)
		value := encoded
		var paramMask uint32
		for index := range paramDepths {
			paramDepths[index] = value % (maxDepth + 1)
			value /= maxDepth + 1
			if paramDepths[index] > 0 {
				paramMask |= uint32(1) << index
			}
		}
		analysis := &ownedVectorAOTAnalysis{
			target:      target,
			paramDepths: paramDepths,
			values:      make(map[*ast.Node]ownedVectorAOTValue),
			nths:        make(map[*ast.Node]int),
			assocs:      make(map[*ast.Node]bool),
			assocIns:    make(map[*ast.Node]ownedVectorAOTAssocIn),
			calls:       make(map[*ast.Node]*aotSpecializationTarget),
			reduces:     make(map[*ast.Node]*ownedVectorAOTReduce),
			mapvs:       make(map[*ast.Node]*ownedVectorAOTMapv),
		}
		analyzer := &ownedVectorAOTAnalyzer{
			analysis: analysis,
			targets:  targets,
			valid:    true,
		}
		locals := make(map[*lang.Symbol]ownedVectorAOTValue, method.FixedArity)
		for index, param := range method.Params {
			depth := paramDepths[index]
			paramValue := ownedVectorAOTValue{
				depth:      depth,
				parameters: uint32(1) << index,
			}
			if depth > 0 {
				paramValue.origins = uint32(1) << index
			}
			locals[param.Sub.(*ast.BindingNode).Name] = paramValue
		}
		analysis.result = analyzer.expr(method.Body, locals)
		if !analyzer.valid || analysis.result.depth == 0 ||
			analysis.result.control || analysis.result.forbidden ||
			!ownedVectorAOTResultMatchesParams(analysis) ||
			analysis.mutated&paramMask != paramMask {
			continue
		}
		if best == nil ||
			bits.OnesCount32(paramMask) >
				bits.OnesCount32(ownedVectorAOTParamMask(best)) ||
			bits.OnesCount32(paramMask) ==
				bits.OnesCount32(ownedVectorAOTParamMask(best)) &&
				ownedVectorAOTDepthScore(analysis) >
					ownedVectorAOTDepthScore(best) {
			best = analysis
		}
	}
	return best
}

func ownedVectorAOTMaxDepth(
	node *ast.Node,
	targets map[*lang.Var]*aotSpecializationTarget,
) int {
	if node == nil {
		return 0
	}
	depth := 0
	if node.Op == ast.OpInvoke {
		invoke := node.Sub.(*ast.InvokeNode)
		if isCoreInvoke(invoke, "assoc-in") && len(invoke.Args) >= 2 &&
			invoke.Args[1] != nil && invoke.Args[1].Op == ast.OpVector {
			depth = len(invoke.Args[1].Sub.(*ast.VectorNode).Items)
		}
		if isCoreInvoke(invoke, "mapv") && depth < 2 {
			depth = 2
		}
		if invoke.Fn != nil && invoke.Fn.Op == ast.OpVar {
			if target := targets[invoke.Fn.Sub.(*ast.VarNode).Var]; target != nil && target.ownedVectorAnalysis != nil {
				for _, candidate := range target.ownedVectorAnalysis.paramDepths {
					if candidate > depth {
						depth = candidate
					}
				}
			}
		}
	}
	if node.Op == ast.OpHostCall &&
		isVectorAOTNth(node.Sub.(*ast.HostCallNode)) {
		depth = 1
	}
	for _, child := range compiler.IRChildren(node) {
		if childDepth := ownedVectorAOTMaxDepth(child, targets); childDepth > depth {
			depth = childDepth
		}
	}
	return depth
}

func ownedVectorAOTResultMatchesParams(
	analysis *ownedVectorAOTAnalysis,
) bool {
	for index, depth := range analysis.paramDepths {
		if analysis.result.origins&(uint32(1)<<index) != 0 &&
			analysis.result.depth != depth {
			return false
		}
	}
	return true
}

func ownedVectorAOTParamMask(analysis *ownedVectorAOTAnalysis) uint32 {
	var result uint32
	if analysis == nil {
		return 0
	}
	for index, depth := range analysis.paramDepths {
		if depth > 0 {
			result |= uint32(1) << index
		}
	}
	return result
}

func ownedVectorAOTDepthScore(analysis *ownedVectorAOTAnalysis) int {
	result := 0
	for _, depth := range analysis.paramDepths {
		result += depth
	}
	return result
}

func (a *ownedVectorAOTAnalyzer) expr(
	node *ast.Node,
	locals map[*lang.Symbol]ownedVectorAOTValue,
) ownedVectorAOTValue {
	if node == nil || !a.valid {
		return ownedVectorAOTValue{}
	}
	var result ownedVectorAOTValue
	switch node.Op {
	case ast.OpConst:
		_, result.linearIndexed = AsLinearIndexed(
			node.Sub.(*ast.ConstNode).Value,
		)

	case ast.OpVar:
		vr := node.Sub.(*ast.VarNode).Var
		if aotVarCanDirectLink(vr) {
			_, result.linearIndexed = AsLinearIndexed(codegenVarValue(vr))
		}

	case ast.OpLocal:
		result = locals[node.Sub.(*ast.LocalNode).Name]
		if result.forbidden {
			a.valid = false
		}

	case ast.OpDo:
		do := node.Sub.(*ast.DoNode)
		for _, statement := range do.Statements {
			if value := a.expr(statement, locals); value.depth > 0 {
				a.valid = false
			}
		}
		result = a.expr(do.Ret, locals)

	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		nested := cloneOwnedVectorAOTLocals(locals)
		for _, bindingNode := range let.Bindings {
			binding := bindingNode.Sub.(*ast.BindingNode)
			nested[binding.Name] = a.expr(binding.Init, nested)
		}
		result = a.expr(let.Body, nested)

	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		if value := a.expr(conditional.Test, locals); value.depth > 0 {
			a.valid = false
			break
		}
		thenValue := a.expr(conditional.Then, locals)
		elseValue := a.expr(conditional.Else, locals)
		result = a.combine(thenValue, elseValue)

	case ast.OpHostCall:
		result = a.hostCall(node, locals)

	case ast.OpAssoc:
		result = a.assoc(node, locals)

	case ast.OpInvoke:
		result = a.invoke(node, locals)

	default:
		// Unsupported forms cannot safely capture, expose, or observe the
		// private mutable representation.
		a.valid = false
	}
	a.analysis.values[node] = result
	return result
}

func (a *ownedVectorAOTAnalyzer) hostCall(
	node *ast.Node,
	locals map[*lang.Symbol]ownedVectorAOTValue,
) ownedVectorAOTValue {
	call := node.Sub.(*ast.HostCallNode)
	if isVectorAOTNth(call) && len(call.Args) == 2 {
		target := a.expr(call.Args[0], locals)
		index := a.expr(call.Args[1], locals)
		if index.depth > 0 {
			a.valid = false
			return ownedVectorAOTValue{}
		}
		if target.depth > 0 {
			a.analysis.nths[node] = target.depth
			if target.depth > 1 {
				return ownedVectorAOTValue{
					depth:   target.depth - 1,
					origins: target.origins,
				}
			}
			return ownedVectorAOTValue{}
		}
		return ownedVectorAOTValue{}
	}

	if call.Target != nil {
		if value := a.expr(call.Target, locals); value.depth > 0 {
			a.valid = false
		}
	}
	for _, argument := range call.Args {
		if value := a.expr(argument, locals); value.depth > 0 {
			a.valid = false
		}
	}
	return ownedVectorAOTValue{}
}

func (a *ownedVectorAOTAnalyzer) assoc(
	node *ast.Node,
	locals map[*lang.Symbol]ownedVectorAOTValue,
) ownedVectorAOTValue {
	assoc := node.Sub.(*ast.AssocNode)
	target := a.expr(assoc.Target, locals)
	for _, entry := range assoc.Entries {
		if key := a.expr(entry.Key, locals); key.depth > 0 {
			a.valid = false
		}
		if value := a.expr(entry.Val, locals); value.depth > 0 {
			a.valid = false
		}
	}
	if target.depth == 0 {
		return ownedVectorAOTValue{}
	}
	if target.depth != 1 {
		a.valid = false
		return ownedVectorAOTValue{}
	}
	_, targetIsAssoc := a.analysis.assocs[assoc.Target]
	a.analysis.assocs[node] = !targetIsAssoc
	a.analysis.mutated |= target.origins
	return target
}

func (a *ownedVectorAOTAnalyzer) invoke(
	node *ast.Node,
	locals map[*lang.Symbol]ownedVectorAOTValue,
) ownedVectorAOTValue {
	invoke := node.Sub.(*ast.InvokeNode)
	switch {
	case isCoreInvoke(invoke, "assoc-in"):
		return a.assocIn(node, invoke, locals)
	case isCoreInvoke(invoke, "reduce"):
		if result, ok := a.reduce(node, invoke, locals); ok {
			return result
		}
	case isCoreInvoke(invoke, "mapv"):
		if result, ok := a.mapv(node, invoke, locals); ok {
			return result
		}
	}

	values := make([]ownedVectorAOTValue, len(invoke.Args))
	hasOwned := false
	for index, argument := range invoke.Args {
		values[index] = a.expr(argument, locals)
		hasOwned = hasOwned || values[index].depth > 0
	}
	if !hasOwned {
		if invoke.Fn.Op == ast.OpFn {
			a.valid = false
		}
		return ownedVectorAOTValue{}
	}

	target := a.ownedCallee(invoke)
	if target == nil {
		a.valid = false
		return ownedVectorAOTValue{}
	}
	var origins uint32
	for index, value := range values {
		expected := target.ownedVectorAnalysis.paramDepths[index]
		if value.depth != expected {
			a.valid = false
			return ownedVectorAOTValue{}
		}
		if expected > 0 {
			origins |= value.origins
		} else if target.ownedVectorAnalysis.indexedParams&
			(uint32(1)<<index) != 0 && !a.requireIndexed(value) {
			a.valid = false
			return ownedVectorAOTValue{}
		}
	}
	a.analysis.calls[node] = target
	a.analysis.mutated |= origins
	return ownedVectorAOTValue{
		depth:   target.ownedVectorAnalysis.result.depth,
		origins: origins,
	}
}

func (a *ownedVectorAOTAnalyzer) assocIn(
	node *ast.Node,
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]ownedVectorAOTValue,
) ownedVectorAOTValue {
	if len(invoke.Args) != 3 ||
		!ownedVectorCoreVarAvailable(invoke) {
		a.valid = false
		return ownedVectorAOTValue{}
	}
	target := a.expr(invoke.Args[0], locals)
	path := invoke.Args[1]
	if target.depth == 0 || path == nil || path.Op != ast.OpVector {
		a.valid = false
		return ownedVectorAOTValue{}
	}
	items := path.Sub.(*ast.VectorNode).Items
	if len(items) == 0 || len(items) != target.depth {
		a.valid = false
		return ownedVectorAOTValue{}
	}
	for _, item := range items {
		if value := a.expr(item, locals); value.depth > 0 {
			a.valid = false
		}
	}
	if value := a.expr(invoke.Args[2], locals); value.depth > 0 {
		a.valid = false
	}
	a.analysis.assocIns[node] = ownedVectorAOTAssocIn{
		path: append([]*ast.Node(nil), items...),
		copy: !ownedVectorAOTIsAssocIn(a.analysis, invoke.Args[0]),
	}
	a.analysis.mutated |= target.origins
	return target
}

func ownedVectorAOTIsAssocIn(
	analysis *ownedVectorAOTAnalysis,
	node *ast.Node,
) bool {
	_, ok := analysis.assocIns[node]
	return ok
}

func (a *ownedVectorAOTAnalyzer) reduce(
	node *ast.Node,
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]ownedVectorAOTValue,
) (ownedVectorAOTValue, bool) {
	if len(invoke.Args) != 3 || !ownedVectorCoreVarAvailable(invoke) {
		return ownedVectorAOTValue{}, false
	}
	method := inlinePipelineMethod(invoke.Args[0])
	if method == nil || method.FixedArity != 2 {
		return ownedVectorAOTValue{}, false
	}
	if !ownedVectorAOTRepeatable(invoke.Args[1]) ||
		!ownedVectorAOTRepeatable(invoke.Args[2]) {
		return ownedVectorAOTValue{}, false
	}
	initial := a.expr(invoke.Args[1], locals)
	source := a.expr(invoke.Args[2], locals)
	if initial.depth == 0 || source.depth > 0 ||
		!a.requireIndexed(source) {
		return ownedVectorAOTValue{}, false
	}

	callbackLocals := forbidOwnedVectorAOTLocals(locals)
	callbackLocals[method.Params[0].Sub.(*ast.BindingNode).Name] = initial
	callbackLocals[method.Params[1].Sub.(*ast.BindingNode).Name] =
		ownedVectorAOTValue{}
	result := a.expr(method.Body, callbackLocals)
	if !sameOwnedVectorAOTValue(result, initial) {
		a.valid = false
		return ownedVectorAOTValue{}, true
	}
	a.analysis.reduces[node] = &ownedVectorAOTReduce{method: method}
	return initial, true
}

func ownedVectorAOTRepeatable(node *ast.Node) bool {
	return node != nil &&
		(node.Op == ast.OpConst || node.Op == ast.OpLocal || node.Op == ast.OpVar)
}

func (a *ownedVectorAOTAnalyzer) requireIndexed(
	value ownedVectorAOTValue,
) bool {
	if value.linearIndexed {
		return true
	}
	parameters := value.parameters
	if value.depth > 0 || parameters == 0 || parameters&(parameters-1) != 0 {
		return false
	}
	a.analysis.indexedParams |= parameters
	return true
}

func (a *ownedVectorAOTAnalyzer) mapv(
	node *ast.Node,
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]ownedVectorAOTValue,
) (ownedVectorAOTValue, bool) {
	if len(invoke.Args) != 2 || !ownedVectorCoreVarAvailable(invoke) {
		return ownedVectorAOTValue{}, false
	}
	method := inlinePipelineMethod(invoke.Args[0])
	if method == nil || method.FixedArity != 1 {
		return ownedVectorAOTValue{}, false
	}
	source := a.expr(invoke.Args[1], locals)
	if source.depth < 2 {
		return ownedVectorAOTValue{}, false
	}
	item := ownedVectorAOTValue{
		depth:   source.depth - 1,
		origins: source.origins,
	}
	callbackLocals := forbidOwnedVectorAOTLocals(locals)
	callbackLocals[method.Params[0].Sub.(*ast.BindingNode).Name] = item
	result := a.expr(method.Body, callbackLocals)
	if !sameOwnedVectorAOTValue(result, item) {
		a.valid = false
		return ownedVectorAOTValue{}, true
	}
	a.analysis.mapvs[node] = &ownedVectorAOTMapv{method: method}
	return source, true
}

func ownedVectorCoreVarAvailable(invoke *ast.InvokeNode) bool {
	if invoke == nil || invoke.Fn == nil || invoke.Fn.Op != ast.OpVar {
		return false
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	return aotVarCanDirectLink(vr) && IsDefaultCoreVar(vr)
}

func (a *ownedVectorAOTAnalyzer) ownedCallee(
	invoke *ast.InvokeNode,
) *aotSpecializationTarget {
	if invoke == nil || invoke.Fn.Op != ast.OpVar {
		return nil
	}
	target := a.targets[invoke.Fn.Sub.(*ast.VarNode).Var]
	if target == nil || !target.directLinked ||
		target.ownedVectorAnalysis == nil ||
		len(invoke.Args) != target.arity {
		return nil
	}
	return target
}

func (a *ownedVectorAOTAnalyzer) combine(
	left, right ownedVectorAOTValue,
) ownedVectorAOTValue {
	if left.control {
		return right
	}
	if right.control {
		return left
	}
	if left.depth != right.depth {
		a.valid = false
		return ownedVectorAOTValue{}
	}
	if left.depth > 0 {
		if left.origins != right.origins {
			a.valid = false
			return ownedVectorAOTValue{}
		}
		return left
	}
	return ownedVectorAOTValue{
		parameters:    left.parameters & right.parameters,
		linearIndexed: left.linearIndexed && right.linearIndexed,
	}
}

func sameOwnedVectorAOTValue(
	left, right ownedVectorAOTValue,
) bool {
	return left.depth > 0 &&
		left.depth == right.depth &&
		left.origins == right.origins &&
		!left.control && !left.forbidden
}

func cloneOwnedVectorAOTLocals(
	locals map[*lang.Symbol]ownedVectorAOTValue,
) map[*lang.Symbol]ownedVectorAOTValue {
	result := make(map[*lang.Symbol]ownedVectorAOTValue, len(locals)+2)
	for name, value := range locals {
		result[name] = value
	}
	return result
}

func forbidOwnedVectorAOTLocals(
	locals map[*lang.Symbol]ownedVectorAOTValue,
) map[*lang.Symbol]ownedVectorAOTValue {
	result := cloneOwnedVectorAOTLocals(locals)
	for name, value := range result {
		if value.depth > 0 {
			value.forbidden = true
			result[name] = value
		}
	}
	return result
}

func ownedVectorAOTParamTypes(
	analysis *ownedVectorAOTAnalysis,
) []string {
	params := make([]string, len(analysis.paramDepths))
	for index, depth := range analysis.paramDepths {
		if depth > 0 {
			params[index] = "*runtime.OwnedVector"
		} else if analysis.indexedParams&(uint32(1)<<index) != 0 {
			params[index] = "lang.Indexed"
		} else {
			params[index] = "any"
		}
	}
	return params
}

func ownedVectorAOTGoType(value ownedVectorAOTValue) string {
	if value.depth > 0 {
		return "*runtime.OwnedVector"
	}
	return "any"
}

func (g *Generator) generateOwnedVectorSpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn ||
		target.ownedVectorAnalysis == nil {
		return false
	}
	analysis := target.ownedVectorAnalysis
	helper := g.allocateTempVar()
	paramTypes := ownedVectorAOTParamTypes(analysis)
	typedParams := make([]string, len(paramNames))
	for index, name := range paramNames {
		typedParams[index] = name + " " + paramTypes[index]
	}
	g.writef("var %s func(%s) (*runtime.OwnedVector, bool)\n",
		helper, strings.Join(typedParams, ", "))
	g.writef("%s = func(%s) (*runtime.OwnedVector, bool) {\n",
		helper, strings.Join(typedParams, ", "))
	previous := g.currentOwnedVector
	g.currentOwnedVector = analysis
	g.generateOwnedVectorFnMethodFixed(method, paramNames)
	g.currentOwnedVector = previous
	g.writef("}\n")
	g.writef("%s = %s\n", target.ownedVectorFnVar, helper)

	signature := ""
	if method.FixedArity > 0 {
		signature = strings.Join(paramNames, ", ") + " any"
	}
	g.writef("%s = lang.FnFunc%d(func(%s) any {\n",
		fnVar, method.FixedArity, signature)
	fastArgs := append([]string(nil), paramNames...)
	var guards []string
	for index, depth := range analysis.paramDepths {
		if depth == 0 &&
			analysis.indexedParams&(uint32(1)<<index) == 0 {
			continue
		}
		value := g.allocateTempVar()
		ok := g.allocateTempVar()
		if depth > 0 {
			g.writef("%s, %s := runtime.NewOwnedVector(%s, %d)\n",
				value, ok, paramNames[index], depth)
		} else {
			g.writef("%s, %s := runtime.AsLinearIndexed(%s)\n",
				value, ok, paramNames[index])
		}
		fastArgs[index] = value
		guards = append(guards, ok)
	}
	g.writef("if %s {\n", strings.Join(guards, " && "))
	result := g.allocateTempVar()
	ok := g.allocateTempVar()
	g.writef("%s, %s := %s(%s)\n",
		result, ok, helper, strings.Join(fastArgs, ", "))
	g.writef("if %s {\n", ok)
	g.writef("return %s.Persistent()\n", result)
	g.writef("}\n")
	g.writef("}\n")
	g.generateFnMethodFixed(method, paramNames)
	g.writef("})\n")
	return true
}

func (g *Generator) generateOwnedVectorFnMethodFixed(
	method *ast.FnMethodNode,
	paramNames []string,
) {
	g.pushVarScope()
	defer g.popVarScope()
	for index, param := range method.Params {
		name := param.Sub.(*ast.BindingNode).Name.Name()
		local := g.allocateLocal(name)
		g.writef("%s := %s\n", local, paramNames[index])
		g.writeAssign("_", local)
	}
	body := g.generateASTNode(method.Body)
	if body != "" {
		g.writef("return %s, true\n", body)
	}
}

func (g *Generator) generateOwnedVectorAOTInvoke(
	node *ast.Node,
) (string, bool) {
	if g.currentOwnedVector == nil {
		return "", false
	}
	if target := g.currentOwnedVector.calls[node]; target != nil {
		invoke := node.Sub.(*ast.InvokeNode)
		args := make([]string, len(invoke.Args))
		for index, argument := range invoke.Args {
			args[index] = g.generateASTNode(argument)
			if target.ownedVectorAnalysis.indexedParams&
				(uint32(1)<<index) != 0 {
				args[index] = "runtime.MustLinearIndexed(" + args[index] + ")"
			}
		}
		result := g.allocateTempVar()
		ok := g.allocateTempVar()
		g.writef("%s, %s := %s(%s)\n",
			result,
			ok,
			target.ownedVectorFnVar,
			strings.Join(args, ", "),
		)
		g.writef("if !%s {\n", ok)
		g.generateOwnedVectorFallbackReturn()
		g.writef("}\n")
		return result, true
	}
	if plan := g.currentOwnedVector.assocIns[node]; len(plan.path) != 0 {
		return g.generateOwnedVectorAOTAssocIn(node, plan)
	}
	if plan := g.currentOwnedVector.reduces[node]; plan != nil {
		return g.generateOwnedVectorAOTReduce(node, plan), true
	}
	if plan := g.currentOwnedVector.mapvs[node]; plan != nil {
		return g.generateOwnedVectorAOTMapv(node, plan), true
	}
	return "", false
}

func (g *Generator) generateOwnedVectorAOTNth(
	node *ast.Node,
) (string, bool) {
	if g.currentOwnedVector == nil {
		return "", false
	}
	depth := g.currentOwnedVector.nths[node]
	if depth == 0 {
		return "", false
	}
	call := node.Sub.(*ast.HostCallNode)
	target := g.generateASTNode(call.Args[0])
	index := g.generateASTNode(call.Args[1])
	result := g.allocateTempVar()
	if depth > 1 {
		g.writef("%s := %s.NestedSnapshot(lang.IntCast(%s))\n",
			result, target, index)
	} else {
		g.writef("%s := %s.Nth(lang.IntCast(%s))\n",
			result, target, index)
	}
	return result, true
}

func (g *Generator) generateOwnedVectorAOTAssoc(
	node *ast.Node,
) (string, bool) {
	if g.currentOwnedVector == nil {
		return "", false
	}
	copyTarget, exists := g.currentOwnedVector.assocs[node]
	if !exists {
		return "", false
	}
	assoc := node.Sub.(*ast.AssocNode)
	target := g.generateASTNode(assoc.Target)
	for index, entry := range assoc.Entries {
		key := g.generateASTNode(entry.Key)
		value := g.generateASTNode(entry.Val)
		if copyTarget && index == 0 {
			updated := g.allocateTempVar()
			g.writef("%s := %s.AssocCopy(lang.IntCast(%s), %s)\n",
				updated, target, key, value)
			target = updated
		} else {
			g.writef("%s.Assoc(lang.IntCast(%s), %s)\n",
				target, key, value)
		}
	}
	return target, true
}

func (g *Generator) generateOwnedVectorAOTAssocIn(
	node *ast.Node,
	plan ownedVectorAOTAssocIn,
) (string, bool) {
	invoke := node.Sub.(*ast.InvokeNode)
	target := g.generateASTNode(invoke.Args[0])
	path := make([]string, len(plan.path))
	for index, item := range plan.path {
		path[index] = g.generateASTNode(item)
	}
	value := g.generateASTNode(invoke.Args[2])
	if len(path) == 2 && plan.copy {
		updated := g.allocateTempVar()
		g.writef(
			"%s := %s.AssocIn2Copy(lang.IntCast(%s), %s, %s)\n",
			updated, target, path[0], path[1], value,
		)
		target = updated
	} else if len(path) == 2 {
		g.writef(
			"%s.AssocIn2(lang.IntCast(%s), %s, %s)\n",
			target, path[0], path[1], value,
		)
	} else {
		indices := make([]string, len(path))
		for index, item := range path {
			indices[index] = "lang.IntCast(" + item + ")"
		}
		method := "AssocPath"
		if plan.copy {
			method = "AssocPathCopy"
		}
		updated := g.allocateTempVar()
		g.writef("%s := %s.%s([]int{%s}, %s)\n",
			updated,
			target,
			method,
			strings.Join(indices, ", "),
			value,
		)
		target = updated
	}
	return target, true
}

func (g *Generator) generateOwnedVectorAOTReduce(
	node *ast.Node,
	plan *ownedVectorAOTReduce,
) string {
	invoke := node.Sub.(*ast.InvokeNode)
	initial := g.generateASTNode(invoke.Args[1])
	source := g.generateASTNode(invoke.Args[2])
	indexed := g.allocateTempVar()
	g.writef("%s := runtime.MustLinearIndexed(%s)\n", indexed, source)

	accumulator := g.allocateTempVar()
	index := g.allocateTempVar()
	item := g.allocateTempVar()
	g.writef("var %s *runtime.OwnedVector = %s\n",
		accumulator, initial)
	g.writef("for %s := 0; %s < %s.Count(); %s++ {\n",
		index, index, indexed, index)
	g.writef("%s := %s.Nth(%s)\n", item, indexed, index)
	next := g.generateOwnedVectorAOTCallback(
		plan.method,
		[]string{accumulator, item},
		[]ownedVectorAOTValue{
			g.currentOwnedVector.values[invoke.Args[1]],
			{},
		},
	)
	g.writef("%s = %s\n", accumulator, next)
	g.writef("}\n")
	return accumulator
}

func (g *Generator) generateOwnedVectorAOTMapv(
	node *ast.Node,
	plan *ownedVectorAOTMapv,
) string {
	invoke := node.Sub.(*ast.InvokeNode)
	source := g.generateASTNode(invoke.Args[1])
	index := g.allocateTempVar()
	item := g.allocateTempVar()
	g.writef("for %s := 0; %s < %s.Count(); %s++ {\n",
		index, index, source, index)
	g.writef("%s := %s.Nested(%s)\n", item, source, index)
	next := g.generateOwnedVectorAOTCallback(
		plan.method,
		[]string{item},
		[]ownedVectorAOTValue{{
			depth:   g.currentOwnedVector.values[invoke.Args[1]].depth - 1,
			origins: g.currentOwnedVector.values[invoke.Args[1]].origins,
		}},
	)
	g.writef("%s.Assoc(%s, %s)\n", source, index, next)
	g.writef("}\n")
	return source
}

func (g *Generator) generateOwnedVectorAOTCallback(
	method *ast.FnMethodNode,
	args []string,
	values []ownedVectorAOTValue,
) string {
	result := g.allocateTempVar()
	g.writef("var %s *runtime.OwnedVector\n", result)
	g.writef("{\n")
	g.pushVarScope()
	for index, param := range method.Params {
		name := param.Sub.(*ast.BindingNode).Name.Name()
		local := g.allocateLocal(name)
		g.writef("var %s %s = %s\n",
			local, ownedVectorAOTGoType(values[index]), args[index])
		g.writeAssign("_", local)
	}
	body := g.generateASTNode(method.Body)
	g.writeAssign(result, body)
	g.popVarScope()
	g.writef("}\n")
	return result
}

func (g *Generator) generateOwnedVectorFallbackReturn() {
	g.writef("return nil, false\n")
}
