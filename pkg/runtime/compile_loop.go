package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// compiledInt64Loop is a conservative, typed fast path for numeric loop/recur
// kernels. It is only selected when every input and operation can remain an
// unboxed int64; all other loops continue through the general evaluator.
type compiledInt64Loop struct {
	arity  int
	test   int64BoolExpr
	next   []int64Expr
	result int64Expr
}

type int64Expr func(*[4]int64) int64
type int64BoolExpr func(*[4]int64) bool

func compileInt64Loop(letNode *ast.LetNode) *compiledInt64Loop {
	if len(letNode.Bindings) == 0 || len(letNode.Bindings) > 4 {
		return nil
	}
	slots := make(map[*lang.Symbol]int, len(letNode.Bindings))
	for i, binding := range letNode.Bindings {
		slots[binding.Sub.(*ast.BindingNode).Name] = i
	}

	body := unwrapSingleExpressionDo(letNode.Body)
	if body.Op != ast.OpIf {
		return nil
	}
	ifNode := body.Sub.(*ast.IfNode)
	test := compileInt64BoolExpr(ifNode.Test, slots)
	if test == nil || ifNode.Then.Op != ast.OpRecur {
		return nil
	}
	recur := ifNode.Then.Sub.(*ast.RecurNode)
	if len(recur.Exprs) != len(letNode.Bindings) {
		return nil
	}
	next := make([]int64Expr, len(recur.Exprs))
	for i, expr := range recur.Exprs {
		next[i] = compileInt64Expr(expr, slots)
		if next[i] == nil {
			return nil
		}
	}
	result := compileInt64Expr(ifNode.Else, slots)
	if result == nil {
		return nil
	}
	return &compiledInt64Loop{
		arity:  len(letNode.Bindings),
		test:   test,
		next:   next,
		result: result,
	}
}

func unwrapSingleExpressionDo(n *ast.Node) *ast.Node {
	if n.Op != ast.OpDo {
		return n
	}
	do := n.Sub.(*ast.DoNode)
	if len(do.Statements) != 0 {
		return n
	}
	return do.Ret
}

func (loop *compiledInt64Loop) run(bindNameVals []interface{}) (interface{}, bool) {
	var values [4]int64
	for i := 0; i < loop.arity; i++ {
		value, ok := bindNameVals[i*2+1].(int64)
		if !ok {
			return nil, false
		}
		values[i] = value
	}
	for loop.test(&values) {
		var next [4]int64
		for i, expr := range loop.next {
			next[i] = expr(&values)
		}
		values = next
	}
	return loop.result(&values), true
}

func compileInt64Expr(n *ast.Node, slots map[*lang.Symbol]int) int64Expr {
	switch n.Op {
	case ast.OpConst:
		value, ok := n.Sub.(*ast.ConstNode).Value.(int64)
		if !ok {
			return nil
		}
		return func(*[4]int64) int64 { return value }

	case ast.OpLocal:
		slot, ok := slots[n.Sub.(*ast.LocalNode).Name]
		if !ok {
			return nil
		}
		return func(values *[4]int64) int64 { return values[slot] }

	case ast.OpHostCall:
		call := n.Sub.(*ast.HostCallNode)
		if !isNumbersCall(call) {
			return nil
		}
		name := strings.ToLower(call.Method.Name())
		switch name {
		case "inc", "unchecked_inc":
			if len(call.Args) != 1 {
				return nil
			}
			arg := compileInt64Expr(call.Args[0], slots)
			if arg == nil {
				return nil
			}
			if name == "unchecked_inc" {
				return func(values *[4]int64) int64 { return arg(values) + 1 }
			}
			return func(values *[4]int64) int64 { return checkedInt64Add(arg(values), 1) }

		case "dec", "uncheckeddec", "unchecked_dec":
			if len(call.Args) != 1 {
				return nil
			}
			arg := compileInt64Expr(call.Args[0], slots)
			if arg == nil {
				return nil
			}
			if name != "dec" {
				return func(values *[4]int64) int64 { return arg(values) - 1 }
			}
			return func(values *[4]int64) int64 { return checkedInt64Sub(arg(values), 1) }

		case "add", "uncheckedadd":
			return compileInt64Binary(call.Args, slots, func(a, b int64) int64 {
				if name == "uncheckedadd" {
					return a + b
				}
				return checkedInt64Add(a, b)
			})

		case "minus", "unchecked_minus":
			return compileInt64Binary(call.Args, slots, func(a, b int64) int64 {
				if name == "unchecked_minus" {
					return a - b
				}
				return checkedInt64Sub(a, b)
			})

		case "multiply", "unchecked_multiply":
			return compileInt64Binary(call.Args, slots, func(a, b int64) int64 {
				if name == "unchecked_multiply" {
					return a * b
				}
				return checkedInt64Multiply(a, b)
			})
		}
	}
	return nil
}

func compileInt64Binary(
	args []*ast.Node,
	slots map[*lang.Symbol]int,
	op func(int64, int64) int64,
) int64Expr {
	if len(args) != 2 {
		return nil
	}
	left := compileInt64Expr(args[0], slots)
	right := compileInt64Expr(args[1], slots)
	if left == nil || right == nil {
		return nil
	}
	return func(values *[4]int64) int64 {
		return op(left(values), right(values))
	}
}

func compileInt64BoolExpr(n *ast.Node, slots map[*lang.Symbol]int) int64BoolExpr {
	if n.Op != ast.OpHostCall {
		return nil
	}
	call := n.Sub.(*ast.HostCallNode)
	if !isNumbersCall(call) || len(call.Args) != 2 {
		return nil
	}
	left := compileInt64Expr(call.Args[0], slots)
	right := compileInt64Expr(call.Args[1], slots)
	if left == nil || right == nil {
		return nil
	}
	switch strings.ToLower(call.Method.Name()) {
	case "lt":
		return func(values *[4]int64) bool { return left(values) < right(values) }
	case "lte":
		return func(values *[4]int64) bool { return left(values) <= right(values) }
	case "gt":
		return func(values *[4]int64) bool { return left(values) > right(values) }
	case "gte":
		return func(values *[4]int64) bool { return left(values) >= right(values) }
	}
	return nil
}

func isNumbersCall(call *ast.HostCallNode) bool {
	if call.ResolvedMethod == nil || call.Target.Op != ast.OpConst {
		return false
	}
	return call.Target.Sub.(*ast.ConstNode).Value == lang.Numbers
}

func checkedInt64Add(a, b int64) int64 {
	result := a + b
	if (result > a) == (b > 0) {
		return result
	}
	panic(lang.NewArithmeticError("integer overflow"))
}

func checkedInt64Sub(a, b int64) int64 {
	result := a - b
	if (result < a) == (b > 0) {
		return result
	}
	panic(lang.NewArithmeticError("integer overflow"))
}

func checkedInt64Multiply(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	result := a * b
	if (result < 0) == ((a < 0) != (b < 0)) && result/b == a {
		return result
	}
	panic(lang.NewArithmeticError("integer overflow"))
}
