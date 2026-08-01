package compiler

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// AnalyzePipeline recognizes collection-producing and collection-consuming
// ASTs. Lowering is based on the callback and source representations rather
// than the identity or expression shape of a particular callback.
func AnalyzePipeline(node *ast.Node) *IRPipelinePlan {
	if node == nil || node.Op != ast.OpInvoke {
		return nil
	}
	invoke := node.Sub.(*ast.InvokeNode)
	consumerVar, namespace, name, known := irVarCall(invoke)
	if !known || namespace != "clojure.core" {
		return nil
	}

	plan := &IRPipelinePlan{ConsumerVar: consumerVar}
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
	if irCanInlineIndexedPipeline(plan) {
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
		stage.CallbackVar = irPipelineCallback(stage.Callback)
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
	callbackVar := irPipelineCallback(callback)
	return IRPipelineStage{
		Kind:        IRPipelineMap,
		Callback:    callback,
		OperatorVar: operator,
		CallbackVar: callbackVar,
	}
}

func irPipelineCallback(callback *ast.Node) *lang.Var {
	if callback != nil && callback.Op == ast.OpVar {
		return callback.Sub.(*ast.VarNode).Var
	}
	return nil
}
