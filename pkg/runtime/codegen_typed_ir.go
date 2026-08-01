//go:build !glj_aot_runtime

package runtime

import (
	"reflect"
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
		facts.Type.Kind != compiler.IRFloat &&
		facts.Type.Kind != compiler.IRBool {
		return "", false
	}

	args := make([]string, len(call.Args))
	types := make([]compiler.IRValueKind, len(call.Args))
	for i, argument := range call.Args {
		types[i] = g.currentIR.Facts(argument).Type.Kind
		if types[i] != compiler.IRInt &&
			types[i] != compiler.IRFloat {
			return "", false
		}
		code := g.generateASTNode(argument)
		if facts.Type.Kind == compiler.IRFloat ||
			facts.Type.Kind == compiler.IRBool &&
				types[i] == compiler.IRFloat {
			args[i] = g.irFloat64Expr(argument, code)
		} else {
			args[i] = g.irInt64Expr(argument, code)
		}
	}

	var expression string
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		switch name {
		case "inc":
			if facts.Type.Kind == compiler.IRFloat {
				expression = "(" + args[0] + " + 1)"
			} else {
				expression = "lang.CheckedAddInt64(" + args[0] + ", 1)"
			}
		case "unchecked_inc":
			expression = "(" + args[0] + " + 1)"
		case "dec":
			if facts.Type.Kind == compiler.IRFloat {
				expression = "(" + args[0] + " - 1)"
			} else {
				expression = "lang.CheckedSubInt64(" + args[0] + ", 1)"
			}
		case "uncheckeddec", "unchecked_dec":
			expression = "(" + args[0] + " - 1)"
		case "minus":
			if facts.Type.Kind == compiler.IRFloat {
				expression = "(-" + args[0] + ")"
			} else {
				expression = "lang.CheckedNegateInt64(" + args[0] + ")"
			}
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
		if facts.Type.Kind == compiler.IRBool && types[0] != types[1] {
			// Clojure's mixed numeric comparisons preserve integer precision
			// that a conversion to float64 could lose.
			return "", false
		}
		switch name {
		case "add":
			if facts.Type.Kind == compiler.IRFloat {
				expression = "(" + args[0] + " + " + args[1] + ")"
			} else {
				expression = "lang.CheckedAddInt64(" +
					args[0] + ", " + args[1] + ")"
			}
		case "uncheckedadd":
			expression = "(" + args[0] + " + " + args[1] + ")"
		case "minus":
			if facts.Type.Kind == compiler.IRFloat {
				expression = "(" + args[0] + " - " + args[1] + ")"
			} else {
				expression = "lang.CheckedSubInt64(" +
					args[0] + ", " + args[1] + ")"
			}
		case "unchecked_minus":
			expression = "(" + args[0] + " - " + args[1] + ")"
		case "multiply":
			if facts.Type.Kind == compiler.IRFloat {
				expression = "(" + args[0] + " * " + args[1] + ")"
			} else {
				expression = "lang.CheckedMultiplyInt64(" +
					args[0] + ", " + args[1] + ")"
			}
		case "unchecked_multiply":
			expression = "(" + args[0] + " * " + args[1] + ")"
		case "divide":
			expression = "(" + args[0] + " / " + args[1] + ")"
		case "quotient":
			expression = "(" + args[0] + " / " + args[1] + ")"
		case "remainder":
			expression = "(" + args[0] + " % " + args[1] + ")"
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

func (g *Generator) generateIRCoreNumericInvoke(
	node *ast.Node,
) (string, bool) {
	if !g.directLink || g.currentIR == nil || node == nil ||
		node.Op != ast.OpInvoke {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	if !facts.Call.Known || facts.Call.Var == nil ||
		facts.Call.Var.Namespace() == nil ||
		facts.Call.Var.Namespace().Name().String() != "clojure.core" ||
		facts.Call.Name != "mod" ||
		facts.Call.Var.IsDynamic() ||
		RT.BooleanCast(lang.Get(facts.Call.Var.Meta(), lang.KWRedef)) ||
		!IsDefaultCoreVar(facts.Call.Var) {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	if len(invoke.Args) != 2 ||
		g.currentIR.Facts(invoke.Args[0]).Type.Kind != compiler.IRInt ||
		g.currentIR.Facts(invoke.Args[1]).Type.Kind != compiler.IRInt {
		return "", false
	}
	leftCode := g.generateASTNode(invoke.Args[0])
	rightCode := g.generateASTNode(invoke.Args[1])
	left := g.irInt64Expr(invoke.Args[0], leftCode)
	right := g.irInt64Expr(invoke.Args[1], rightCode)
	result := g.allocateTempVar()
	g.writef("%s := lang.ModInt64(%s, %s)\n", result, left, right)
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
	if len(call.Args) < 2 || len(call.Args) > 3 {
		return "", false
	}
	mode := g.currentIR.Facts(node).OwnedMapMode
	if mode != compiler.IROwnedMapAdaptive &&
		g.currentIR.Facts(call.Args[1]).Type.Kind != compiler.IRString {
		return "", false
	}
	target := g.generateASTNode(call.Args[0])
	keyCode := g.generateASTNode(call.Args[1])
	fallback := "nil"
	if len(call.Args) == 3 {
		fallback = g.generateASTNode(call.Args[2])
	}
	result := g.allocateTempVar()
	if mode == compiler.IROwnedMapAdaptive {
		g.writef(
			"%s := %s.(*runtime.OwnedLoopMap).ValAtDefault(%s, %s)\n",
			result,
			target,
			keyCode,
			fallback,
		)
		return result, true
	}
	key := g.irStringExpr(call.Args[1], keyCode)
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

func (g *Generator) irFloat64Expr(node *ast.Node, code string) string {
	if g.irHasFloat64Representation(node) {
		return code
	}
	if g.irHasInt64Representation(node) {
		return "float64(" + code + ")"
	}
	return "lang.AsFloat64(" + code + ")"
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
	case ast.OpDo:
		if g.currentIR == nil ||
			g.currentIR.Facts(node).Type.Kind != compiler.IRInt {
			return false
		}
		return g.irHasInt64Representation(
			node.Sub.(*ast.DoNode).Ret,
		)
	case ast.OpLet, ast.OpLoop:
		if g.currentIR == nil ||
			g.currentIR.Facts(node).Type.Kind != compiler.IRInt {
			return false
		}
		return g.irHasInt64Representation(
			node.Sub.(*ast.LetNode).Body,
		)
	case ast.OpIf:
		if g.currentIR == nil ||
			g.currentIR.Facts(node).Type.Kind != compiler.IRInt {
			return false
		}
		ifNode := node.Sub.(*ast.IfNode)
		return g.irInt64BranchHasRepresentation(ifNode.Then) &&
			g.irInt64BranchHasRepresentation(ifNode.Else)
	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		if call.Target == nil || call.Target.Op != ast.OpConst ||
			call.ResolvedMethod == nil {
			return false
		}
		return call.Target.Sub.(*ast.ConstNode).Value == lang.Numbers &&
			g.currentIR.Facts(node).Type.Kind == compiler.IRInt &&
			irNumbersHostCallHasPrimitiveRepresentation(call)
	case ast.OpInvoke:
		invoke := node.Sub.(*ast.InvokeNode)
		if invoke.Fn != nil && invoke.Fn.Op == ast.OpConst {
			callable := invoke.Fn.Sub.(*ast.ConstNode).Value
			typ := reflect.TypeOf(callable)
			return typ != nil && typ.Kind() == reflect.Func &&
				typ.NumOut() == 1 &&
				typ.Out(0) == reflect.TypeFor[int64]()
		}
		if !g.directLink || g.currentIR == nil {
			return false
		}
		facts := g.currentIR.Facts(node)
		if facts.Signature != nil {
			return facts.Signature.Result.Kind == compiler.IRInt &&
				facts.Signature.Result.GoType == reflect.TypeFor[int64]()
		}
		return facts.Type.Kind == compiler.IRInt &&
			facts.Call.Known &&
			facts.Call.Name == "mod" &&
			facts.Call.Var != nil &&
			facts.Call.Var.Namespace() != nil &&
			facts.Call.Var.Namespace().Name().String() == "clojure.core" &&
			!facts.Call.Var.IsDynamic() &&
			!RT.BooleanCast(lang.Get(facts.Call.Var.Meta(), lang.KWRedef))
	case ast.OpCase:
		if g.currentIR == nil ||
			g.currentIR.Facts(node).Type.Kind != compiler.IRInt {
			return false
		}
		caseNode := node.Sub.(*ast.CaseNode)
		for _, entry := range caseNode.Entries {
			if !g.irInt64BranchHasRepresentation(entry.ResultExpr) {
				return false
			}
		}
		return caseNode.Default == nil ||
			g.irInt64BranchHasRepresentation(caseNode.Default)
	default:
		return false
	}
}

func (g *Generator) irHasIntRepresentation(node *ast.Node) bool {
	if node == nil || g.currentIR == nil ||
		g.currentIR.Facts(node).Type.GoType != reflect.TypeFor[int]() {
		return false
	}
	switch node.Op {
	case ast.OpConst:
		_, ok := node.Sub.(*ast.ConstNode).Value.(int)
		return ok
	case ast.OpDo:
		return g.irHasIntRepresentation(node.Sub.(*ast.DoNode).Ret)
	case ast.OpLet, ast.OpLoop:
		return g.irHasIntRepresentation(node.Sub.(*ast.LetNode).Body)
	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		return g.irIntBranchHasRepresentation(ifNode.Then) &&
			g.irIntBranchHasRepresentation(ifNode.Else)
	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		return call.ResolvedMethod != nil
	case ast.OpInvoke:
		facts := g.currentIR.Facts(node)
		invoke := node.Sub.(*ast.InvokeNode)
		if invoke.Fn != nil && invoke.Fn.Op == ast.OpConst {
			callable := invoke.Fn.Sub.(*ast.ConstNode).Value
			typ := reflect.TypeOf(callable)
			return typ != nil && typ.Kind() == reflect.Func &&
				typ.NumOut() == 1 && typ.Out(0) == reflect.TypeFor[int]()
		}
		return g.directLink && facts.Call.Known &&
			facts.Call.Var != nil && facts.Call.Var.Namespace() != nil &&
			facts.Call.Var.Namespace().Name().String() == "clojure.core" &&
			facts.Call.Name == "count" &&
			!facts.Call.Var.IsDynamic() &&
			!RT.BooleanCast(lang.Get(facts.Call.Var.Meta(), lang.KWRedef))
	case ast.OpCase:
		caseNode := node.Sub.(*ast.CaseNode)
		for _, entry := range caseNode.Entries {
			if !g.irIntBranchHasRepresentation(entry.ResultExpr) {
				return false
			}
		}
		return caseNode.Default == nil ||
			g.irIntBranchHasRepresentation(caseNode.Default)
	default:
		return false
	}
}

func (g *Generator) irIntBranchHasRepresentation(node *ast.Node) bool {
	if node == nil || g.currentIR == nil {
		return false
	}
	return g.currentIR.Facts(node).NeverReturns ||
		g.irHasIntRepresentation(node)
}

func (g *Generator) irInt64BranchHasRepresentation(node *ast.Node) bool {
	if node == nil || g.currentIR == nil {
		return false
	}
	return g.currentIR.Facts(node).NeverReturns ||
		g.irHasInt64Representation(node)
}

func (g *Generator) irHasFloat64Representation(node *ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Op {
	case ast.OpConst:
		_, ok := node.Sub.(*ast.ConstNode).Value.(float64)
		return ok
	case ast.OpLocal:
		name := node.Sub.(*ast.LocalNode).Name
		return name != nil && g.getLocalType(name.Name()) == compiler.IRFloat
	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		if call.Target == nil || call.Target.Op != ast.OpConst ||
			call.ResolvedMethod == nil {
			return false
		}
		return call.Target.Sub.(*ast.ConstNode).Value == lang.Numbers &&
			g.currentIR.Facts(node).Type.Kind == compiler.IRFloat
	case ast.OpInvoke:
		if !g.directLink || g.currentIR == nil {
			return false
		}
		signature := g.currentIR.Facts(node).Signature
		return signature != nil &&
			signature.Result.Kind == compiler.IRFloat &&
			signature.Result.GoType == reflect.TypeFor[float64]()
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

func (g *Generator) generateIROwnedStringParts(
	node *ast.Node,
) (string, bool) {
	if g.currentIR == nil {
		return "", false
	}
	facts := g.currentIR.Facts(node)
	if facts.StringPartsAppend != nil {
		parts, ok := g.getOwnedStringParts(
			facts.StringPartsAppend.Parts.Name(),
		)
		if !ok {
			return "", false
		}
		value := g.generateASTNode(facts.StringPartsAppend.Value)
		g.writef("%s = append(%s, %s)\n", parts, parts, value)
		return parts, true
	}
	if facts.StringPartsFinish == nil {
		return "", false
	}
	parts, ok := g.getOwnedStringParts(
		facts.StringPartsFinish.Parts.Name(),
	)
	if !ok {
		return "", false
	}
	result := g.allocateTempVar()
	g.writef("%s := runtime.ConcatStringParts(%s)\n", result, parts)
	return result, true
}

func (g *Generator) irOwnedStringPartsEnabled() bool {
	if !g.directLink {
		return false
	}
	for _, name := range []string{"apply", "conj", "str"} {
		vr := lang.NSCore.FindInternedVar(lang.NewSymbol(name))
		if !aotVarCanDirectLink(vr) || !IsDefaultCoreVar(vr) {
			return false
		}
	}
	return true
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
