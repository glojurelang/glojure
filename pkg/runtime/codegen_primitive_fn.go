//go:build !glj_aot_runtime

package runtime

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// int64CallableAOTAnalysis describes a primitive entry point inferred for an
// ordinary unary function value. Unlike direct-call specializations, this
// entry point is discoverable by any higher-order consumer through an optional
// language interface.
type int64CallableAOTAnalysis struct {
	primitive *int64AOTAnalysis
	result    aotPrimitiveType
}

func (g *Generator) analyzeInt64Callable(
	fn *Fn,
	method *ast.FnMethodNode,
) *int64CallableAOTAnalysis {
	if !g.directLink || fn == nil || method == nil ||
		method.IsVariadic || method.FixedArity != 1 ||
		len(g.aotProtocolMethods[fn]) != 0 ||
		len(g.aotMultiFnMethods[fn]) != 0 {
		return nil
	}

	target := &aotSpecializationTarget{
		fn:           fn,
		arity:        1,
		directLinked: true,
	}
	analysis := &int64AOTAnalysis{
		target:             target,
		arity:              1,
		uncheckedHostCalls: make(map[*ast.HostCallNode]bool),
		callees:            make(map[*aotSpecializationTarget]struct{}),
		allowCoreMod:       true,
	}
	analyzer := newInt64AOTAnalyzer(analysis, g.aotCallTargets)
	parameter := method.Params[0].Sub.(*ast.BindingNode).Name
	result := analyzer.exprType(
		method.Body,
		map[*lang.Symbol]aotPrimitiveType{
			parameter: int64AOTPrimitive,
		},
	)
	if result != int64AOTPrimitive && result != boolAOTPrimitive {
		return nil
	}
	if analysis.usesSelf || len(analysis.callees) != 0 {
		return nil
	}
	return &int64CallableAOTAnalysis{
		primitive: analysis,
		result:    result,
	}
}

func (g *Generator) generateInt64CallableFixedFn(
	fnVar string,
	method *ast.FnMethodNode,
	analysis *int64CallableAOTAnalysis,
) {
	g.writef("{\n")
	defer g.writef("}\n")

	fallback := g.allocateTempVar()
	g.writef("%s := lang.FnFunc1(func(p0 any) any {\n", fallback)
	g.generateFnMethodFixed(method, []string{"p0"})
	g.writef("})\n")

	parameter := g.allocateTempVar()
	locals := map[*lang.Symbol]aotTypedLocal{
		method.Params[0].Sub.(*ast.BindingNode).Name: {
			name: parameter,
			typ:  int64AOTPrimitive,
		},
	}
	constructor := "lang.NewInt64UnaryFn"
	resultType := "int64"
	if analysis.result == boolAOTPrimitive {
		constructor = "lang.NewInt64PredicateFn"
		resultType = "bool"
	}
	g.writef("%s = %s(%s, func(%s int64) %s {\n",
		fnVar,
		constructor,
		fallback,
		parameter,
		resultType,
	)
	emitter := int64AOTEmitter{
		g:        g,
		analysis: analysis.primitive,
		callees:  map[*aotSpecializationTarget]string{},
	}
	result := emitter.emitExpr(method.Body, locals)
	g.writef("return %s\n", result)
	g.writef("})\n")
}
