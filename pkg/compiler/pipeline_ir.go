package compiler

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// AnalyzePipeline recognizes collection-producing and collection-consuming
// ASTs. It describes general map/filter/take pipelines, but marks only the
// independently benchmarked integer reduce case for lowering.
func AnalyzePipeline(node *ast.Node) *IRPipelinePlan {
	if node == nil || node.Op != ast.OpInvoke {
		return nil
	}
	invoke := node.Sub.(*ast.InvokeNode)
	consumerVar, namespace, name, known := irVarCall(invoke)
	if !known || namespace != "clojure.core" {
		return nil
	}

	plan := &IRPipelinePlan{
		ConsumerVar: consumerVar,
		TakeLimit:   -1,
	}
	switch name {
	case "reduce":
		if len(invoke.Args) != 3 {
			return nil
		}
		plan.Consumer = IRPipelineReduce
		plan.Reducer = invoke.Args[0]
		plan.Initial = invoke.Args[1]
		plan.Source = invoke.Args[2]
	case "mapv":
		if len(invoke.Args) != 2 {
			return nil
		}
		plan.Consumer = IRPipelineMapv
		plan.Source = invoke.Args[1]
		plan.Stages = append(plan.Stages, irPipelineMapStage(
			invoke.Args[0],
			nil,
		))
	case "into":
		if len(invoke.Args) != 2 {
			return nil
		}
		plan.Consumer = IRPipelineInto
		plan.IntoTarget = invoke.Args[0]
		plan.Source = invoke.Args[1]
	case "map", "filter", "take":
		plan.Consumer = IRPipelineSeq
		plan.Source = node
	default:
		return nil
	}

	var sourceGuards []*lang.Var
	if plan.Consumer == IRPipelineMapv {
		source, stages, guards := irParsePipelineSource(plan.Source)
		plan.Source = source
		plan.Stages = append(stages, plan.Stages...)
		sourceGuards = guards
	} else {
		source, stages, guards := irParsePipelineSource(plan.Source)
		plan.Source = source
		plan.Stages = stages
		sourceGuards = guards
	}
	if len(plan.Stages) == 0 && plan.Consumer != IRPipelineReduce {
		return nil
	}

	plan.GuardVars = append(plan.GuardVars, consumerVar)
	plan.GuardVars = append(plan.GuardVars, sourceGuards...)
	for _, stage := range plan.Stages {
		if stage.OperatorVar != nil {
			plan.GuardVars = append(plan.GuardVars, stage.OperatorVar)
		}
		if stage.CallbackVar != nil {
			plan.GuardVars = append(plan.GuardVars, stage.CallbackVar)
		}
	}
	if irCanLowerInt64Reduce(plan) {
		plan.Lowering = IRPipelineReduceInt64
	} else if irCanInlineIndexedPipeline(plan) {
		plan.Lowering = IRPipelineInlineIndexed
	}
	return plan
}

func irCanInlineIndexedPipeline(plan *IRPipelinePlan) bool {
	if plan == nil {
		return false
	}
	switch plan.Consumer {
	case IRPipelineReduce:
		return len(plan.Stages) == 0 &&
			irInlineCallbackArity(plan.Reducer, 2)
	case IRPipelineMapv:
		return len(plan.Stages) == 1 &&
			plan.Stages[0].Kind == IRPipelineMap &&
			irInlineCallbackArity(plan.Stages[0].Callback, 1)
	default:
		return false
	}
}

func irInlineCallbackArity(node *ast.Node, arity int) bool {
	if !irFixedFnArity(node, arity) {
		return false
	}
	fn := node.Sub.(*ast.FnNode)
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	return !irContainsRecur(method.Body)
}

func irContainsRecur(node *ast.Node) bool {
	if node == nil {
		return false
	}
	if node.Op == ast.OpRecur {
		return true
	}
	for _, child := range irChildren(node) {
		if irContainsRecur(child) {
			return true
		}
	}
	return false
}

func irParsePipelineSource(
	node *ast.Node,
) (*ast.Node, []IRPipelineStage, []*lang.Var) {
	if node == nil || node.Op != ast.OpInvoke {
		return node, nil, nil
	}
	invoke := node.Sub.(*ast.InvokeNode)
	operator, namespace, name, known := irVarCall(invoke)
	if !known || namespace != "clojure.core" {
		return node, nil, nil
	}

	if name == "range" {
		return node, nil, []*lang.Var{operator}
	}
	if len(invoke.Args) != 2 ||
		(name != "map" && name != "filter" && name != "take") {
		return node, nil, nil
	}

	source, stages, guards := irParsePipelineSource(invoke.Args[1])
	stage := IRPipelineStage{OperatorVar: operator}
	switch name {
	case "map":
		stage = irPipelineMapStage(invoke.Args[0], operator)
	case "filter":
		stage.Kind = IRPipelineFilter
		stage.Callback = invoke.Args[0]
		stage.CallbackVar, stage.Primitive =
			irPipelineCallback("filter", stage.Callback)
	case "take":
		stage.Kind = IRPipelineTake
		stage.Limit = invoke.Args[0]
	}
	stages = append(stages, stage)
	guards = append(guards, operator)
	if stage.CallbackVar != nil {
		guards = append(guards, stage.CallbackVar)
	}
	return source, stages, guards
}

func irPipelineMapStage(
	callback *ast.Node,
	operator *lang.Var,
) IRPipelineStage {
	callbackVar, primitive := irPipelineCallback("map", callback)
	return IRPipelineStage{
		Kind:        IRPipelineMap,
		Callback:    callback,
		OperatorVar: operator,
		CallbackVar: callbackVar,
		Primitive:   primitive,
	}
}

func irPipelineCallback(
	operator string,
	callback *ast.Node,
) (*lang.Var, IRPipelinePrimitive) {
	if callback != nil && callback.Op == ast.OpVar {
		vr := callback.Sub.(*ast.VarNode).Var
		if vr != nil && vr.Namespace() != nil &&
			vr.Namespace().Name().String() == "clojure.core" {
			return vr, irPipelinePrimitive(operator, vr.Symbol().String())
		}
	}
	if operator == "map" && irInt64SquareFn(callback) {
		return nil, IRPipelineMapSquare
	}
	return nil, IRPipelinePrimitiveUnknown
}

func irPipelinePrimitive(operator, callback string) IRPipelinePrimitive {
	if operator == "map" {
		switch callback {
		case "identity":
			return IRPipelineMapIdentity
		case "inc":
			return IRPipelineMapInc
		case "dec":
			return IRPipelineMapDec
		}
		return IRPipelinePrimitiveUnknown
	}
	switch callback {
	case "odd?":
		return IRPipelineFilterOdd
	case "even?":
		return IRPipelineFilterEven
	case "pos?":
		return IRPipelineFilterPos
	case "neg?":
		return IRPipelineFilterNeg
	case "zero?":
		return IRPipelineFilterZero
	default:
		return IRPipelinePrimitiveUnknown
	}
}

func irCanLowerInt64Reduce(plan *IRPipelinePlan) bool {
	if plan.Consumer != IRPipelineReduce ||
		!irCoreVarNode(plan.Reducer, "+") ||
		!irInt64ConstNode(plan.Initial) ||
		!irConstantInt64Range(plan.Source) {
		return false
	}

	transformCount := 0
	for i, stage := range plan.Stages {
		switch stage.Kind {
		case IRPipelineMap:
			if stage.Primitive < IRPipelineMapIdentity ||
				stage.Primitive > IRPipelineMapSquare {
				return false
			}
			transformCount++
		case IRPipelineFilter:
			if stage.Primitive < IRPipelineFilterOdd ||
				stage.Primitive > IRPipelineFilterZero {
				return false
			}
			transformCount++
		case IRPipelineTake:
			if i != len(plan.Stages)-1 || plan.TakeLimit >= 0 {
				return false
			}
			limit, ok := irInt64ConstValue(stage.Limit)
			if !ok || limit < 0 {
				return false
			}
			plan.TakeLimit = limit
		default:
			return false
		}
	}
	if transformCount == 0 {
		return false
	}
	if reducer := plan.Reducer.Sub.(*ast.VarNode).Var; reducer != nil {
		plan.GuardVars = append(plan.GuardVars, reducer)
	}
	return true
}

func irConstantInt64Range(node *ast.Node) bool {
	if node == nil || node.Op != ast.OpInvoke {
		return false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	if !irCoreVarNode(invoke.Fn, "range") ||
		len(invoke.Args) < 1 || len(invoke.Args) > 3 {
		return false
	}
	for _, arg := range invoke.Args {
		if !irInt64ConstNode(arg) {
			return false
		}
	}
	return true
}

func irCoreVarNode(node *ast.Node, name string) bool {
	if node == nil || node.Op != ast.OpVar {
		return false
	}
	vr := node.Sub.(*ast.VarNode).Var
	return vr != nil && vr.Namespace() != nil &&
		vr.Namespace().Name().String() == "clojure.core" &&
		vr.Symbol().String() == name
}

func irInt64ConstNode(node *ast.Node) bool {
	_, ok := irInt64ConstValue(node)
	return ok
}

func irInt64ConstValue(node *ast.Node) (int64, bool) {
	if node == nil || node.Op != ast.OpConst {
		return 0, false
	}
	value, ok := node.Sub.(*ast.ConstNode).Value.(int64)
	return value, ok
}

func irInt64SquareFn(node *ast.Node) bool {
	if node == nil || node.Op != ast.OpFn {
		return false
	}
	fn := node.Sub.(*ast.FnNode)
	if fn.IsVariadic || len(fn.Methods) != 1 {
		return false
	}
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	if method.IsVariadic || method.FixedArity != 1 || len(method.Params) != 1 {
		return false
	}
	param := method.Params[0].Sub.(*ast.BindingNode).Name
	body := irUnwrapDo(method.Body)
	if body == nil || body.Op != ast.OpHostCall {
		return false
	}
	call := body.Sub.(*ast.HostCallNode)
	if call.Method == nil ||
		strings.ToLower(call.Method.Name()) != "multiply" ||
		len(call.Args) != 2 ||
		call.Target == nil || call.Target.Op != ast.OpConst {
		return false
	}
	hostSymbol := call.Target.Sub.(*ast.ConstNode).HostSymbol
	if hostSymbol == nil ||
		hostSymbol.String() !=
			"github.com:glojurelang:glojure:pkg:lang.Numbers" {
		return false
	}
	for _, arg := range call.Args {
		if !irLocalIs(arg, param) {
			return false
		}
	}
	return true
}
