//go:build !glj_aot_runtime

package runtime

import (
	"math"
	"math/big"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// int64AOTInterval is a conservative inclusive range. Unknown ranges never
// authorize removal of an overflow check.
type int64AOTInterval struct {
	min, max int64
	known    bool
}

func exactInt64AOTInterval(value int64) int64AOTInterval {
	return int64AOTInterval{min: value, max: value, known: true}
}

type int64AOTRangeAnalyzer struct {
	analysis         *int64AOTAnalysis
	speculativeInt32 bool
}

func (a *int64AOTAnalysis) proveSafeOperations(method *ast.FnMethodNode) {
	locals := make(map[*lang.Symbol]int64AOTInterval, method.FixedArity)
	for _, param := range method.Params {
		locals[param.Sub.(*ast.BindingNode).Name] = int64AOTInterval{}
	}
	(&int64AOTRangeAnalyzer{analysis: a}).expr(method.Body, locals)

	speculative := *a
	speculative.uncheckedHostCalls = cloneUncheckedInt64AOTCalls(
		a.uncheckedHostCalls,
	)
	speculative.guardInt32Loops = make(map[*ast.LetNode]bool)
	for _, param := range method.Params {
		locals[param.Sub.(*ast.BindingNode).Name] = int32AOTInterval()
	}
	(&int64AOTRangeAnalyzer{
		analysis:         &speculative,
		speculativeInt32: true,
	}).expr(method.Body, locals)
	if len(speculative.uncheckedHostCalls) > len(a.uncheckedHostCalls) {
		a.uncheckedHostCalls = speculative.uncheckedHostCalls
		a.guardInt32Params = method.FixedArity > 0
		a.guardInt32Loops = speculative.guardInt32Loops
	}
}

func int32AOTInterval() int64AOTInterval {
	return int64AOTInterval{
		min:   -math.MaxInt32,
		max:   math.MaxInt32,
		known: true,
	}
}

func cloneUncheckedInt64AOTCalls(
	calls map[*ast.HostCallNode]bool,
) map[*ast.HostCallNode]bool {
	clone := make(map[*ast.HostCallNode]bool, len(calls))
	for call, unchecked := range calls {
		clone[call] = unchecked
	}
	return clone
}

func (a *int64AOTRangeAnalyzer) expr(
	node *ast.Node,
	locals map[*lang.Symbol]int64AOTInterval,
) int64AOTInterval {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpConst:
		if value, ok := node.Sub.(*ast.ConstNode).Value.(int64); ok {
			return exactInt64AOTInterval(value)
		}
	case ast.OpLocal:
		return locals[node.Sub.(*ast.LocalNode).Name]
	case ast.OpLet:
		letNode := node.Sub.(*ast.LetNode)
		nested := cloneInt64AOTIntervals(locals)
		for _, binding := range letNode.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			nested[bindingNode.Name] = a.expr(bindingNode.Init, nested)
		}
		return a.expr(letNode.Body, nested)
	case ast.OpLoop:
		a.loop(node.Sub.(*ast.LetNode), locals)
	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		a.expr(ifNode.Test, locals)
		a.expr(ifNode.Then, locals)
		a.expr(ifNode.Else, locals)
	case ast.OpHostCall:
		return a.hostCall(node.Sub.(*ast.HostCallNode), locals)
	case ast.OpInvoke:
		for _, arg := range node.Sub.(*ast.InvokeNode).Args {
			a.expr(arg, locals)
		}
	}
	return int64AOTInterval{}
}

func (a *int64AOTRangeAnalyzer) hostCall(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]int64AOTInterval,
) int64AOTInterval {
	args := make([]int64AOTInterval, len(call.Args))
	for i, arg := range call.Args {
		args[i] = a.expr(arg, locals)
	}
	if !isNumbersCall(call) {
		return int64AOTInterval{}
	}

	name := strings.ToLower(call.Method.Name())
	var result int64AOTInterval
	var safe bool
	switch {
	case len(args) == 1 && name == "inc":
		result, safe = int64AOTBinaryRange("add", args[0], exactInt64AOTInterval(1))
	case len(args) == 1 && name == "dec":
		result, safe = int64AOTBinaryRange("sub", args[0], exactInt64AOTInterval(1))
	case len(args) == 1 && name == "minus":
		if args[0].known && args[0].min != math.MinInt64 {
			result = int64AOTInterval{
				min:   -args[0].max,
				max:   -args[0].min,
				known: true,
			}
			safe = true
		}
	case len(args) == 2 && name == "add":
		result, safe = int64AOTBinaryRange("add", args[0], args[1])
	case len(args) == 2 && name == "minus":
		result, safe = int64AOTBinaryRange("sub", args[0], args[1])
	case len(args) == 2 && name == "multiply":
		result, safe = int64AOTBinaryRange("mul", args[0], args[1])
	case len(args) == 2 && name == "quotient":
		result, safe = int64AOTQuotientRange(args[0], args[1])
	}
	if safe && (name == "inc" || name == "dec" || name == "minus" ||
		name == "add" || name == "multiply") {
		a.analysis.uncheckedHostCalls[call] = true
	}
	return result
}

func int64AOTBinaryRange(
	op string,
	left, right int64AOTInterval,
) (int64AOTInterval, bool) {
	if !left.known || !right.known {
		return int64AOTInterval{}, false
	}
	values := make([]*big.Int, 0, 4)
	for _, x := range []int64{left.min, left.max} {
		for _, y := range []int64{right.min, right.max} {
			bx, by := big.NewInt(x), big.NewInt(y)
			switch op {
			case "add":
				values = append(values, new(big.Int).Add(bx, by))
			case "sub":
				values = append(values, new(big.Int).Sub(bx, by))
			case "mul":
				values = append(values, new(big.Int).Mul(bx, by))
			}
		}
	}
	return int64AOTRangeFromBig(values...)
}

func int64AOTQuotientRange(
	numerator, denominator int64AOTInterval,
) (int64AOTInterval, bool) {
	if !numerator.known || !denominator.known ||
		denominator.min != denominator.max || denominator.min == 0 ||
		(numerator.min == math.MinInt64 && denominator.min == -1) {
		return int64AOTInterval{}, false
	}
	first := numerator.min / denominator.min
	last := numerator.max / denominator.min
	return int64AOTInterval{
		min:   min(first, last),
		max:   max(first, last),
		known: true,
	}, true
}

func int64AOTRangeFromBig(values ...*big.Int) (int64AOTInterval, bool) {
	low, high := new(big.Int).Set(values[0]), new(big.Int).Set(values[0])
	for _, value := range values[1:] {
		if value.Cmp(low) < 0 {
			low.Set(value)
		}
		if value.Cmp(high) > 0 {
			high.Set(value)
		}
	}
	if low.Cmp(big.NewInt(math.MinInt64)) < 0 ||
		high.Cmp(big.NewInt(math.MaxInt64)) > 0 {
		return int64AOTInterval{}, false
	}
	return int64AOTInterval{min: low.Int64(), max: high.Int64(), known: true}, true
}

type boundedInt64AOTLoop struct {
	control    int
	iterations *big.Int
	recurThen  bool
	recur      *ast.RecurNode
}

func (a *int64AOTRangeAnalyzer) loop(
	loop *ast.LetNode,
	outer map[*lang.Symbol]int64AOTInterval,
) {
	initial := cloneInt64AOTIntervals(outer)
	names := make([]*lang.Symbol, len(loop.Bindings))
	for i, binding := range loop.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		names[i] = bindingNode.Name
		initial[bindingNode.Name] = a.expr(bindingNode.Init, initial)
	}

	if a.speculativeInt32 {
		guarded := cloneInt64AOTIntervals(initial)
		for _, name := range names {
			guarded[name] = int32AOTInterval()
		}
		a.analysis.guardInt32Loops[loop] = true
		a.loopTail(loop.Body, guarded, loop.LoopID)
		return
	}

	bounded := detectBoundedInt64AOTLoop(loop, initial, names)
	if bounded == nil {
		unknown := cloneInt64AOTIntervals(initial)
		for _, name := range names {
			unknown[name] = int64AOTInterval{}
		}
		a.loopTail(loop.Body, unknown, loop.LoopID)
		return
	}

	all := cloneInt64AOTIntervals(initial)
	continuing := cloneInt64AOTIntervals(initial)
	exiting := cloneInt64AOTIntervals(initial)
	for _, name := range names {
		all[name] = int64AOTInterval{}
		continuing[name] = int64AOTInterval{}
		exiting[name] = int64AOTInterval{}
	}

	controlName := names[bounded.control]
	start := initial[controlName].min
	step, _ := int64AOTRecurrenceStep(
		bounded.recur.Exprs[bounded.control],
		controlName,
	)
	all[controlName] = int64AOTProgressionRange(
		start, step, bounded.iterations,
	)
	exiting[controlName] = int64AOTProgressionRange(
		start, step, bounded.iterations,
	)
	if bounded.iterations.Sign() > 0 {
		last := new(big.Int).Sub(bounded.iterations, big.NewInt(1))
		continuing[controlName] = int64AOTProgressionRange(start, step, last)
	}

	for i, name := range names {
		if i == bounded.control {
			continue
		}
		deltaNode, ok := int64AOTAdditiveRecurrence(bounded.recur.Exprs[i], name)
		if !ok || !initial[name].known {
			continue
		}
		delta := a.expr(deltaNode, continuing)
		if !delta.known {
			continue
		}
		all[name] = int64AOTRepeatedAddRange(
			initial[name], delta, bounded.iterations,
		)
		exiting[name] = int64AOTStateRange(
			initial[name], delta, bounded.iterations,
		)
		if bounded.iterations.Sign() > 0 {
			last := new(big.Int).Sub(bounded.iterations, big.NewInt(1))
			continuing[name] = int64AOTRepeatedAddRange(
				initial[name], delta, last,
			)
		}
	}

	ifNode := unwrapAOTDo(loop.Body).Sub.(*ast.IfNode)
	a.expr(ifNode.Test, all)
	if bounded.recurThen {
		a.loopTail(ifNode.Then, continuing, loop.LoopID)
		a.loopTail(ifNode.Else, exiting, loop.LoopID)
	} else {
		a.loopTail(ifNode.Then, exiting, loop.LoopID)
		a.loopTail(ifNode.Else, continuing, loop.LoopID)
	}
}

func (a *int64AOTRangeAnalyzer) loopTail(
	node *ast.Node,
	locals map[*lang.Symbol]int64AOTInterval,
	loopID *lang.Symbol,
) {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpRecur:
		recur := node.Sub.(*ast.RecurNode)
		if lang.Equals(recur.LoopID, loopID) {
			for _, expr := range recur.Exprs {
				a.expr(expr, locals)
			}
			return
		}
	case ast.OpIf:
		ifNode := node.Sub.(*ast.IfNode)
		a.expr(ifNode.Test, locals)
		a.loopTail(ifNode.Then, locals, loopID)
		a.loopTail(ifNode.Else, locals, loopID)
		return
	case ast.OpLet:
		letNode := node.Sub.(*ast.LetNode)
		nested := cloneInt64AOTIntervals(locals)
		for _, binding := range letNode.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			nested[bindingNode.Name] = a.expr(bindingNode.Init, nested)
		}
		a.loopTail(letNode.Body, nested, loopID)
		return
	}
	a.expr(node, locals)
}

func detectBoundedInt64AOTLoop(
	loop *ast.LetNode,
	initial map[*lang.Symbol]int64AOTInterval,
	names []*lang.Symbol,
) *boundedInt64AOTLoop {
	body := unwrapAOTDo(loop.Body)
	if body.Op != ast.OpIf {
		return nil
	}
	ifNode := body.Sub.(*ast.IfNode)
	thenRecur := findSingleInt64AOTRecur(ifNode.Then, loop.LoopID)
	elseRecur := findSingleInt64AOTRecur(ifNode.Else, loop.LoopID)
	if (thenRecur == nil) == (elseRecur == nil) {
		return nil
	}
	recur, recurThen := thenRecur, true
	if recur == nil {
		recur, recurThen = elseRecur, false
	}
	if len(recur.Exprs) != len(names) {
		return nil
	}

	op, args, ok := int64AOTComparison(unwrapAOTDo(ifNode.Test))
	if !ok {
		return nil
	}
	if !recurThen {
		op = invertInt64AOTComparison(op)
	}
	for i, name := range names {
		start := initial[name]
		if !start.known || start.min != start.max {
			continue
		}
		bound, normalizedOp, ok := int64AOTComparisonBound(
			args, name, initial, op,
		)
		if !ok {
			continue
		}
		step, ok := int64AOTRecurrenceStep(recur.Exprs[i], name)
		if !ok || step == 0 {
			continue
		}
		iterations, ok := int64AOTLoopIterations(
			start.min, bound, step, normalizedOp,
		)
		if !ok || !int64AOTProgressionRange(start.min, step, iterations).known {
			continue
		}
		return &boundedInt64AOTLoop{
			control: i, iterations: iterations,
			recurThen: recurThen, recur: recur,
		}
	}
	return nil
}

func int64AOTComparison(node *ast.Node) (string, []*ast.Node, bool) {
	switch node.Op {
	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		if isNumbersCall(call) && len(call.Args) == 2 {
			switch name := strings.ToLower(call.Method.Name()); name {
			case "lt", "lte", "gt", "gte", "equiv":
				return name, call.Args, true
			}
		}
	case ast.OpInvoke:
		invoke := node.Sub.(*ast.InvokeNode)
		if invoke.Fn.Op == ast.OpVar && len(invoke.Args) == 2 &&
			invoke.Fn.Sub.(*ast.VarNode).Var.String() == "#'clojure.core/=" {
			return "equiv", invoke.Args, true
		}
	}
	return "", nil, false
}

func int64AOTComparisonBound(
	args []*ast.Node,
	local *lang.Symbol,
	locals map[*lang.Symbol]int64AOTInterval,
	op string,
) (int64, string, bool) {
	left, right := unwrapAOTDo(args[0]), unwrapAOTDo(args[1])
	if int64AOTIsLocal(left, local) {
		bound := int64AOTSimpleRange(right, locals)
		if bound.known && bound.min == bound.max {
			return bound.min, op, true
		}
	}
	if int64AOTIsLocal(right, local) {
		bound := int64AOTSimpleRange(left, locals)
		if bound.known && bound.min == bound.max {
			return bound.min, reverseInt64AOTComparison(op), true
		}
	}
	return 0, "", false
}

func int64AOTSimpleRange(
	node *ast.Node,
	locals map[*lang.Symbol]int64AOTInterval,
) int64AOTInterval {
	node = unwrapAOTDo(node)
	if node.Op == ast.OpConst {
		if value, ok := node.Sub.(*ast.ConstNode).Value.(int64); ok {
			return exactInt64AOTInterval(value)
		}
	}
	if node.Op == ast.OpLocal {
		return locals[node.Sub.(*ast.LocalNode).Name]
	}
	return int64AOTInterval{}
}

func int64AOTRecurrenceStep(
	node *ast.Node,
	local *lang.Symbol,
) (int64, bool) {
	node = unwrapAOTDo(node)
	if node.Op != ast.OpHostCall {
		return 0, false
	}
	call := node.Sub.(*ast.HostCallNode)
	if !isNumbersCall(call) {
		return 0, false
	}
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 1 && int64AOTIsLocal(call.Args[0], local) {
		if name == "inc" || name == "unchecked_inc" {
			return 1, true
		}
		if name == "dec" || name == "uncheckeddec" || name == "unchecked_dec" {
			return -1, true
		}
	}
	if len(call.Args) != 2 {
		return 0, false
	}
	if name == "add" || name == "uncheckedadd" {
		if int64AOTIsLocal(call.Args[0], local) {
			return int64AOTConstant(call.Args[1])
		}
		if int64AOTIsLocal(call.Args[1], local) {
			return int64AOTConstant(call.Args[0])
		}
	}
	if (name == "minus" || name == "unchecked_minus") &&
		int64AOTIsLocal(call.Args[0], local) {
		value, ok := int64AOTConstant(call.Args[1])
		if !ok || value == math.MinInt64 {
			return 0, false
		}
		return -value, true
	}
	return 0, false
}

func int64AOTAdditiveRecurrence(
	node *ast.Node,
	local *lang.Symbol,
) (*ast.Node, bool) {
	node = unwrapAOTDo(node)
	if node.Op != ast.OpHostCall {
		return nil, false
	}
	call := node.Sub.(*ast.HostCallNode)
	if !isNumbersCall(call) {
		return nil, false
	}
	name := strings.ToLower(call.Method.Name())
	if len(call.Args) == 2 && (name == "add" || name == "uncheckedadd") {
		if int64AOTIsLocal(call.Args[0], local) {
			return call.Args[1], true
		}
		if int64AOTIsLocal(call.Args[1], local) {
			return call.Args[0], true
		}
	}
	return nil, false
}

func findSingleInt64AOTRecur(
	node *ast.Node,
	loopID *lang.Symbol,
) *ast.RecurNode {
	var found *ast.RecurNode
	var visit func(*ast.Node) bool
	visit = func(node *ast.Node) bool {
		node = unwrapAOTDo(node)
		switch node.Op {
		case ast.OpRecur:
			recur := node.Sub.(*ast.RecurNode)
			if lang.Equals(recur.LoopID, loopID) {
				if found != nil {
					return false
				}
				found = recur
			}
		case ast.OpIf:
			ifNode := node.Sub.(*ast.IfNode)
			return visit(ifNode.Then) && visit(ifNode.Else)
		case ast.OpLet:
			return visit(node.Sub.(*ast.LetNode).Body)
		}
		return true
	}
	if !visit(node) {
		return nil
	}
	return found
}

func int64AOTIsLocal(node *ast.Node, local *lang.Symbol) bool {
	node = unwrapAOTDo(node)
	return node.Op == ast.OpLocal && node.Sub.(*ast.LocalNode).Name == local
}

func int64AOTConstant(node *ast.Node) (int64, bool) {
	node = unwrapAOTDo(node)
	if node.Op != ast.OpConst {
		return 0, false
	}
	value, ok := node.Sub.(*ast.ConstNode).Value.(int64)
	return value, ok
}

func invertInt64AOTComparison(op string) string {
	switch op {
	case "lt":
		return "gte"
	case "lte":
		return "gt"
	case "gt":
		return "lte"
	case "gte":
		return "lt"
	case "equiv":
		return "ne"
	}
	return ""
}

func reverseInt64AOTComparison(op string) string {
	switch op {
	case "lt":
		return "gt"
	case "lte":
		return "gte"
	case "gt":
		return "lt"
	case "gte":
		return "lte"
	}
	return op
}

func int64AOTLoopIterations(
	start, bound, step int64,
	op string,
) (*big.Int, bool) {
	s, b, d := big.NewInt(start), big.NewInt(bound), big.NewInt(step)
	zero := new(big.Int)
	switch op {
	case "lt":
		if step <= 0 {
			return nil, false
		}
		if s.Cmp(b) >= 0 {
			return zero, true
		}
		return ceilPositiveBig(new(big.Int).Sub(b, s), d), true
	case "lte":
		if step <= 0 {
			return nil, false
		}
		if s.Cmp(b) > 0 {
			return zero, true
		}
		return new(big.Int).Add(
			new(big.Int).Quo(new(big.Int).Sub(b, s), d),
			big.NewInt(1),
		), true
	case "gt":
		if step >= 0 {
			return nil, false
		}
		if s.Cmp(b) <= 0 {
			return zero, true
		}
		return ceilPositiveBig(
			new(big.Int).Sub(s, b),
			new(big.Int).Neg(d),
		), true
	case "gte":
		if step >= 0 {
			return nil, false
		}
		if s.Cmp(b) < 0 {
			return zero, true
		}
		return new(big.Int).Add(
			new(big.Int).Quo(new(big.Int).Sub(s, b), new(big.Int).Neg(d)),
			big.NewInt(1),
		), true
	case "ne":
		difference := new(big.Int).Sub(b, s)
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(difference, d, remainder)
		if remainder.Sign() != 0 || quotient.Sign() < 0 {
			return nil, false
		}
		return quotient, true
	}
	return nil, false
}

func ceilPositiveBig(numerator, denominator *big.Int) *big.Int {
	return new(big.Int).Quo(
		new(big.Int).Add(numerator, new(big.Int).Sub(denominator, big.NewInt(1))),
		denominator,
	)
}

func int64AOTProgressionRange(
	start, step int64,
	iterations *big.Int,
) int64AOTInterval {
	end := new(big.Int).Mul(big.NewInt(step), iterations)
	end.Add(end, big.NewInt(start))
	result, ok := int64AOTRangeFromBig(big.NewInt(start), end)
	if !ok {
		return int64AOTInterval{}
	}
	return result
}

func int64AOTRepeatedAddRange(
	initial, delta int64AOTInterval,
	iterations *big.Int,
) int64AOTInterval {
	values := []*big.Int{big.NewInt(initial.min), big.NewInt(initial.max)}
	for _, change := range []int64{delta.min, delta.max} {
		for _, base := range []int64{initial.min, initial.max} {
			value := new(big.Int).Mul(big.NewInt(change), iterations)
			values = append(values, value.Add(value, big.NewInt(base)))
		}
	}
	result, ok := int64AOTRangeFromBig(values...)
	if !ok {
		return int64AOTInterval{}
	}
	return result
}

func int64AOTStateRange(
	initial, delta int64AOTInterval,
	iterations *big.Int,
) int64AOTInterval {
	values := make([]*big.Int, 0, 4)
	for _, change := range []int64{delta.min, delta.max} {
		for _, base := range []int64{initial.min, initial.max} {
			value := new(big.Int).Mul(big.NewInt(change), iterations)
			values = append(values, value.Add(value, big.NewInt(base)))
		}
	}
	result, ok := int64AOTRangeFromBig(values...)
	if !ok {
		return int64AOTInterval{}
	}
	return result
}

func cloneInt64AOTIntervals(
	locals map[*lang.Symbol]int64AOTInterval,
) map[*lang.Symbol]int64AOTInterval {
	copy := make(map[*lang.Symbol]int64AOTInterval, len(locals))
	for symbol, interval := range locals {
		copy[symbol] = interval
	}
	return copy
}
