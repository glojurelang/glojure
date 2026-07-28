//go:build !glj_aot_runtime

package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (g *Generator) generateIRNumbersHostCall(
	node *ast.Node,
) (string, bool) {
	if g.currentIR == nil || node == nil || node.Op != ast.OpHostCall {
		return "", false
	}
	call := node.Sub.(*ast.HostCallNode)
	if call.ResolvedMethod == nil || call.Target == nil ||
		call.Target.Op != ast.OpConst {
		return "", false
	}
	target := call.Target.Sub.(*ast.ConstNode)
	if target.Value != lang.Numbers {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	if facts.Type.Kind != compiler.IRInt &&
		facts.Type.Kind != compiler.IRBool {
		return "", false
	}

	args := make([]string, len(call.Args))
	for i, argument := range call.Args {
		if g.currentIR.Facts(argument).Type.Kind != compiler.IRInt {
			return "", false
		}
		code := g.generateASTNode(argument)
		args[i] = g.irInt64Expr(argument, code)
	}

	var expression string
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		switch name {
		case "inc":
			expression = "lang.CheckedAddInt64(" + args[0] + ", 1)"
		case "unchecked_inc":
			expression = "(" + args[0] + " + 1)"
		case "dec":
			expression = "lang.CheckedSubInt64(" + args[0] + ", 1)"
		case "uncheckeddec", "unchecked_dec":
			expression = "(" + args[0] + " - 1)"
		case "minus":
			expression = "lang.CheckedNegateInt64(" + args[0] + ")"
		case "unchecked_minus":
			expression = "(-" + args[0] + ")"
		case "iszero":
			expression = "(" + args[0] + " == 0)"
		case "ispos":
			expression = "(" + args[0] + " > 0)"
		case "isneg":
			expression = "(" + args[0] + " < 0)"
		}
	}
	if len(args) == 2 {
		switch name {
		case "add":
			expression = "lang.CheckedAddInt64(" +
				args[0] + ", " + args[1] + ")"
		case "uncheckedadd":
			expression = "(" + args[0] + " + " + args[1] + ")"
		case "minus":
			expression = "lang.CheckedSubInt64(" +
				args[0] + ", " + args[1] + ")"
		case "unchecked_minus":
			expression = "(" + args[0] + " - " + args[1] + ")"
		case "multiply":
			expression = "lang.CheckedMultiplyInt64(" +
				args[0] + ", " + args[1] + ")"
		case "unchecked_multiply":
			expression = "(" + args[0] + " * " + args[1] + ")"
		case "lt":
			expression = "(" + args[0] + " < " + args[1] + ")"
		case "lte":
			expression = "(" + args[0] + " <= " + args[1] + ")"
		case "gt":
			expression = "(" + args[0] + " > " + args[1] + ")"
		case "gte":
			expression = "(" + args[0] + " >= " + args[1] + ")"
		}
	}
	if expression == "" {
		return "", false
	}
	result := g.allocateTempVar()
	g.writef("%s := %s\n", result, expression)
	return result, true
}

func (g *Generator) generateIROwnedMapHostCall(
	node *ast.Node,
) (string, bool) {
	if g.currentIR == nil || node == nil || node.Op != ast.OpHostCall ||
		!g.currentIR.Facts(node).OwnedMapGet {
		return "", false
	}
	call := node.Sub.(*ast.HostCallNode)
	if len(call.Args) < 2 || len(call.Args) > 3 ||
		g.currentIR.Facts(call.Args[1]).Type.Kind != compiler.IRString {
		return "", false
	}
	target := g.generateASTNode(call.Args[0])
	keyCode := g.generateASTNode(call.Args[1])
	key := g.irStringExpr(call.Args[1], keyCode)
	fallback := "nil"
	if len(call.Args) == 3 {
		fallback = g.generateASTNode(call.Args[2])
	}
	result := g.allocateTempVar()
	g.writef(
		"%s := %s.(*lang.TransientMap).ValAtStringDefault(%s, %s)\n",
		result,
		target,
		key,
		fallback,
	)
	return result, true
}

func (g *Generator) irInt64Expr(node *ast.Node, code string) string {
	if g.irHasInt64Representation(node) {
		return code
	}
	return "lang.AsInt64(" + code + ")"
}

func (g *Generator) irStringExpr(node *ast.Node, code string) string {
	if g.irHasStringRepresentation(node) {
		return code
	}
	return "any(" + code + ").(string)"
}

func (g *Generator) irHasStringRepresentation(node *ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Op {
	case ast.OpConst:
		_, ok := node.Sub.(*ast.ConstNode).Value.(string)
		return ok
	case ast.OpLocal:
		name := node.Sub.(*ast.LocalNode).Name
		return name != nil &&
			g.getLocalType(name.Name()) == compiler.IRString
	case ast.OpInvoke:
		if !g.directLink {
			return false
		}
		facts := g.currentIR.Facts(node)
		if !facts.Call.Known || facts.Call.Name != "subs" ||
			facts.Call.Var == nil ||
			facts.Call.Var.Namespace().Name().String() != "clojure.core" ||
			facts.Call.Var.IsDynamic() ||
			RT.BooleanCast(lang.Get(facts.Call.Var.Meta(), lang.KWRedef)) {
			return false
		}
		return true
	default:
		return false
	}
}

func (g *Generator) irHasInt64Representation(node *ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Op {
	case ast.OpConst:
		_, ok := node.Sub.(*ast.ConstNode).Value.(int64)
		return ok
	case ast.OpLocal:
		name := node.Sub.(*ast.LocalNode).Name
		return name != nil && g.getLocalType(name.Name()) == compiler.IRInt
	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		if call.Target == nil || call.Target.Op != ast.OpConst ||
			call.ResolvedMethod == nil {
			return false
		}
		return call.Target.Sub.(*ast.ConstNode).Value == lang.Numbers &&
			g.currentIR.Facts(node).Type.Kind == compiler.IRInt &&
			irNumbersHostCallHasPrimitiveRepresentation(call)
	default:
		return false
	}
}

func irNumbersHostCallHasPrimitiveRepresentation(
	call *ast.HostCallNode,
) bool {
	if call == nil || call.Method == nil {
		return false
	}
	name := strings.ToLower(call.Method.Name())
	switch len(call.Args) {
	case 1:
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec",
			"unchecked_dec", "minus", "unchecked_minus":
			return true
		}
	case 2:
		switch name {
		case "add", "uncheckedadd", "minus", "unchecked_minus",
			"multiply", "unchecked_multiply":
			return true
		}
	}
	return false
}

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
