package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type fnGuardRoot struct {
	variable *lang.Var
	version  *lang.VarRootVersion
}

type unaryFalseGuard struct {
	test      func(int64) bool
	nonzero   bool
	otherwise interface{}
	roots     []fnGuardRoot
}

type unaryGuardExpr struct {
	eval    func(int64) int64
	nonzero bool
}

type unaryGuardCompiler struct {
	param *lang.Symbol
	env   *environment
	roots []fnGuardRoot
}

func (g *unaryFalseGuard) reject(arg interface{}) bool {
	value, ok := arg.(int64)
	if !ok || g.nonzero && value == 0 {
		return false
	}
	for _, root := range g.roots {
		if root.variable.RootVersion() != root.version || root.variable.IsMacro() || root.variable.IsDynamic() {
			return false
		}
	}
	return !g.test(value)
}

// Only pure integer comparisons and division enter this early return path.
// Failed type guards, redefined core functions, and zero divisors use the
// complete evaluator, including its normal error handling and taken branch.
func compileUnaryFalseGuard(body *ast.Node, param *lang.Symbol, env *environment) *unaryFalseGuard {
	body = unwrapSingleExpressionDo(body)
	if body.Op != ast.OpIf {
		return nil
	}
	branch := body.Sub.(*ast.IfNode)
	if branch.Else.Op != ast.OpConst {
		return nil
	}
	compiler := unaryGuardCompiler{param: param, env: env}
	method, args := compiler.call(branch.Test)
	if len(args) != 2 {
		return nil
	}
	left := compiler.expression(args[0])
	right := compiler.expression(args[1])
	if left.eval == nil || right.eval == nil {
		return nil
	}
	guard := &unaryFalseGuard{
		nonzero:   left.nonzero || right.nonzero,
		otherwise: branch.Else.Sub.(*ast.ConstNode).Value,
		roots:     compiler.roots,
	}
	switch method {
	case "equiv":
		guard.test = func(n int64) bool { return left.eval(n) == right.eval(n) }
	case "lt":
		guard.test = func(n int64) bool { return left.eval(n) < right.eval(n) }
	case "lte":
		guard.test = func(n int64) bool { return left.eval(n) <= right.eval(n) }
	case "gt":
		guard.test = func(n int64) bool { return left.eval(n) > right.eval(n) }
	case "gte":
		guard.test = func(n int64) bool { return left.eval(n) >= right.eval(n) }
	default:
		return nil
	}
	return guard
}

func (c *unaryGuardCompiler) call(node *ast.Node) (string, []*ast.Node) {
	if node.Op == ast.OpHostCall {
		call := node.Sub.(*ast.HostCallNode)
		if isNumbersCall(call) {
			return strings.ToLower(call.Method.Name()), call.Args
		}
	}
	if node.Op != ast.OpInvoke {
		return "", nil
	}
	call := node.Sub.(*ast.InvokeNode)
	if call.Fn.Op != ast.OpVar {
		return "", nil
	}
	variable := call.Fn.Sub.(*ast.VarNode).Var
	if variable.Namespace() != lang.NSCore {
		return "", nil
	}
	var method string
	switch variable.Symbol().Name() {
	case "=":
		method = "equiv"
	case "rem":
		method = "remainder"
	case "quot":
		method = "quotient"
	}
	if method == "" {
		return "", nil
	}
	version := variable.RootVersion()
	if !IsDefaultCoreVar(variable) || variable.RootVersion() != version {
		return "", nil
	}
	for _, root := range c.roots {
		if root.variable == variable {
			return method, call.Args
		}
	}
	c.roots = append(c.roots, fnGuardRoot{variable: variable, version: version})
	return method, call.Args
}

func (c *unaryGuardCompiler) expression(node *ast.Node) unaryGuardExpr {
	switch node.Op {
	case ast.OpConst:
		value, ok := node.Sub.(*ast.ConstNode).Value.(int64)
		if ok {
			return unaryGuardExpr{eval: func(int64) int64 { return value }}
		}
	case ast.OpLocal:
		name := node.Sub.(*ast.LocalNode).Name
		if name == c.param {
			return unaryGuardExpr{eval: func(n int64) int64 { return n }}
		}
		if c.env.scope == nil {
			return unaryGuardExpr{}
		}
		if value, ok := c.env.scope.lookup(name); ok {
			if integer, ok := value.(int64); ok {
				return unaryGuardExpr{eval: func(int64) int64 { return integer }}
			}
		}
	case ast.OpHostCall, ast.OpInvoke:
		method, args := c.call(node)
		if len(args) != 2 || method != "remainder" && method != "quotient" {
			return unaryGuardExpr{}
		}
		// Check a constant or argument denominator before doing arithmetic.
		divisor := args[1]
		if divisor.Op != ast.OpConst && divisor.Op != ast.OpLocal {
			return unaryGuardExpr{}
		}
		left := c.expression(args[0])
		right := c.expression(divisor)
		if left.eval == nil || right.eval == nil {
			return unaryGuardExpr{}
		}
		nonzero := left.nonzero || right.nonzero
		if divisor.Op == ast.OpLocal && divisor.Sub.(*ast.LocalNode).Name == c.param {
			nonzero = true
		} else if right.eval(0) == 0 {
			return unaryGuardExpr{}
		}
		if method == "remainder" {
			return unaryGuardExpr{eval: func(n int64) int64 { return left.eval(n) % right.eval(n) }, nonzero: nonzero}
		}
		return unaryGuardExpr{eval: func(n int64) int64 { return left.eval(n) / right.eval(n) }, nonzero: nonzero}
	}
	return unaryGuardExpr{}
}

type nilCountGuard struct {
	parameter int
	root      fnGuardRoot
	otherwise interface{}
}

func (g *nilCountGuard) reject(a0, a1, a2 interface{}) bool {
	value := a0
	switch g.parameter {
	case 1:
		value = a1
	case 2:
		value = a2
	}
	return value == nil && g.root.variable.RootVersion() == g.root.version &&
		!g.root.variable.IsMacro() && !g.root.variable.IsDynamic()
}

// A nil sequence has count zero under the standard count root. This avoids
// creating a frame just to return the constant empty-input branch.
func compileNilCountGuard(body *ast.Node, params []*ast.Node) *nilCountGuard {
	body = unwrapSingleExpressionDo(body)
	if body.Op != ast.OpIf {
		return nil
	}
	branch := body.Sub.(*ast.IfNode)
	if branch.Else.Op != ast.OpConst || branch.Test.Op != ast.OpHostCall {
		return nil
	}
	test := branch.Test.Sub.(*ast.HostCallNode)
	if !isNumbersCall(test) || !strings.EqualFold(test.Method.Name(), "gt") || len(test.Args) != 2 {
		return nil
	}
	zero, ok := int64ConstValue(test.Args[1])
	if !ok || zero != 0 || test.Args[0].Op != ast.OpInvoke {
		return nil
	}
	count := test.Args[0].Sub.(*ast.InvokeNode)
	if count.Fn.Op != ast.OpVar || len(count.Args) != 1 || count.Args[0].Op != ast.OpLocal {
		return nil
	}
	variable := count.Fn.Sub.(*ast.VarNode).Var
	if variable.Namespace() != lang.NSCore || variable.Symbol().Name() != "count" {
		return nil
	}
	version := variable.RootVersion()
	if !IsDefaultCoreVar(variable) || variable.RootVersion() != version {
		return nil
	}
	name := count.Args[0].Sub.(*ast.LocalNode).Name
	for i, param := range params {
		if param.Sub.(*ast.BindingNode).Name == name {
			return &nilCountGuard{
				parameter: i, root: fnGuardRoot{variable: variable, version: version},
				otherwise: branch.Else.Sub.(*ast.ConstNode).Value,
			}
		}
	}
	return nil
}
