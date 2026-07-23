//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func sortedAOTTargets(
	targets map[*aotSpecializationTarget]struct{},
) []*aotSpecializationTarget {
	result := make([]*aotSpecializationTarget, 0, len(targets))
	for target := range targets {
		result = append(result, target)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].directFnVar < result[j].directFnVar
	})
	return result
}

// prepareAOTCallTargets allocates package-level call slots for ordinary
// fixed-arity functions in the namespace. Generated callers can use the slot
// while its Var retains the root seen by LoadNS, and fall back to Var dispatch
// after a redefinition.
func (g *Generator) prepareAOTCallTargets(vars []namedVar) {
	for _, named := range vars {
		vr := named.vr
		if !vr.IsBound() || vr.IsMacro() ||
			RT.BooleanCast(lang.Get(vr.Meta(), lang.KWDynamic)) {
			continue
		}

		value := codegenVarValue(vr)
		fn, ok := value.(*Fn)
		if !ok {
			continue
		}
		fnNode := fn.ASTNode().Sub.(*ast.FnNode)
		if len(fnNode.Methods) != 1 || fnNode.IsVariadic {
			continue
		}
		method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		if method.IsVariadic || method.FixedArity > 4 {
			continue
		}

		index := len(g.aotCallTargets)
		target := &aotSpecializationTarget{
			vr:             vr,
			fn:             fn,
			arity:          method.FixedArity,
			directFnVar:    fmt.Sprintf("aotDirectFn%d", index),
			int64FnVar:     fmt.Sprintf("aotInt64Fn%d", index),
			float64FnVar:   fmt.Sprintf("aotFloat64Fn%d", index),
			rootVersionVar: fmt.Sprintf("aotRootVersion%d", index),
		}
		g.aotCallTargets[vr] = target
		fmt.Fprintf(
			&g.aotDeclarations,
			"var %s lang.FnFunc%d\nvar %s *lang.VarRootVersion\n",
			target.directFnVar,
			target.arity,
			target.rootVersionVar,
		)
	}
	for {
		changed := false
		for _, named := range vars {
			target := g.aotCallTargets[named.vr]
			if target == nil || target.int64Analysis != nil {
				continue
			}
			fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
			method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
			analysis := analyzeInt64AOTFunction(
				target,
				method,
				g.aotCallTargets,
			)
			if analysis == nil {
				continue
			}
			target.int64Analysis = analysis
			changed = true
		}
		if !changed {
			break
		}
	}
	for _, named := range vars {
		target := g.aotCallTargets[named.vr]
		if target == nil || target.int64Analysis == nil {
			continue
		}
		fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
		method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		target.int64Analysis.proveSafeOperations(method)
	}
	for {
		changed := false
		for _, named := range vars {
			target := g.aotCallTargets[named.vr]
			if target == nil || target.int64Analysis != nil ||
				target.float64Analysis != nil {
				continue
			}
			fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
			method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
			analysis := analyzeFloat64AOTFunction(
				target,
				method,
				g.aotCallTargets,
			)
			if analysis == nil {
				continue
			}
			target.float64Analysis = analysis
			changed = true
		}
		if !changed {
			break
		}
	}
	for _, named := range vars {
		target := g.aotCallTargets[named.vr]
		if target == nil {
			continue
		}
		if target.int64Analysis != nil {
			params := make([]string, target.arity)
			for i := range params {
				params[i] = "int64"
			}
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s func(%s) (int64, bool)\n",
				target.int64FnVar,
				strings.Join(params, ", "),
			)
		}
		if target.float64Analysis != nil {
			params := make([]string, target.arity)
			for i := range params {
				params[i] = "float64"
			}
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s func(%s) (float64, bool)\n",
				target.float64FnVar,
				strings.Join(params, ", "),
			)
		}
	}
	if len(g.aotCallTargets) > 0 {
		g.aotDeclarations.WriteByte('\n')
	}
}

func codegenVarValue(vr *lang.Var) any {
	// Dynamic Vars are resolved against goroutine-local bindings. Reading from
	// a fresh goroutine obtains the root value used to build an AOT loader.
	value := make(chan any)
	go func() {
		value <- vr.Get()
	}()
	return <-value
}

func (g *Generator) aotInvokeTarget(
	invoke *ast.InvokeNode,
) *aotSpecializationTarget {
	if invoke.Fn.Op != ast.OpVar {
		return nil
	}
	target := g.aotCallTargets[invoke.Fn.Sub.(*ast.VarNode).Var]
	if target == nil || target.arity != len(invoke.Args) {
		return nil
	}
	return target
}

// aotExternalInvokeTarget caches the root of a statically resolved call into
// another namespace. Compiled calls use the root present when the namespace
// loads while its version remains current, and fall back to Var dispatch after
// a redefinition. Dynamic Vars always retain runtime lookup.
func (g *Generator) aotExternalInvokeTarget(
	invoke *ast.InvokeNode,
) *aotExternalCallTarget {
	if invoke.Fn.Op != ast.OpVar {
		return nil
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr.Namespace() == g.aotNamespace ||
		!vr.IsBound() ||
		vr.IsMacro() ||
		vr.IsDynamic() {
		return nil
	}
	arity := len(invoke.Args)
	if arity > 5 || !aotSupportsArity(codegenVarValue(vr), arity) {
		return nil
	}
	key := aotExternalCallKey{vr: vr, arity: arity}
	if target := g.aotExternalCallTargets[key]; target != nil {
		return target
	}
	index := len(g.aotExternalCallTargets)
	target := &aotExternalCallTarget{
		vr:    vr,
		arity: arity,
		fnVar: fmt.Sprintf("aotExternalFn%d", index),
	}
	g.allocVarVar(
		vr.Namespace().Name().String(),
		vr.Symbol().String(),
	)
	g.aotExternalCallTargets[key] = target
	return target
}

func (g *Generator) generateAOTExternalCacheAdapters() {
	var arities [6]bool
	for _, target := range g.aotExternalCallTargets {
		arities[target.arity] = true
	}
	for arity, used := range arities {
		if !used {
			continue
		}
		params := make([]string, arity)
		args := make([]string, arity)
		for i := range params {
			args[i] = fmt.Sprintf("p%d", i)
			params[i] = args[i] + " any"
		}
		paramList := strings.Join(params, ", ")
		argList := strings.Join(args, ", ")
		fmt.Fprintf(&g.aotDeclarations,
			"func aotCacheFn%d(vr *lang.Var) lang.FnFunc%d {\n"+
				"version := vr.RootVersion()\n"+
				"fn := checkDerefVar(vr)\n",
			arity, arity)
		fmt.Fprintf(&g.aotDeclarations,
			"if direct, ok := fn.(lang.FnFunc%d); ok {\n"+
				"return func(%s) any {\n"+
				"if vr.RootVersion() == version { return direct(%s) }\n"+
				"return lang.Apply%d(checkDerefVar(vr)%s)\n"+
				"}\n"+
				"}\n",
			arity, paramList, argList, arity, aotAdapterArgs(args))
		fmt.Fprintf(&g.aotDeclarations,
			"if fixed, ok := fn.(lang.FixedArityFn%d); ok {\n"+
				"return func(%s) any {\n"+
				"if vr.RootVersion() == version { return fixed.Invoke%d(%s) }\n"+
				"return lang.Apply%d(checkDerefVar(vr)%s)\n"+
				"}\n"+
				"}\n",
			arity, paramList, arity, argList, arity, aotAdapterArgs(args))
		fmt.Fprintf(&g.aotDeclarations,
			"return func(%s) any {\n"+
				"if vr.RootVersion() == version { return lang.Apply%d(fn%s) }\n"+
				"return lang.Apply%d(checkDerefVar(vr)%s)\n"+
				"}\n"+
				"}\n\n",
			paramList,
			arity, aotAdapterArgs(args),
			arity, aotAdapterArgs(args))
	}
}

func aotAdapterArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

func aotSupportsArity(value any, arity int) bool {
	switch arity {
	case 0:
		_, ok := value.(lang.FixedArityFn0)
		return ok
	case 1:
		_, ok := value.(lang.FixedArityFn1)
		return ok
	case 2:
		_, ok := value.(lang.FixedArityFn2)
		return ok
	case 3:
		_, ok := value.(lang.FixedArityFn3)
		return ok
	case 4:
		_, ok := value.(lang.FixedArityFn4)
		return ok
	case 5:
		_, ok := value.(lang.FixedArityFn5)
		return ok
	default:
		return false
	}
}
