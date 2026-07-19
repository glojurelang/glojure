//go:build !glj_aot_runtime

package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type aotReducePipelineTransform struct {
	kind     ReducePipelineTransformKind
	callback *ast.Node
}

type aotReducePipelinePlan struct {
	reducer    *ast.Node
	initial    *ast.Node
	source     *ast.Node
	transforms []aotReducePipelineTransform
	guardVars  []*lang.Var
}

var aotPurePipelineCallbacks = map[string]map[string]bool{
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

func analyzeAOTReducePipeline(invoke *ast.InvokeNode) *aotReducePipelinePlan {
	if !isCoreVarNode(invoke.Fn, "reduce") || len(invoke.Args) != 3 ||
		!isCoreVarNode(invoke.Args[0], "+") ||
		!isInt64ConstNode(invoke.Args[1]) {
		return nil
	}
	source, transforms, guards, ok := analyzeAOTPipelineSource(invoke.Args[2])
	if !ok || len(transforms) == 0 {
		return nil
	}
	reduceVar := invoke.Fn.Sub.(*ast.VarNode).Var
	addVar := invoke.Args[0].Sub.(*ast.VarNode).Var
	return &aotReducePipelinePlan{
		reducer:    invoke.Args[0],
		initial:    invoke.Args[1],
		source:     source,
		transforms: transforms,
		guardVars:  append([]*lang.Var{reduceVar, addVar}, guards...),
	}
}

func analyzeAOTPipelineSource(
	node *ast.Node,
) (*ast.Node, []aotReducePipelineTransform, []*lang.Var, bool) {
	if node.Op != ast.OpInvoke {
		return nil, nil, nil, false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	if isCoreVarNode(invoke.Fn, "range") {
		if len(invoke.Args) < 1 || len(invoke.Args) > 3 {
			return nil, nil, nil, false
		}
		for _, arg := range invoke.Args {
			if !isInt64ConstNode(arg) {
				return nil, nil, nil, false
			}
		}
		return node, nil, []*lang.Var{invoke.Fn.Sub.(*ast.VarNode).Var}, true
	}
	if len(invoke.Args) != 2 || invoke.Fn.Op != ast.OpVar ||
		invoke.Args[0].Op != ast.OpVar {
		return nil, nil, nil, false
	}
	operator := invoke.Fn.Sub.(*ast.VarNode).Var
	if operator.Namespace().Name().String() != "clojure.core" {
		return nil, nil, nil, false
	}
	name := operator.Symbol().String()
	allowed := aotPurePipelineCallbacks[name]
	callback := invoke.Args[0].Sub.(*ast.VarNode).Var
	if allowed == nil ||
		callback.Namespace().Name().String() != "clojure.core" ||
		!allowed[callback.Symbol().String()] {
		return nil, nil, nil, false
	}
	source, transforms, guards, ok := analyzeAOTPipelineSource(invoke.Args[1])
	if !ok {
		return nil, nil, nil, false
	}
	kind := aotPipelineTransformKind(name, callback.Symbol().String())
	transforms = append(transforms, aotReducePipelineTransform{
		kind:     kind,
		callback: invoke.Args[0],
	})
	guards = append(guards, operator, callback)
	return source, transforms, guards, true
}

func aotPipelineTransformKind(
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

func (g *Generator) generateAOTReducePipeline(
	invoke *ast.InvokeNode,
	plan *aotReducePipelinePlan,
) string {
	guards := make([]string, 0, len(plan.guardVars))
	seen := make(map[*lang.Var]bool, len(plan.guardVars))
	for _, vr := range plan.guardVars {
		if seen[vr] {
			continue
		}
		seen[vr] = true
		varID := g.allocVarVar(
			vr.Namespace().Name().String(),
			vr.Symbol().String(),
		)
		if g.currentValueInit != nil && varID != g.currentValueInit.name {
			g.currentValueInit.deps[varID] = struct{}{}
		}
		guards = append(guards, "runtime.IsDefaultCoreVar("+varID+")")
	}

	result := g.allocateTempVar()
	g.writef("var %s any\n", result)
	g.writef("if %s {\n", strings.Join(guards, " && "))
	reducer := g.generateASTNode(plan.reducer)
	initial := g.generateASTNode(plan.initial)
	for _, transform := range plan.transforms {
		callback := g.generateASTNode(transform.callback)
		g.writef("_ = %s\n", callback)
	}
	source := g.generateASTNode(plan.source)
	g.writef("_ = %s\n", reducer)
	g.writef("%s = runtime.ReduceInt64Pipeline(\n", result)
	g.writef("%s, %s,\n", initial, source)
	g.writef("[]runtime.ReducePipelineTransformKind{\n")
	for _, transform := range plan.transforms {
		g.writef("%s,\n", aotPipelineTransformName(transform.kind))
	}
	g.writef("},\n")
	g.writef(")\n")
	g.writef("} else {\n")
	fallback := g.generateInvokeDefault(invoke)
	g.writef("%s = %s\n", result, fallback)
	g.writef("}\n")
	return result
}

func aotPipelineTransformName(kind ReducePipelineTransformKind) string {
	switch kind {
	case ReducePipelineMapIdentity:
		return "runtime.ReducePipelineMapIdentity"
	case ReducePipelineMapInc:
		return "runtime.ReducePipelineMapInc"
	case ReducePipelineMapDec:
		return "runtime.ReducePipelineMapDec"
	case ReducePipelineFilterOdd:
		return "runtime.ReducePipelineFilterOdd"
	case ReducePipelineFilterEven:
		return "runtime.ReducePipelineFilterEven"
	case ReducePipelineFilterPos:
		return "runtime.ReducePipelineFilterPos"
	case ReducePipelineFilterNeg:
		return "runtime.ReducePipelineFilterNeg"
	default:
		return "runtime.ReducePipelineFilterZero"
	}
}
