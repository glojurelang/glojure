//go:build !glj_aot_runtime

package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (g *Generator) generateAOTReducePipeline(
	invoke *ast.InvokeNode,
	plan *compiler.IRPipelinePlan,
) string {
	guards := make([]string, 0, len(plan.GuardVars))
	seen := make(map[*lang.Var]bool, len(plan.GuardVars))
	for _, vr := range plan.GuardVars {
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
	reducer := g.generateASTNode(plan.Reducer)
	initial := g.generateASTNode(plan.Initial)
	for _, stage := range plan.Stages {
		if stage.Callback != nil {
			callback := g.generateASTNode(stage.Callback)
			g.writef("_ = %s\n", callback)
		}
	}
	source := g.generateASTNode(plan.Source)
	g.writef("_ = %s\n", reducer)
	g.writef("%s = runtime.ReduceInt64Pipeline(\n", result)
	g.writef("%s, %s,\n", initial, source)
	g.writef("[]runtime.ReducePipelineTransformKind{\n")
	for _, stage := range plan.Stages {
		if stage.Kind == compiler.IRPipelineMap ||
			stage.Kind == compiler.IRPipelineFilter {
			g.writef("%s,\n", aotPipelineTransformName(stage.Primitive))
		}
	}
	g.writef("},\n")
	g.writef("%d,\n", plan.TakeLimit)
	g.writef(")\n")
	g.writef("} else {\n")
	fallback := g.generateInvokeDefault(invoke)
	g.writef("%s = %s\n", result, fallback)
	g.writef("}\n")
	return result
}

func aotPipelineTransformName(kind compiler.IRPipelinePrimitive) string {
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
