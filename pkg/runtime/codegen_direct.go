//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

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
	for _, named := range vars {
		target := g.aotCallTargets[named.vr]
		if target == nil || target.int64Analysis == nil {
			continue
		}
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
