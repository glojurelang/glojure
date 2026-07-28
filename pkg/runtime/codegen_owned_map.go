//go:build !glj_aot_runtime

package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (g *Generator) generateIROwnedMapReduce(
	invoke *ast.InvokeNode,
	plan *compiler.IROwnedMapReducePlan,
) (string, bool) {
	if !g.directLink || !aotVarCanDirectLink(plan.ReduceVar) {
		return "", false
	}
	for _, vr := range plan.UpdateInVars {
		if !aotVarCanDirectLink(vr) {
			return "", false
		}
	}
	reduceTarget := g.aotExternalInvokeTarget(invoke)
	if reduceTarget == nil || !reduceTarget.directLinked {
		return "", false
	}

	previousUpdates := g.ownedMapUpdates
	g.ownedMapUpdates = make(
		map[*ast.Node]*compiler.IROwnedMapUpdatePlan,
		len(plan.Updates),
	)
	for _, update := range plan.Updates {
		g.ownedMapUpdates[update] =
			g.currentIR.Facts(update).OwnedMapUpdateIn
	}
	reducer := g.generateASTNode(plan.Reducer)
	g.ownedMapUpdates = previousUpdates

	initial := g.generateASTNode(plan.Initial)
	source := g.generateASTNode(plan.Source)
	result := g.allocateTempVar()
	g.writef(
		"%s := runtime.ReduceOwnedMap(%s, %s, %s, %s)\n",
		result,
		reduceTarget.fnVar,
		reducer,
		initial,
		source,
	)
	return result, true
}

func (g *Generator) generateIROwnedMapUpdateIn(
	node *ast.Node,
) (string, bool) {
	plan := g.ownedMapUpdates[node]
	if g.ownedMapUpdates == nil || plan == nil ||
		node == nil || node.Op != ast.OpInvoke {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	if len(invoke.Args) < 3 {
		return "", false
	}
	if len(plan.Keys) == 2 &&
		(len(invoke.Args) == 3 || len(invoke.Args) == 4) {
		arguments := []string{
			g.generateASTNode(invoke.Args[0]),
			g.generateASTNode(plan.Keys[0]),
			g.generateASTNode(plan.Keys[1]),
		}
		helper := "runtime.UpdateOwnedMapPath2_3"
		if plan.Fnil != nil && aotVarCanDirectLink(plan.Fnil.Var) {
			arguments = append(
				arguments,
				g.generateASTNode(plan.Fnil.Fn),
				g.generateASTNode(plan.Fnil.Default),
			)
			helper = "runtime.UpdateOwnedMapPath2Default3"
		} else {
			arguments = append(
				arguments,
				g.generateASTNode(invoke.Args[2]),
			)
		}
		if len(invoke.Args) == 4 {
			arguments = append(
				arguments,
				g.generateASTNode(invoke.Args[3]),
			)
			if plan.Fnil != nil &&
				aotVarCanDirectLink(plan.Fnil.Var) {
				helper = "runtime.UpdateOwnedMapPath2Default4"
			} else {
				helper = "runtime.UpdateOwnedMapPath2_4"
			}
		}
		result := g.allocateTempVar()
		g.writef("%s := %s(%s)\n",
			result, helper, strings.Join(arguments, ", "))
		return result, true
	}
	arguments := make([]string, len(invoke.Args))
	for index, argument := range invoke.Args {
		arguments[index] = g.generateASTNode(argument)
	}
	helper := "runtime.UpdateOwnedMap"
	switch len(arguments) {
	case 3:
		helper = "runtime.UpdateOwnedMap3"
	case 4:
		helper = "runtime.UpdateOwnedMap4"
	}
	result := g.allocateTempVar()
	g.writef("%s := %s(%s)\n",
		result, helper, strings.Join(arguments, ", "))
	return result, true
}

func aotVarCanDirectLink(vr *lang.Var) bool {
	return vr != nil && vr.IsBound() && !vr.IsMacro() && !vr.IsDynamic() &&
		!RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef))
}
