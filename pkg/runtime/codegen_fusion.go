//go:build !glj_aot_runtime

package runtime

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
)

func (g *Generator) generateAOTInlineIndexedPipeline(
	invoke *ast.InvokeNode,
	plan *compiler.IRPipelinePlan,
) (string, bool) {
	if !g.directLink || invoke == nil || plan == nil ||
		!aotVarCanDirectLink(plan.ConsumerVar) {
		return "", false
	}
	target := g.aotExternalInvokeTarget(invoke)
	if target == nil || !target.directLinked {
		return "", false
	}

	var callback *ast.Node
	switch plan.Consumer {
	case compiler.IRPipelineReduce:
		if len(plan.Stages) != 0 || len(invoke.Args) != 3 {
			return "", false
		}
		callback = plan.Reducer
	case compiler.IRPipelineMapv:
		if len(plan.Stages) != 1 || len(invoke.Args) != 2 {
			return "", false
		}
		callback = plan.Stages[0].Callback
	default:
		return "", false
	}
	method := inlinePipelineMethod(callback)
	if method == nil {
		return "", false
	}

	var initial string
	if plan.Consumer == compiler.IRPipelineReduce {
		initial = g.generateASTNode(plan.Initial)
	}
	source := g.generateASTNode(plan.Source)
	result := g.allocateTempVar()
	indexed := g.allocateTempVar()
	ok := g.allocateTempVar()
	g.writef("var %s any\n", result)
	g.writef("%s, %s := any(%s).(lang.Indexed)\n",
		indexed, ok, source)
	g.writef("if %s {\n", ok)
	switch plan.Consumer {
	case compiler.IRPipelineReduce:
		g.generateAOTInlineIndexedReduce(
			result,
			indexed,
			initial,
			method,
		)
	case compiler.IRPipelineMapv:
		g.generateAOTInlineIndexedMapv(result, indexed, method)
	}
	g.writef("} else {\n")
	fallbackCallback := g.generateASTNode(callback)
	switch plan.Consumer {
	case compiler.IRPipelineReduce:
		g.writef("%s = %s(%s, %s, %s)\n",
			result,
			target.fnVar,
			fallbackCallback,
			initial,
			source,
		)
	case compiler.IRPipelineMapv:
		g.writef("%s = %s(%s, %s)\n",
			result,
			target.fnVar,
			fallbackCallback,
			source,
		)
	}
	g.writef("}\n")
	return result, true
}

func inlinePipelineMethod(callback *ast.Node) *ast.FnMethodNode {
	if callback == nil || callback.Op != ast.OpFn {
		return nil
	}
	fn := callback.Sub.(*ast.FnNode)
	if fn.IsVariadic || len(fn.Methods) != 1 {
		return nil
	}
	method := fn.Methods[0].Sub.(*ast.FnMethodNode)
	if method.IsVariadic {
		return nil
	}
	return method
}

func (g *Generator) generateAOTInlineIndexedReduce(
	result, indexed, initial string,
	method *ast.FnMethodNode,
) {
	accumulator := g.allocateTempVar()
	index := g.allocateTempVar()
	item := g.allocateTempVar()
	g.writef("var %s any = %s\n", accumulator, initial)
	g.writef("for %s := 0; %s < %s.Count(); %s++ {\n",
		index, index, indexed, index)
	g.writef("%s := %s.Nth(%s)\n", item, indexed, index)
	next := g.generateInlinePipelineCallback(
		method,
		[]string{accumulator, item},
	)
	g.writef("if lang.IsReduced(%s) {\n", next)
	g.writef("%s = %s.(lang.IDeref).Deref()\n", accumulator, next)
	g.writef("break\n")
	g.writef("}\n")
	g.writef("%s = %s\n", accumulator, next)
	g.writef("}\n")
	g.writef("%s = %s\n", result, accumulator)
}

func (g *Generator) generateAOTInlineIndexedMapv(
	result, indexed string,
	method *ast.FnMethodNode,
) {
	builder := g.allocateTempVar()
	index := g.allocateTempVar()
	item := g.allocateTempVar()
	g.writef("%s := lang.NewVector().AsTransient().(*lang.TransientVector)\n",
		builder)
	g.writef("for %s := 0; %s < %s.Count(); %s++ {\n",
		index, index, indexed, index)
	g.writef("%s := %s.Nth(%s)\n", item, indexed, index)
	next := g.generateInlinePipelineCallback(method, []string{item})
	g.writef("%s.Conj(%s)\n", builder, next)
	g.writef("}\n")
	g.writef("%s = %s.Persistent()\n", result, builder)
}

func (g *Generator) generateInlinePipelineCallback(
	method *ast.FnMethodNode,
	args []string,
) string {
	result := g.allocateTempVar()
	g.writef("var %s any\n", result)
	g.writef("{\n")
	g.pushVarScope()
	for index, param := range method.Params {
		name := param.Sub.(*ast.BindingNode).Name.Name()
		local := g.allocateLocal(name)
		g.writef("var %s any = %s\n", local, args[index])
		g.writeAssign("_", local)
	}
	body := g.generateASTNode(method.Body)
	g.writeAssign(result, body)
	g.popVarScope()
	g.writef("}\n")
	return result
}
