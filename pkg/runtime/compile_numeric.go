package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
)

// Numeric regions keep homogeneous primitive arithmetic unboxed between the
// region's inputs and its result. They are emitted only for analyzed loop
// bodies, where arithmetic is likely to amortize the guards. The existing
// evaluator remains the semantic fallback when an input does not match.
type numericKind uint8

const (
	unknownNumericKind numericKind = iota
	int64NumericKind
	float64NumericKind
)

type primitiveRegionExpr[T any] struct {
	eval       func(*environment) (T, bool)
	operations int
	loopBased  bool
	evidence   bool
}

type int64RegionExpr struct {
	eval       func(*environment) (int64, bool)
	operations int
	loopBased  bool
}

type (
	primitiveUnaryOp[T any]     func(T) T
	primitiveBinaryOp[T any]    func(T, T) T
	primitivePredicateOp[T any] func(T, T) bool
)

type primitiveRegionSpec[T any] struct {
	value           func(interface{}) (T, bool)
	evidence        func(interface{}) bool
	slotEvidence    func(localSlot) bool
	unary           func(string) primitiveUnaryOp[T]
	binary          func(string) primitiveBinaryOp[T]
	unaryPredicate  func(string) func(T) bool
	predicate       func(string) primitivePredicateOp[T]
	minOperations   int
	requireEvidence bool
}

type regionResult func(*environment) (interface{}, bool)

var (
	float64RegionSpec = primitiveRegionSpec[float64]{
		value: func(value interface{}) (float64, bool) {
			floating, ok := value.(float64)
			return floating, ok
		},
		evidence: func(value interface{}) bool {
			_, ok := value.(float64)
			return ok
		},
		slotEvidence: func(slot localSlot) bool {
			return slot.numericKind == float64NumericKind
		},
		unary:           float64RegionUnary,
		binary:          float64RegionBinary,
		unaryPredicate:  unaryRegionPredicate[float64],
		predicate:       orderedRegionPredicate[float64],
		minOperations:   1,
		requireEvidence: true,
	}
)

func (c threadedEvalCompiler) compileNumericRegion(
	call *ast.HostCallNode,
	fallback evalFn,
) evalFn {
	integer := c.compileInt64RegionResult(call)
	var floating regionResult
	if c.hasStaticFloatEvidence(call) {
		floating = compilePrimitiveRegionResult(c, call, float64RegionSpec)
	}
	if integer == nil && floating == nil {
		return fallback
	}

	return func(env *environment) (interface{}, error) {
		if integer != nil {
			if result, ok := integer(env); ok {
				return result, nil
			}
		}
		if floating != nil {
			if result, ok := floating(env); ok {
				return result, nil
			}
		}
		return fallback(env)
	}
}

func (c threadedEvalCompiler) compileInt64RegionResult(call *ast.HostCallNode) regionResult {
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 1 {
		arg := c.compileInt64RegionExpr(call.Args[0])
		if arg.eval == nil || !arg.loopBased || arg.operations == 0 {
			return nil
		}
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec":
			return func(env *environment) (interface{}, bool) {
				value, ok := arg.eval(env)
				if !ok {
					return nil, false
				}
				switch name {
				case "inc":
					return checkedInt64Add(value, 1), true
				case "dec":
					return checkedInt64Sub(value, 1), true
				case "unchecked_inc":
					return value + 1, true
				default:
					return value - 1, true
				}
			}
		case "iszero", "ispos", "isneg":
			return func(env *environment) (interface{}, bool) {
				value, ok := arg.eval(env)
				if !ok {
					return nil, false
				}
				switch name {
				case "iszero":
					return value == 0, true
				case "ispos":
					return value > 0, true
				default:
					return value < 0, true
				}
			}
		}
		return nil
	}
	if len(call.Args) != 2 {
		return nil
	}

	left := c.compileInt64RegionExpr(call.Args[0])
	right := c.compileInt64RegionExpr(call.Args[1])
	if left.eval == nil || right.eval == nil ||
		(!left.loopBased && !right.loopBased) ||
		left.operations+right.operations == 0 {
		return nil
	}
	switch name {
	case "add", "uncheckedadd", "minus", "unchecked_minus",
		"multiply", "unchecked_multiply", "quotient", "remainder":
		return func(env *environment) (interface{}, bool) {
			a, ok := left.eval(env)
			if !ok {
				return nil, false
			}
			b, ok := right.eval(env)
			if !ok {
				return nil, false
			}
			switch name {
			case "add":
				return checkedInt64Add(a, b), true
			case "uncheckedadd":
				return a + b, true
			case "minus":
				return checkedInt64Sub(a, b), true
			case "unchecked_minus":
				return a - b, true
			case "multiply":
				return checkedInt64Multiply(a, b), true
			case "unchecked_multiply":
				return a * b, true
			case "quotient":
				return checkedInt64Quotient(a, b), true
			default:
				return checkedInt64Remainder(a, b), true
			}
		}
	case "lt", "lte", "gt", "gte", "equiv":
		return func(env *environment) (interface{}, bool) {
			a, ok := left.eval(env)
			if !ok {
				return nil, false
			}
			b, ok := right.eval(env)
			if !ok {
				return nil, false
			}
			switch name {
			case "lt":
				return a < b, true
			case "lte":
				return a <= b, true
			case "gt":
				return a > b, true
			case "gte":
				return a >= b, true
			default:
				return a == b, true
			}
		}
	}
	return nil
}

func (c threadedEvalCompiler) compileInt64RegionExpr(n *ast.Node) int64RegionExpr {
	switch n.Op {
	case ast.OpConst:
		value, ok := n.Sub.(*ast.ConstNode).Value.(int64)
		if !ok {
			return int64RegionExpr{}
		}
		return int64RegionExpr{
			eval: func(*environment) (int64, bool) { return value, true },
		}
	case ast.OpLocal:
		slot, ok := c.localSlots[n.Sub.(*ast.LocalNode).Name]
		if !ok {
			return int64RegionExpr{}
		}
		return int64RegionExpr{
			loopBased: slot.kind == loopLocalSlot,
			eval: func(env *environment) (int64, bool) {
				value, ok := localSlotValue(env, slot).(int64)
				return value, ok
			},
		}
	case ast.OpVar:
		vr := n.Sub.(*ast.VarNode).Var
		if vr.IsMacro() {
			return int64RegionExpr{}
		}
		return int64RegionExpr{
			eval: func(*environment) (int64, bool) {
				value, ok := vr.Get().(int64)
				return value, ok
			},
		}
	case ast.OpHostCall:
		call := n.Sub.(*ast.HostCallNode)
		if !isNumbersCall(call) {
			return int64RegionExpr{}
		}
		return c.compileInt64ArithmeticRegion(call)
	default:
		return int64RegionExpr{}
	}
}

func (c threadedEvalCompiler) compileInt64ArithmeticRegion(
	call *ast.HostCallNode,
) int64RegionExpr {
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 1 {
		arg := c.compileInt64RegionExpr(call.Args[0])
		if arg.eval == nil {
			return int64RegionExpr{}
		}
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec":
			return int64RegionExpr{
				operations: arg.operations + 1,
				loopBased:  arg.loopBased,
				eval: func(env *environment) (int64, bool) {
					value, ok := arg.eval(env)
					if !ok {
						return 0, false
					}
					switch name {
					case "inc":
						return checkedInt64Add(value, 1), true
					case "dec":
						return checkedInt64Sub(value, 1), true
					case "unchecked_inc":
						return value + 1, true
					default:
						return value - 1, true
					}
				},
			}
		}
		return int64RegionExpr{}
	}
	if len(call.Args) != 2 {
		return int64RegionExpr{}
	}
	switch name {
	case "add", "uncheckedadd", "minus", "unchecked_minus",
		"multiply", "unchecked_multiply", "quotient", "remainder":
	default:
		return int64RegionExpr{}
	}
	left := c.compileInt64RegionExpr(call.Args[0])
	right := c.compileInt64RegionExpr(call.Args[1])
	if left.eval == nil || right.eval == nil {
		return int64RegionExpr{}
	}
	return int64RegionExpr{
		operations: left.operations + right.operations + 1,
		loopBased:  left.loopBased || right.loopBased,
		eval: func(env *environment) (int64, bool) {
			a, ok := left.eval(env)
			if !ok {
				return 0, false
			}
			b, ok := right.eval(env)
			if !ok {
				return 0, false
			}
			switch name {
			case "add":
				return checkedInt64Add(a, b), true
			case "uncheckedadd":
				return a + b, true
			case "minus":
				return checkedInt64Sub(a, b), true
			case "unchecked_minus":
				return a - b, true
			case "multiply":
				return checkedInt64Multiply(a, b), true
			case "unchecked_multiply":
				return a * b, true
			case "quotient":
				return checkedInt64Quotient(a, b), true
			default:
				return checkedInt64Remainder(a, b), true
			}
		},
	}
}

func compilePrimitiveRegionResult[T any](
	c threadedEvalCompiler,
	call *ast.HostCallNode,
	spec primitiveRegionSpec[T],
) regionResult {
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 1 {
		arg := compilePrimitiveRegionExpr(c, call.Args[0], spec)
		if !usablePrimitiveRegion(arg, spec, arg.operations+1) {
			return nil
		}
		if operation := spec.unary(name); operation != nil {
			return func(env *environment) (interface{}, bool) {
				value, ok := arg.eval(env)
				if !ok {
					return nil, false
				}
				return operation(value), true
			}
		}
		if predicate := spec.unaryPredicate(name); predicate != nil {
			return func(env *environment) (interface{}, bool) {
				value, ok := arg.eval(env)
				if !ok {
					return nil, false
				}
				return predicate(value), true
			}
		}
		return nil
	}
	if len(call.Args) != 2 {
		return nil
	}

	left := compilePrimitiveRegionExpr(c, call.Args[0], spec)
	right := compilePrimitiveRegionExpr(c, call.Args[1], spec)
	combined := primitiveRegionExpr[T]{
		eval:       left.eval,
		operations: left.operations + right.operations,
		loopBased:  left.loopBased || right.loopBased,
		evidence:   left.evidence || right.evidence,
	}
	if left.eval == nil || right.eval == nil ||
		!usablePrimitiveRegion(combined, spec, combined.operations+1) {
		return nil
	}
	if operation := spec.binary(name); operation != nil {
		return func(env *environment) (interface{}, bool) {
			a, ok := left.eval(env)
			if !ok {
				return nil, false
			}
			b, ok := right.eval(env)
			if !ok {
				return nil, false
			}
			return operation(a, b), true
		}
	}
	if predicate := spec.predicate(name); predicate != nil {
		return func(env *environment) (interface{}, bool) {
			a, ok := left.eval(env)
			if !ok {
				return nil, false
			}
			b, ok := right.eval(env)
			if !ok {
				return nil, false
			}
			return predicate(a, b), true
		}
	}
	return nil
}

func usablePrimitiveRegion[T any](
	expr primitiveRegionExpr[T],
	spec primitiveRegionSpec[T],
	operations int,
) bool {
	return expr.eval != nil &&
		expr.loopBased &&
		operations >= spec.minOperations &&
		(!spec.requireEvidence || expr.evidence)
}

func compilePrimitiveRegionExpr[T any](
	c threadedEvalCompiler,
	n *ast.Node,
	spec primitiveRegionSpec[T],
) primitiveRegionExpr[T] {
	switch n.Op {
	case ast.OpConst:
		value := n.Sub.(*ast.ConstNode).Value
		primitive, ok := spec.value(value)
		if !ok {
			return primitiveRegionExpr[T]{}
		}
		return primitiveRegionExpr[T]{
			evidence: spec.evidence(value),
			eval: func(*environment) (T, bool) {
				return primitive, true
			},
		}
	case ast.OpLocal:
		slot, ok := c.localSlots[n.Sub.(*ast.LocalNode).Name]
		if !ok {
			return primitiveRegionExpr[T]{}
		}
		return primitiveRegionExpr[T]{
			loopBased: slot.kind == loopLocalSlot,
			evidence:  spec.slotEvidence(slot),
			eval: func(env *environment) (T, bool) {
				return spec.value(localSlotValue(env, slot))
			},
		}
	case ast.OpVar:
		vr := n.Sub.(*ast.VarNode).Var
		if vr.IsMacro() {
			return primitiveRegionExpr[T]{}
		}
		return primitiveRegionExpr[T]{
			eval: func(*environment) (T, bool) {
				return spec.value(vr.Get())
			},
		}
	case ast.OpHostCall:
		call := n.Sub.(*ast.HostCallNode)
		if !isNumbersCall(call) {
			return primitiveRegionExpr[T]{}
		}
		return compilePrimitiveArithmeticRegion(c, call, spec)
	default:
		return primitiveRegionExpr[T]{}
	}
}

func compilePrimitiveArithmeticRegion[T any](
	c threadedEvalCompiler,
	call *ast.HostCallNode,
	spec primitiveRegionSpec[T],
) primitiveRegionExpr[T] {
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 1 {
		arg := compilePrimitiveRegionExpr(c, call.Args[0], spec)
		operation := spec.unary(name)
		if arg.eval == nil || operation == nil {
			return primitiveRegionExpr[T]{}
		}
		return primitiveRegionExpr[T]{
			operations: arg.operations + 1,
			loopBased:  arg.loopBased,
			evidence:   arg.evidence,
			eval: func(env *environment) (T, bool) {
				value, ok := arg.eval(env)
				if !ok {
					var zero T
					return zero, false
				}
				return operation(value), true
			},
		}
	}
	if len(call.Args) != 2 {
		return primitiveRegionExpr[T]{}
	}

	operation := spec.binary(name)
	if operation == nil {
		return primitiveRegionExpr[T]{}
	}
	left := compilePrimitiveRegionExpr(c, call.Args[0], spec)
	right := compilePrimitiveRegionExpr(c, call.Args[1], spec)
	if left.eval == nil || right.eval == nil {
		return primitiveRegionExpr[T]{}
	}
	return primitiveRegionExpr[T]{
		operations: left.operations + right.operations + 1,
		loopBased:  left.loopBased || right.loopBased,
		evidence:   left.evidence || right.evidence,
		eval: func(env *environment) (T, bool) {
			a, ok := left.eval(env)
			if !ok {
				var zero T
				return zero, false
			}
			b, ok := right.eval(env)
			if !ok {
				var zero T
				return zero, false
			}
			return operation(a, b), true
		},
	}
}

func float64RegionUnary(name string) primitiveUnaryOp[float64] {
	switch name {
	case "inc", "unchecked_inc":
		return func(value float64) float64 { return value + 1 }
	case "dec", "uncheckeddec", "unchecked_dec":
		return func(value float64) float64 { return value - 1 }
	default:
		return nil
	}
}

func float64RegionBinary(name string) primitiveBinaryOp[float64] {
	switch name {
	case "add", "uncheckedadd":
		return func(a, b float64) float64 { return a + b }
	case "minus", "unchecked_minus":
		return func(a, b float64) float64 { return a - b }
	case "multiply", "unchecked_multiply":
		return func(a, b float64) float64 { return a * b }
	default:
		return nil
	}
}

func unaryRegionPredicate[T int64 | float64](name string) func(T) bool {
	switch name {
	case "iszero":
		return func(value T) bool { return value == 0 }
	case "ispos":
		return func(value T) bool { return value > 0 }
	case "isneg":
		return func(value T) bool { return value < 0 }
	default:
		return nil
	}
}

func orderedRegionPredicate[T int64 | float64](name string) primitivePredicateOp[T] {
	switch name {
	case "lt":
		return func(a, b T) bool { return a < b }
	case "lte":
		return func(a, b T) bool { return a <= b }
	case "gt":
		return func(a, b T) bool { return a > b }
	case "gte":
		return func(a, b T) bool { return a >= b }
	case "equiv":
		return func(a, b T) bool { return a == b }
	default:
		return nil
	}
}

func (c threadedEvalCompiler) hasStaticFloatEvidence(call *ast.HostCallNode) bool {
	for _, arg := range call.Args {
		switch arg.Op {
		case ast.OpConst:
			if _, ok := arg.Sub.(*ast.ConstNode).Value.(float64); ok {
				return true
			}
		case ast.OpLocal:
			if c.localSlots[arg.Sub.(*ast.LocalNode).Name].numericKind == float64NumericKind {
				return true
			}
		case ast.OpHostCall:
			nested := arg.Sub.(*ast.HostCallNode)
			if isNumbersCall(nested) && c.hasStaticFloatEvidence(nested) {
				return true
			}
		}
	}
	return false
}

func (c threadedEvalCompiler) inferNumericKind(n *ast.Node) numericKind {
	switch n.Op {
	case ast.OpConst:
		switch n.Sub.(*ast.ConstNode).Value.(type) {
		case int64:
			return int64NumericKind
		case float64:
			return float64NumericKind
		}
	case ast.OpLocal:
		return c.localSlots[n.Sub.(*ast.LocalNode).Name].numericKind
	case ast.OpHostCall:
		call := n.Sub.(*ast.HostCallNode)
		if !isNumbersCall(call) {
			return unknownNumericKind
		}
		name := strings.ToLower(call.Method.Name())
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec":
			if len(call.Args) == 1 {
				return c.inferNumericKind(call.Args[0])
			}
		case "add", "uncheckedadd", "minus", "unchecked_minus",
			"multiply", "unchecked_multiply":
			if len(call.Args) != 2 {
				return unknownNumericKind
			}
			left := c.inferNumericKind(call.Args[0])
			right := c.inferNumericKind(call.Args[1])
			if left == float64NumericKind || right == float64NumericKind {
				return float64NumericKind
			}
			if left == int64NumericKind && right == int64NumericKind {
				return int64NumericKind
			}
		case "quotient", "remainder":
			if len(call.Args) == 2 &&
				c.inferNumericKind(call.Args[0]) == int64NumericKind &&
				c.inferNumericKind(call.Args[1]) == int64NumericKind {
				return int64NumericKind
			}
		}
	}
	return unknownNumericKind
}

func localSlotValue(env *environment, slot localSlot) interface{} {
	switch slot.kind {
	case loopLocalSlot:
		return env.loopFrame.args[slot.index]
	case letLocalSlot:
		return env.evalFrame.args[slot.index]
	default:
		return env.fnFrame.args[slot.index]
	}
}
