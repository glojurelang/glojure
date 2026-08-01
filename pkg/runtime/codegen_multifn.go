//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

// Stable fixed-arity dispatchers can select exact methods directly. Literal
// vector results are decomposed into components so the otherwise unobservable
// vector need not be allocated; scalar dispatch is the one-component case.
// Any method, preference, or hierarchy change falls back to ordinary MultiFn
// dispatch.
func (g *Generator) prepareAOTMultiFnTargets(vars []namedVar) {
	if !g.directLink {
		return
	}
	for _, named := range vars {
		vr := named.vr
		if !vr.IsBound() || vr.IsMacro() || vr.IsDynamic() ||
			RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef)) {
			continue
		}
		multiFn, ok := codegenVarValue(vr).(*lang.MultiFn)
		if !ok || multiFn.IsProtocol() ||
			multiFn.PreferTable() == nil ||
			multiFn.PreferTable().Count() != 0 ||
			!aotMultiFnHierarchyEmpty(multiFn) {
			continue
		}
		dispatch, ok := multiFn.GetDispatchFn().(*Fn)
		if !ok {
			continue
		}
		dispatchPlan := compiler.AnalyzeFixedDispatchResult(dispatch.ASTNode())
		if dispatchPlan == nil {
			continue
		}
		arity := dispatchPlan.Method.FixedArity
		methods, defaultMethod := prepareAOTMultiFnMethods(
			multiFn,
			arity,
			dispatchPlan,
		)
		if len(methods) == 0 || defaultMethod == nil {
			continue
		}

		index := len(g.aotMultiFnCallTargets)
		target := &aotMultiFnCallTarget{
			vr:            vr,
			multiFn:       multiFn,
			multiFnVar:    fmt.Sprintf("aotMultiFn%d", index),
			generationVar: fmt.Sprintf("aotMultiFnGeneration%d", index),
			exactVar:      fmt.Sprintf("aotMultiFnExact%d", index),
			dispatchVar:   fmt.Sprintf("aotMultiFnDispatch%d", index),
			fastFnVar:     fmt.Sprintf("aotMultiFnFast%d", index),
			arity:         arity,
			dispatch:      dispatch,
			dispatchPlan:  dispatchPlan,
			methods:       methods,
			defaultMethod: defaultMethod,
		}
		for methodIndex, method := range append(
			append([]*aotMultiFnMethod(nil), methods...),
			defaultMethod,
		) {
			method.helperVar = fmt.Sprintf(
				"aotMultiFnMethod%dFor%d",
				methodIndex,
				index,
			)
			g.aotMultiFnMethods[method.fn] = append(
				g.aotMultiFnMethods[method.fn],
				method,
			)
		}
		g.aotMultiFnCallTargets[vr] = target
		g.aotMultiFnValues[multiFn] = target

		fmt.Fprintf(
			&g.aotDeclarations,
			"var %s *lang.MultiFn\nvar %s uint64\nvar %s bool\n",
			target.multiFnVar,
			target.generationVar,
			target.exactVar,
		)
		fmt.Fprintf(
			&g.aotDeclarations,
			"var %s func(%s) (%s)\n",
			target.dispatchVar,
			strings.Join(aotRepeatedType("any", arity), ", "),
			strings.Join(
				aotRepeatedType("any", len(dispatchPlan.Components)),
				", ",
			),
		)
		fmt.Fprintf(
			&g.aotDeclarations,
			"var %s func(%s) (any, bool)\n",
			target.fastFnVar,
			strings.Join(aotRepeatedType("any", arity), ", "),
		)
		for _, method := range append(
			append([]*aotMultiFnMethod(nil), methods...),
			defaultMethod,
		) {
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s lang.FnFunc%d\n",
				method.helperVar,
				arity,
			)
		}
	}
}

func aotRepeatedType(typ string, count int) []string {
	result := make([]string, count)
	for index := range result {
		result[index] = typ
	}
	return result
}

func aotMultiFnHierarchyEmpty(multiFn *lang.MultiFn) bool {
	hierarchy := multiFn.GetHierarchy().Deref()
	for _, key := range []string{"parents", "ancestors", "descendants"} {
		value := lang.Get(hierarchy, lang.NewKeyword(key))
		if !lang.IsNil(value) && lang.Count(value) != 0 {
			return false
		}
	}
	return true
}

func prepareAOTMultiFnMethods(
	multiFn *lang.MultiFn,
	arity int,
	plan *compiler.IRFixedDispatchResultPlan,
) ([]*aotMultiFnMethod, *aotMultiFnMethod) {
	var methods []*aotMultiFnMethod
	var defaultMethod *aotMultiFnMethod
	table := multiFn.GetMethodTable()
	if table == nil {
		return nil, nil
	}
	defaultValue := multiFn.GetDefaultDispatchVal()
	for seq := table.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(lang.IMapEntry)
		fn, ok := entry.Val().(*Fn)
		if !ok || !aotFnHasFixedArity(fn, arity) {
			return nil, nil
		}
		method := &aotMultiFnMethod{fn: fn}
		if lang.Equals(entry.Key(), defaultValue) {
			if defaultMethod != nil {
				return nil, nil
			}
			defaultMethod = method
			continue
		}
		if plan.Kind == compiler.IRDispatchScalar {
			method.components = []any{entry.Key()}
		} else {
			vector, ok := entry.Key().(lang.IPersistentVector)
			if !ok || vector.Count() != len(plan.Components) {
				return nil, nil
			}
			method.components = make([]any, len(plan.Components))
			for index := range method.components {
				method.components[index] = vector.Nth(index)
			}
		}
		methods = append(methods, method)
	}
	sort.Slice(methods, func(i, j int) bool {
		return lang.PrintString(
			lang.NewVector(methods[i].components...),
		) < lang.PrintString(
			lang.NewVector(methods[j].components...),
		)
	})
	return methods, defaultMethod
}

func aotFnHasFixedArity(fn *Fn, arity int) bool {
	if fn == nil {
		return false
	}
	node := fn.ASTNode().Sub.(*ast.FnNode)
	if node.IsVariadic || len(node.Methods) != 1 {
		return false
	}
	method := node.Methods[0].Sub.(*ast.FnMethodNode)
	return !method.IsVariadic && method.FixedArity == arity
}

func (g *Generator) generateAOTMultiFnFastPath(
	target *aotMultiFnCallTarget,
) {
	if target == nil {
		return
	}
	params := fixedParamNames(target.arity)
	signature := make([]string, target.arity)
	for index := range signature {
		signature[index] = params[index] + " any"
	}
	g.writef("%s = func(%s) (%s) {\n",
		target.dispatchVar,
		strings.Join(signature, ", "),
		strings.Join(
			aotRepeatedType("any", len(target.dispatchPlan.Components)),
			", ",
		),
	)
	g.pushVarScope()
	for index, parameter := range target.dispatchPlan.Method.Params {
		name := parameter.Sub.(*ast.BindingNode).Name
		local := g.allocateLocal(name.Name())
		g.writef("%s := %s\n", local, params[index])
		g.writef("_ = %s\n", local)
	}
	plainIR := g.currentIR
	g.currentIR = compiler.BuildTypedIR(target.dispatch.ASTNode())
	components := make([]string, len(target.dispatchPlan.Components))
	for index, component := range target.dispatchPlan.Components {
		components[index] = g.generateASTNode(component)
	}
	g.currentIR = plainIR
	g.writef("return %s\n", strings.Join(components, ", "))
	g.popVarScope()
	g.writef("}\n")

	g.writef("%s = func(%s) (any, bool) {\n",
		target.fastFnVar,
		strings.Join(signature, ", "),
	)
	g.writef("if !%s { return nil, false }\n",
		target.exactVar,
	)
	componentNames := make([]string, len(components))
	for index := range componentNames {
		componentNames[index] = g.allocateTempVar()
	}
	g.writef("%s := %s(%s)\n",
		strings.Join(componentNames, ", "),
		target.dispatchVar,
		strings.Join(params, ", "),
	)
	// Dispatch functions are arbitrary user code and may alter the method
	// table, preferences, or hierarchy themselves.
	g.writef("if !%s.IsGeneration(%s) { return nil, false }\n",
		target.multiFnVar,
		target.generationVar,
	)
	for _, method := range target.methods {
		conditions := make([]string, len(componentNames))
		for index, component := range method.components {
			conditions[index] = g.aotMultiFnMatchExpr(
				componentNames[index],
				component,
			)
		}
		g.writef("if %s { return %s(%s), true }\n",
			strings.Join(conditions, " && "),
			method.helperVar,
			strings.Join(params, ", "),
		)
	}
	g.writef("return %s(%s), true\n",
		target.defaultMethod.helperVar,
		strings.Join(params, ", "),
	)
	g.writef("}\n")
}

func (g *Generator) aotMultiFnMatchExpr(value string, expected any) string {
	switch expected := expected.(type) {
	case lang.Keyword:
		return fmt.Sprintf(
			"lang.EqualsKeyword(%s, %s)",
			value,
			g.generateValue(expected),
		)
	case int64:
		return fmt.Sprintf(
			"lang.EqualsInt64(%d, %s)",
			expected,
			value,
		)
	default:
		return fmt.Sprintf(
			"lang.Equiv(%s, %s)",
			value,
			g.generateValue(expected),
		)
	}
}

func (g *Generator) generateAOTMultiFnInvoke(
	node *ast.Node,
) (string, bool) {
	if !g.directLink || node == nil || node.Op != ast.OpInvoke {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	if invoke.Fn.Op != ast.OpVar {
		return "", false
	}
	target := g.aotMultiFnCallTargets[invoke.Fn.Sub.(*ast.VarNode).Var]
	if target == nil || len(invoke.Args) != target.arity {
		return "", false
	}
	varID := g.allocVarVar(
		target.vr.Namespace().Name().String(),
		target.vr.Symbol().String(),
	)
	if g.currentValueInit != nil && varID != g.currentValueInit.name {
		g.currentValueInit.deps[varID] = struct{}{}
	}
	args := make([]string, len(invoke.Args))
	for index, argument := range invoke.Args {
		args[index] = g.generateASTNode(argument)
	}
	result := g.allocateTempVar()
	ok := g.allocateTempVar()
	g.writef("// exact multimethod dispatch %s\n", target.vr)
	g.writef("%s, %s := %s(%s)\n",
		result,
		ok,
		target.fastFnVar,
		strings.Join(args, ", "),
	)
	g.writef("if !%s { %s = %s.Invoke%d(%s) }\n",
		ok,
		result,
		target.multiFnVar,
		target.arity,
		strings.Join(args, ", "),
	)
	return result, true
}
