//go:build !glj_aot_runtime

package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (g *Generator) generateAOTReducePipeline(
	invoke *ast.InvokeNode,
	plan *reducePipelinePlan,
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
	g.writef("%d,\n", plan.takeLimit)
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
	case ReducePipelineMapSquare:
		return "runtime.ReducePipelineMapSquare"
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

// A freshly constructed map used only by vec need not publish cached lazy
// values or tails. Both roots are checked before evaluating callbacks and
// source expressions, as in the original nested calls.
func (g *Generator) generateAOTVectorMap(invoke *ast.InvokeNode) (string, bool) {
	if !isCoreVarNode(invoke.Fn, "vec") || len(invoke.Args) != 1 ||
		invoke.Args[0].Op != ast.OpInvoke {
		return "", false
	}
	mapped := invoke.Args[0].Sub.(*ast.InvokeNode)
	if len(mapped.Args) != 2 {
		return "", false
	}
	helper := "runtime.VectorMap"
	roots := []*ast.Node{invoke.Fn, mapped.Fn}
	if !isCoreVarNode(mapped.Fn, "map") {
		if !isCoreVarNode(mapped.Fn, "mapcat") || !isCoreVarNode(mapped.Args[0], "seq") {
			return "", false
		}
		helper = "runtime.VectorMapcatSeq"
		roots = append(roots, mapped.Args[0])
	}
	guards := make([]string, 0, 2)
	for _, node := range roots {
		vr := node.Sub.(*ast.VarNode).Var
		if vr.IsDynamic() || lang.IsTruthy(lang.Get(vr.Meta(), lang.KWRedef)) {
			return "", false
		}
		id := g.allocVarVar(vr.Namespace().Name().String(), vr.Symbol().String())
		if g.currentValueInit != nil && id != g.currentValueInit.name {
			g.currentValueInit.deps[id] = struct{}{}
		}
		guards = append(guards, "runtime.IsDefaultCoreVar("+id+")")
	}
	result := g.allocateTempVar()
	g.writef("var %s any\n", result)
	g.writef("if %s {\n", strings.Join(guards, " && "))
	callback := g.generateASTNode(mapped.Args[0])
	source := g.generateASTNode(mapped.Args[1])
	g.writef("%s = %s(%s, %s)\n", result, helper, callback, source)
	g.writef("} else {\n")
	fallback := g.generateInvokeDefault(invoke)
	g.writef("%s = %s\n", result, fallback)
	g.writef("}\n")
	return result, true
}
