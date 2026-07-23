//go:build !glj_aot_runtime

package runtime

import (
	"reflect"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

var astNodePointerType = reflect.TypeOf((*ast.Node)(nil))
var astPackagePath = reflect.TypeOf(ast.Node{}).PkgPath()

// scalarReplaceableAtomInit returns the initial value of a lexical atom when
// every use is a synchronous deref, reset!, or swap! operation. Such an atom's
// identity and synchronization machinery are unobservable, so AOT code can
// represent its state as an ordinary Go local.
func scalarReplaceableAtomInit(
	binding *ast.BindingNode,
	laterBindings []*ast.Node,
	body *ast.Node,
) *ast.Node {
	init := binding.Init
	if init == nil || init.Op != ast.OpInvoke {
		return nil
	}
	invoke := init.Sub.(*ast.InvokeNode)
	if !isCoreInvoke(invoke, "atom") || len(invoke.Args) != 1 {
		return nil
	}
	for _, name := range []string{"atom", "deref", "reset!", "swap!"} {
		vr := lang.NSCore.FindInternedVar(lang.NewSymbol(name))
		if vr == nil || !IsDefaultCoreVar(vr) {
			return nil
		}
	}

	usage := localAtomUsage{target: binding.Name, safe: true}
	for _, later := range laterBindings {
		usage.walkNode(later.Sub.(*ast.BindingNode).Init, false)
	}
	usage.walkNode(body, false)
	if !usage.safe || usage.uses == 0 {
		return nil
	}
	return invoke.Args[0]
}

type localAtomUsage struct {
	target *lang.Symbol
	safe   bool
	uses   int
}

func (u *localAtomUsage) walkNode(node *ast.Node, inFunction bool) {
	if node == nil || !u.safe {
		return
	}
	if u.isTargetLocal(node) {
		u.uses++
		u.safe = false
		return
	}
	if node.Op == ast.OpFn || node.Op == ast.OpFnMethod || node.Op == ast.OpGo {
		u.walkValue(reflect.ValueOf(node.Sub), true)
		return
	}
	if node.Op == ast.OpInvoke {
		invoke := node.Sub.(*ast.InvokeNode)
		if operation := localAtomOperation(invoke); operation != "" &&
			len(invoke.Args) > 0 && u.isTargetLocal(invoke.Args[0]) {
			u.uses++
			if inFunction {
				u.safe = false
				return
			}
			u.walkNode(invoke.Fn, inFunction)
			for _, arg := range invoke.Args[1:] {
				u.walkNode(arg, inFunction)
			}
			return
		}
	}
	u.walkValue(reflect.ValueOf(node.Sub), inFunction)
}

func (u *localAtomUsage) walkValue(value reflect.Value, inFunction bool) {
	if !value.IsValid() || !u.safe {
		return
	}
	if value.Type() == astNodePointerType {
		if !value.IsNil() {
			u.walkNode(value.Interface().(*ast.Node), inFunction)
		}
		return
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !value.IsNil() {
			element := value.Elem()
			if element.Kind() == reflect.Struct &&
				element.Type().PkgPath() == astPackagePath {
				u.walkValue(element, inFunction)
			}
		}
	case reflect.Struct:
		if value.Type().PkgPath() != astPackagePath {
			return
		}
		for i := 0; i < value.NumField(); i++ {
			u.walkValue(value.Field(i), inFunction)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			u.walkValue(value.Index(i), inFunction)
		}
	}
}

func (u *localAtomUsage) isTargetLocal(node *ast.Node) bool {
	if node == nil || node.Op != ast.OpLocal {
		return false
	}
	return node.Sub.(*ast.LocalNode).Name == u.target
}

func localAtomOperation(invoke *ast.InvokeNode) string {
	switch {
	case isCoreInvoke(invoke, "deref") && len(invoke.Args) == 1:
		return "deref"
	case isCoreInvoke(invoke, "reset!") && len(invoke.Args) == 2:
		return "reset!"
	case isCoreInvoke(invoke, "swap!") && len(invoke.Args) >= 2:
		return "swap!"
	default:
		return ""
	}
}

func isCoreInvoke(invoke *ast.InvokeNode, name string) bool {
	if invoke == nil || invoke.Fn == nil || invoke.Fn.Op != ast.OpVar {
		return false
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr == nil || vr.Namespace() == nil {
		return false
	}
	return vr.Namespace().Name().String() == "clojure.core" &&
		vr.Symbol().String() == name
}

func (g *Generator) generateLocalAtomInvoke(
	invoke *ast.InvokeNode,
) (string, bool) {
	operation := localAtomOperation(invoke)
	if operation == "" || len(invoke.Args) == 0 ||
		invoke.Args[0].Op != ast.OpLocal {
		return "", false
	}
	localName := invoke.Args[0].Sub.(*ast.LocalNode).Name.Name()
	atomVar, ok := g.getLocalAtom(localName)
	if !ok {
		return "", false
	}

	switch operation {
	case "deref":
		return atomVar, true
	case "reset!":
		value := g.generateASTNode(invoke.Args[1])
		g.writeAssign(atomVar, value)
		return atomVar, true
	case "swap!":
		args := []string{atomVar}
		var fn string
		if invoke.Args[1].Op == ast.OpFn {
			for _, arg := range invoke.Args[2:] {
				args = append(args, g.generateASTNode(arg))
			}
			if result, ok := g.generateImmediateFnInvoke(invoke.Args[1], args); ok {
				g.writeAssign(atomVar, result)
				return result, true
			}
			fn = g.generateASTNode(invoke.Args[1])
		} else {
			fn = g.generateASTNode(invoke.Args[1])
			for _, arg := range invoke.Args[2:] {
				args = append(args, g.generateASTNode(arg))
			}
		}
		result := g.allocateTempVar()
		g.generateApply(result, fn, args, true)
		g.writeAssign(atomVar, result)
		return result, true
	default:
		panic("unsupported local atom operation: " + operation)
	}
}

func (g *Generator) generateImmediateFnInvoke(
	fn *ast.Node,
	args []string,
) (string, bool) {
	if fn == nil || fn.Op != ast.OpFn {
		return "", false
	}
	fnNode := fn.Sub.(*ast.FnNode)
	if fnNode.Local != nil {
		return "", false
	}
	var method *ast.FnMethodNode
	for _, candidate := range fnNode.Methods {
		candidateMethod := candidate.Sub.(*ast.FnMethodNode)
		if !candidateMethod.IsVariadic &&
			candidateMethod.FixedArity == len(args) {
			method = candidateMethod
			break
		}
	}
	if method == nil || astContainsOp(method.Body, ast.OpRecur) {
		return "", false
	}

	result := g.allocateTempVar()
	g.writef("var %s any\n", result)
	g.writef("{ // immediate fn\n")
	g.pushVarScope()
	for i, param := range method.Params {
		paramNode := param.Sub.(*ast.BindingNode)
		name := paramNode.Name.Name()
		local := g.allocateLocal(name)
		g.writef("var %s any = %s\n", local, args[i])
		g.writeAssign("_", local)
	}
	body := g.generateASTNode(method.Body)
	if body != "" {
		g.writeAssign(result, body)
	}
	g.popVarScope()
	g.writef("} // end immediate fn\n")
	return result, true
}

func astContainsOp(node *ast.Node, operation ast.NodeOp) bool {
	found := false
	var walkNode func(*ast.Node)
	var walkValue func(reflect.Value)
	walkNode = func(current *ast.Node) {
		if current == nil || found {
			return
		}
		if current.Op == operation {
			found = true
			return
		}
		walkValue(reflect.ValueOf(current.Sub))
	}
	walkValue = func(value reflect.Value) {
		if !value.IsValid() || found {
			return
		}
		if value.Type() == astNodePointerType {
			if !value.IsNil() {
				walkNode(value.Interface().(*ast.Node))
			}
			return
		}
		switch value.Kind() {
		case reflect.Interface, reflect.Pointer:
			if !value.IsNil() {
				element := value.Elem()
				if element.Kind() == reflect.Struct &&
					element.Type().PkgPath() == astPackagePath {
					walkValue(element)
				}
			}
		case reflect.Struct:
			if value.Type().PkgPath() != astPackagePath {
				return
			}
			for i := 0; i < value.NumField(); i++ {
				walkValue(value.Field(i))
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				walkValue(value.Index(i))
			}
		}
	}
	walkNode(node)
	return found
}
