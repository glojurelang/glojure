//go:build !glj_aot_runtime

package runtime

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (g *Generator) generateIRGetIn(node *ast.Node) (string, bool) {
	if !g.directLink || g.currentIR == nil {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	plan := facts.GetIn
	if plan == nil || facts.Call.Var == nil ||
		facts.Call.Var.IsMacro() || facts.Call.Var.IsDynamic() ||
		RT.BooleanCast(lang.Get(facts.Call.Var.Meta(), lang.KWRedef)) {
		return "", false
	}

	result := g.generateASTNode(plan.Target)
	keys := make([]string, len(plan.Keys))
	for i, keyNode := range plan.Keys {
		keys[i] = g.generateASTNode(keyNode)
	}
	for _, key := range keys {
		next := g.allocateTempVar()
		g.writef("%s := lang.Get(%s, %s)\n", next, result, key)
		result = next
	}
	return result, true
}

func (g *Generator) generateIRSwap(node *ast.Node) (string, bool) {
	if !g.directLink || g.currentIR == nil {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	plan := facts.Swap
	if plan == nil || facts.Call.Var == nil ||
		facts.Call.Var.IsMacro() || facts.Call.Var.IsDynamic() ||
		RT.BooleanCast(lang.Get(facts.Call.Var.Meta(), lang.KWRedef)) ||
		astContainsOp(plan.Callback, ast.OpRecur) {
		return "", false
	}

	target := g.generateASTNode(plan.Target)
	result := g.allocateTempVar()
	directAtom := g.allocateTempVar()
	g.writef("var %s any\n", result)
	g.writef("if %s, ok := %s.(*lang.Atom); ok {\n", directAtom, target)
	param := g.allocateTempVar()
	g.writef("%s = %s.SwapFunc(func(%s any) any {\n",
		result, directAtom, param)
	body, ok := g.generateImmediateFnBody(plan.Callback, []string{param})
	if !ok {
		panic("typed IR swap plan lost its fixed callback")
	}
	g.writef("return %s\n", body)
	g.writef("})\n")
	g.writef("} else {\n")
	callback := g.generateASTNode(plan.Callback)
	g.writef("%s = runtime.DirectSwap0(%s, %s)\n",
		result, target, callback)
	g.writef("}\n")
	return result, true
}

func (g *Generator) generateIRStringStack(node *ast.Node) (string, bool) {
	if g.currentIR == nil {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	if facts.Append != nil {
		stack, ok := g.getLocalStringStack(facts.Append.Stack.Name())
		if !ok {
			return "", false
		}
		value := g.generateASTNode(facts.Append.Value)
		g.writef("%s = append(%s, runtime.CoreString(%s))\n",
			stack, stack, value)
		return stack, true
	}
	if facts.Join == nil {
		return "", false
	}
	stack, ok := g.getLocalStringStack(facts.Join.Stack.Name())
	if !ok {
		return "", false
	}

	separator := g.generateASTNode(facts.Join.Separator)
	head := g.generateASTNode(facts.Join.Head)
	parts := g.allocateTempVar()
	g.writef("%s := make([]string, len(%s)+1)\n", parts, stack)
	g.writef("%s[0] = runtime.CoreString(%s)\n", parts, head)
	index := g.allocateTempVar()
	g.writef("for %s := range %s {\n", index, stack)
	g.writef("%s[%s+1] = %s[len(%s)-1-%s]\n",
		parts, index, stack, stack, index)
	g.writef("}\n")
	stringsPackage := g.addImportWithAlias("strings")
	result := g.allocateTempVar()
	g.writef("%s := %s.Join(%s, any(%s).(string))\n",
		result, stringsPackage, parts, separator)
	return result, true
}

func (g *Generator) irScalarAtomInit(binding *ast.Node) *ast.Node {
	if g.currentIR == nil {
		return nil
	}
	facts := g.currentIR.BindingFacts(binding)
	if facts.Escape != compiler.IRDoesNotEscape || facts.AtomInit == nil {
		return nil
	}
	for _, name := range []string{"atom", "deref", "reset!", "swap!"} {
		vr := lang.NSCore.FindInternedVar(lang.NewSymbol(name))
		if vr == nil || !IsDefaultCoreVar(vr) {
			return nil
		}
	}
	return facts.AtomInit
}

func irStaticKeywordMapNames(facts compiler.IRFacts) ([]string, bool) {
	if facts.Shape.Kind != compiler.IRShapeKeywordMap ||
		len(facts.Shape.Keywords) != facts.Shape.Count {
		return nil, false
	}
	names := make([]string, len(facts.Shape.Keywords))
	for i, keyword := range facts.Shape.Keywords {
		names[i] = keywordName(keyword)
	}
	return names, true
}
