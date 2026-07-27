package runtime

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (c threadedEvalCompiler) compileReducePipeline(
	node *ast.Node,
	invoke *ast.InvokeNode,
) evalFn {
	plan := compiler.AnalyzePipeline(node)
	if plan == nil || plan.Lowering != compiler.IRPipelineReduceInt64 {
		return nil
	}
	initial, ok := int64ConstValue(plan.Initial)
	if !ok {
		return nil
	}
	source := c.compile(plan.Source)
	if source == nil {
		return nil
	}
	var transforms []ReducePipelineTransformKind
	for _, stage := range plan.Stages {
		if stage.Kind == compiler.IRPipelineMap ||
			stage.Kind == compiler.IRPipelineFilter {
			transforms = append(transforms, stage.Primitive)
		}
	}
	guards := append([]*lang.Var(nil), plan.GuardVars...)

	return func(env *environment) (result interface{}, err error) {
		for _, vr := range guards {
			if !IsDefaultCoreVar(vr) {
				return env.evalASTInvokeDefault(node)
			}
		}
		defer env.recoverInvoke(node, &result, &err)
		coll, err := source(env)
		if err != nil {
			return nil, err
		}
		return ReduceInt64Pipeline(
			initial,
			coll,
			transforms,
			plan.TakeLimit,
		), nil
	}
}
