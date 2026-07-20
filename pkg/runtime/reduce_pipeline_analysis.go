package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type reducePipelineTransform struct {
	kind     ReducePipelineTransformKind
	callback *ast.Node
}

type reducePipelinePlan struct {
	reducer    *ast.Node
	initial    *ast.Node
	source     *ast.Node
	transforms []reducePipelineTransform
	takeLimit  int64
	guardVars  []*lang.Var
}

var purePipelineCallbacks = map[string]map[string]bool{
	"map": {
		"dec":      true,
		"identity": true,
		"inc":      true,
	},
	"filter": {
		"even?": true,
		"neg?":  true,
		"odd?":  true,
		"pos?":  true,
		"zero?": true,
	},
}

func analyzeReducePipeline(invoke *ast.InvokeNode) *reducePipelinePlan {
	if !isCoreVarNode(invoke.Fn, "reduce") || len(invoke.Args) != 3 ||
		!isCoreVarNode(invoke.Args[0], "+") ||
		!isInt64ConstNode(invoke.Args[1]) {
		return nil
	}
	source, transforms, takeLimit, guards, ok := analyzePipelineSource(invoke.Args[2])
	if !ok || len(transforms) == 0 {
		return nil
	}
	reduceVar := invoke.Fn.Sub.(*ast.VarNode).Var
	addVar := invoke.Args[0].Sub.(*ast.VarNode).Var
	return &reducePipelinePlan{
		reducer:    invoke.Args[0],
		initial:    invoke.Args[1],
		source:     source,
		transforms: transforms,
		takeLimit:  takeLimit,
		guardVars:  append([]*lang.Var{reduceVar, addVar}, guards...),
	}
}

func analyzePipelineSource(
	node *ast.Node,
) (*ast.Node, []reducePipelineTransform, int64, []*lang.Var, bool) {
	if node.Op != ast.OpInvoke {
		return nil, nil, -1, nil, false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	if isCoreVarNode(invoke.Fn, "range") {
		if len(invoke.Args) < 1 || len(invoke.Args) > 3 {
			return nil, nil, -1, nil, false
		}
		for _, arg := range invoke.Args {
			if !isInt64ConstNode(arg) {
				return nil, nil, -1, nil, false
			}
		}
		return node, nil, -1,
			[]*lang.Var{invoke.Fn.Sub.(*ast.VarNode).Var}, true
	}
	if len(invoke.Args) != 2 || invoke.Fn.Op != ast.OpVar {
		return nil, nil, -1, nil, false
	}
	operator := invoke.Fn.Sub.(*ast.VarNode).Var
	if operator.Namespace().Name().String() != "clojure.core" {
		return nil, nil, -1, nil, false
	}
	name := operator.Symbol().String()
	if name == "take" {
		limit, ok := int64ConstValue(invoke.Args[0])
		if !ok || limit < 0 {
			return nil, nil, -1, nil, false
		}
		source, transforms, innerLimit, guards, ok :=
			analyzePipelineSource(invoke.Args[1])
		if !ok || innerLimit >= 0 {
			return nil, nil, -1, nil, false
		}
		guards = append(guards, operator)
		return source, transforms, limit, guards, true
	}

	allowed := purePipelineCallbacks[name]
	if allowed == nil {
		return nil, nil, -1, nil, false
	}
	callbackNode := invoke.Args[0]
	var (
		kind        ReducePipelineTransformKind
		callbackVar *lang.Var
	)
	if callbackNode.Op == ast.OpVar {
		callbackVar = callbackNode.Sub.(*ast.VarNode).Var
		if callbackVar.Namespace().Name().String() != "clojure.core" ||
			!allowed[callbackVar.Symbol().String()] {
			return nil, nil, -1, nil, false
		}
		kind = pipelineTransformKind(name, callbackVar.Symbol().String())
	} else if name == "map" && isInt64SquareFn(callbackNode) {
		kind = ReducePipelineMapSquare
	} else {
		return nil, nil, -1, nil, false
	}

	source, transforms, takeLimit, guards, ok :=
		analyzePipelineSource(invoke.Args[1])
	if !ok {
		return nil, nil, -1, nil, false
	}
	transforms = append(transforms, reducePipelineTransform{
		kind:     kind,
		callback: callbackNode,
	})
	guards = append(guards, operator)
	if callbackVar != nil {
		guards = append(guards, callbackVar)
	}
	return source, transforms, takeLimit, guards, true
}

func int64ConstValue(node *ast.Node) (int64, bool) {
	if node.Op != ast.OpConst {
		return 0, false
	}
	value, ok := node.Sub.(*ast.ConstNode).Value.(int64)
	return value, ok
}

func isInt64SquareFn(node *ast.Node) bool {
	if node.Op != ast.OpFn {
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
	body := method.Body
	if body.Op != ast.OpDo {
		return false
	}
	do := body.Sub.(*ast.DoNode)
	if len(do.Statements) != 0 || do.Ret.Op != ast.OpHostCall {
		return false
	}
	call := do.Ret.Sub.(*ast.HostCallNode)
	if !isNumbersCall(call) ||
		strings.ToLower(call.Method.Name()) != "multiply" ||
		len(call.Args) != 2 {
		return false
	}
	for _, arg := range call.Args {
		if arg.Op != ast.OpLocal ||
			arg.Sub.(*ast.LocalNode).Name != param {
			return false
		}
	}
	return true
}

func pipelineTransformKind(
	operator, callback string,
) ReducePipelineTransformKind {
	if operator == "map" {
		switch callback {
		case "inc":
			return ReducePipelineMapInc
		case "dec":
			return ReducePipelineMapDec
		default:
			return ReducePipelineMapIdentity
		}
	}
	switch callback {
	case "odd?":
		return ReducePipelineFilterOdd
	case "even?":
		return ReducePipelineFilterEven
	case "pos?":
		return ReducePipelineFilterPos
	case "neg?":
		return ReducePipelineFilterNeg
	default:
		return ReducePipelineFilterZero
	}
}

func isCoreVarNode(node *ast.Node, name string) bool {
	if node.Op != ast.OpVar {
		return false
	}
	vr := node.Sub.(*ast.VarNode).Var
	return vr.Namespace().Name().String() == "clojure.core" &&
		vr.Symbol().String() == name
}

func isInt64ConstNode(node *ast.Node) bool {
	if node.Op != ast.OpConst {
		return false
	}
	_, ok := node.Sub.(*ast.ConstNode).Value.(int64)
	return ok
}
