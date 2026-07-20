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
