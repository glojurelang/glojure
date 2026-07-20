package runtime

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (c threadedEvalCompiler) compileReducePipeline(
	node *ast.Node,
	invoke *ast.InvokeNode,
) evalFn {
	plan := analyzeReducePipeline(invoke)
	if plan == nil {
		return nil
	}
	initial, ok := int64ConstValue(plan.initial)
	if !ok {
		return nil
	}
	source := c.compile(plan.source)
	if source == nil {
		return nil
	}
	transforms := make([]ReducePipelineTransformKind, len(plan.transforms))
	for i, transform := range plan.transforms {
		transforms[i] = transform.kind
	}
	guards := append([]*lang.Var(nil), plan.guardVars...)

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
			plan.takeLimit,
		), nil
	}
}
