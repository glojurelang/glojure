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

func (g *Generator) canGenerateInt64Callable(fn *Fn) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn {
		return true
	}
	return target.int64Analysis == nil &&
		target.int64ParamAnalysis == nil &&
		target.float64Analysis == nil &&
		target.vectorAnalysis == nil &&
		target.ownedVectorAnalysis == nil &&
		target.recordAnalysis == nil
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

	parameter := method.Params[0].Sub.(*ast.BindingNode).Name
	for _, resultType := range []aotPrimitiveType{
		int64AOTPrimitive,
		boolAOTPrimitive,
	} {
		target := &aotSpecializationTarget{
			fn:           fn,
			arity:        1,
			directLinked: true,
		}
		analysis := &int64AOTAnalysis{
			target:             target,
			arity:              1,
			resultType:         resultType,
			uncheckedHostCalls: make(map[*ast.HostCallNode]bool),
			callees:            make(map[*aotSpecializationTarget]struct{}),
			allowCoreMod:       true,
		}
		analyzer := newInt64AOTAnalyzer(analysis, g.aotCallTargets)
		result := analyzer.exprType(
			method.Body,
			map[*lang.Symbol]aotPrimitiveType{
				parameter: int64AOTPrimitive,
			},
		)
		if result != resultType ||
			analysis.usesSelf || len(analysis.callees) != 0 {
			continue
		}
		return &int64CallableAOTAnalysis{
			primitive: analysis,
			result:    result,
		}
	}
	return nil
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

// generateTopLevelInt64CallableFixedFn keeps the ordinary FnFunc1 shape used
// by direct call slots while selecting the inferred primitive body for int64
// arguments. Other numeric types and redefined call sites retain the original
// generated function body.
func (g *Generator) generateTopLevelInt64CallableFixedFn(
	fnVar string,
	method *ast.FnMethodNode,
	analysis *int64CallableAOTAnalysis,
) {
	g.writef("%s = lang.FnFunc1(func(p0 any) any {\n", fnVar)
	parameter := g.allocateTempVar()
	ok := g.allocateTempVar()
	g.writef("if %s, %s := p0.(int64); %s {\n",
		parameter, ok, ok)
	g.writef("_ = %s\n", parameter)
	locals := map[*lang.Symbol]aotTypedLocal{
		method.Params[0].Sub.(*ast.BindingNode).Name: {
			name: parameter,
			typ:  int64AOTPrimitive,
		},
	}
	emitter := int64AOTEmitter{
		g:        g,
		analysis: analysis.primitive,
		callees:  map[*aotSpecializationTarget]string{},
	}
	result := emitter.emitExpr(method.Body, locals)
	g.writef("return %s\n", result)
	g.writef("}\n")
	g.generateFnMethodFixed(method, []string{"p0"})
	g.writef("})\n")
}
