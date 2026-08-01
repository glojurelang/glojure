//go:build !glj_aot_runtime

package runtime

import (
	"math/bits"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

// vectorAOTAnalysis describes a region where persistent vector updates can be
// represented by transients. The public function keeps its ordinary dynamic
// implementation as a fallback. Its guarded fast path copies vector arguments
// once, while direct-linked helper calls pass the owned transient through the
// whole update region.
type vectorAOTAnalysis struct {
	target     *aotSpecializationTarget
	paramMask  uint32
	paramTypes []compiler.IRType
	result     vectorAOTValue
	values     map[*ast.Node]vectorAOTValue
	assocs     map[*ast.Node]bool
	nths       map[*ast.Node]bool
	numbers    map[*ast.Node]bool
	calls      map[*ast.Node]*aotSpecializationTarget
	freeze     map[*ast.Node]bool
	mutated    uint32
}

type vectorAOTValue struct {
	transient bool
	origins   uint32
	control   bool
	typ       compiler.IRType
}

type vectorAOTLoop struct {
	id       *lang.Symbol
	expected []vectorAOTValue
	parent   *vectorAOTLoop
}

type vectorAOTAnalyzer struct {
	analysis *vectorAOTAnalysis
	targets  map[*lang.Var]*aotSpecializationTarget
	valid    bool
	loop     *vectorAOTLoop
	written  uint32
}

func analyzeVectorAOTFunction(
	target *aotSpecializationTarget,
	method *ast.FnMethodNode,
	targets map[*lang.Var]*aotSpecializationTarget,
) *vectorAOTAnalysis {
	if target == nil || method == nil || method.IsVariadic ||
		method.FixedArity < 1 || method.FixedArity > 31 {
		return nil
	}
	var best *vectorAOTAnalysis
	candidates := []compiler.IRType{
		{Kind: compiler.IRDynamic, Nullable: true},
		{Kind: compiler.IRBool},
		{Kind: compiler.IRInt},
		{Kind: compiler.IRVector},
	}
	combinations := 1
	for range method.FixedArity {
		combinations *= len(candidates)
		if combinations > 1<<16 {
			return nil
		}
	}
	for encoded := 0; encoded < combinations; encoded++ {
		value := encoded
		paramTypes := make([]compiler.IRType, method.FixedArity)
		var mask uint32
		for index := range paramTypes {
			paramTypes[index] = candidates[value%len(candidates)]
			value /= len(candidates)
			if paramTypes[index].Kind == compiler.IRVector {
				mask |= uint32(1) << index
			}
		}
		if mask == 0 {
			continue
		}
		analysis := &vectorAOTAnalysis{
			target:     target,
			paramMask:  mask,
			paramTypes: paramTypes,
			values:     make(map[*ast.Node]vectorAOTValue),
			assocs:     make(map[*ast.Node]bool),
			nths:       make(map[*ast.Node]bool),
			numbers:    make(map[*ast.Node]bool),
			calls:      make(map[*ast.Node]*aotSpecializationTarget),
			freeze:     make(map[*ast.Node]bool),
		}
		analyzer := &vectorAOTAnalyzer{
			analysis: analysis,
			targets:  targets,
			valid:    true,
		}
		locals := make(map[*lang.Symbol]vectorAOTValue, method.FixedArity)
		for index, param := range method.Params {
			if mask&(uint32(1)<<index) != 0 {
				locals[param.Sub.(*ast.BindingNode).Name] = vectorAOTValue{
					transient: true,
					origins:   uint32(1) << index,
				}
			} else {
				locals[param.Sub.(*ast.BindingNode).Name] =
					vectorAOTValue{typ: paramTypes[index]}
			}
		}
		analysis.result = analyzer.expr(method.Body, locals, true)
		if !analyzer.valid || analysis.result.control ||
			analysis.mutated&mask != mask {
			continue
		}
		if best == nil ||
			bits.OnesCount32(mask) > bits.OnesCount32(best.paramMask) ||
			bits.OnesCount32(mask) == bits.OnesCount32(best.paramMask) &&
				(len(analysis.numbers) > len(best.numbers) ||
					len(analysis.numbers) == len(best.numbers) &&
						vectorAOTConcreteParamCount(analysis) <
							vectorAOTConcreteParamCount(best)) {
			best = analysis
		}
	}
	return best
}

func vectorAOTConcreteParamCount(analysis *vectorAOTAnalysis) int {
	count := 0
	for index, typ := range analysis.paramTypes {
		if analysis.paramMask&(uint32(1)<<index) == 0 &&
			typ.Kind != compiler.IRDynamic {
			count++
		}
	}
	return count
}

func (a *vectorAOTAnalyzer) expr(
	node *ast.Node,
	locals map[*lang.Symbol]vectorAOTValue,
	tail bool,
) vectorAOTValue {
	if node == nil || !a.valid {
		return vectorAOTValue{}
	}
	var result vectorAOTValue
	switch node.Op {
	case ast.OpConst:
		result.typ = compiler.ConstantType(node.Sub.(*ast.ConstNode).Value)

	case ast.OpLocal:
		result = locals[node.Sub.(*ast.LocalNode).Name]

	case ast.OpDo:
		do := node.Sub.(*ast.DoNode)
		for _, statement := range do.Statements {
			a.expr(statement, locals, false)
		}
		result = a.expr(do.Ret, locals, tail)

	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		nested := cloneVectorAOTLocals(locals)
		for _, binding := range let.Bindings {
			binding := binding.Sub.(*ast.BindingNode)
			nested[binding.Name] = a.expr(binding.Init, nested, false)
		}
		result = a.expr(let.Body, nested, tail)

	case ast.OpLoop:
		loop := node.Sub.(*ast.LetNode)
		nested := cloneVectorAOTLocals(locals)
		expected := make([]vectorAOTValue, len(loop.Bindings))
		for index, binding := range loop.Bindings {
			binding := binding.Sub.(*ast.BindingNode)
			expected[index] = a.expr(binding.Init, nested, false)
			expected[index].control = false
			nested[binding.Name] = expected[index]
		}
		previous := a.loop
		a.loop = &vectorAOTLoop{
			id:       loop.LoopID,
			expected: expected,
			parent:   previous,
		}
		result = a.expr(loop.Body, nested, tail)
		a.loop = previous
		result.control = false

	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		if a.expr(conditional.Test, locals, false).transient {
			a.valid = false
			break
		}
		writtenBefore := a.written
		thenValue := a.expr(conditional.Then, locals, tail)
		thenWritten := a.written
		a.written = writtenBefore
		elseValue := a.expr(conditional.Else, locals, tail)
		a.written |= thenWritten
		result = a.combine(thenValue, elseValue)

	case ast.OpRecur:
		recur := node.Sub.(*ast.RecurNode)
		loop := a.findLoop(recur.LoopID)
		if loop == nil || len(recur.Exprs) != len(loop.expected) {
			a.valid = false
			break
		}
		for index, expression := range recur.Exprs {
			value := a.expr(expression, locals, false)
			expected := loop.expected[index]
			if value.transient != expected.transient ||
				value.transient && value.origins != expected.origins ||
				!value.transient && !expected.typ.Accepts(value.typ) {
				a.valid = false
			}
		}
		result.control = true

	case ast.OpAssoc:
		assoc := node.Sub.(*ast.AssocNode)
		target := a.expr(assoc.Target, locals, false)
		for _, entry := range assoc.Entries {
			key := a.expr(entry.Key, locals, false)
			value := a.expr(entry.Val, locals, false)
			if key.transient || value.transient ||
				target.transient &&
					(key.typ.Kind != compiler.IRInt ||
						value.typ.Kind != compiler.IRInt) {
				a.valid = false
			}
		}
		if target.transient {
			a.analysis.assocs[node] = true
			a.analysis.mutated |= target.origins
			a.written |= target.origins
			result = target
		}

	case ast.OpHostCall:
		call := node.Sub.(*ast.HostCallNode)
		values := make([]vectorAOTValue, len(call.Args))
		hasTransient := false
		for index, argument := range call.Args {
			values[index] = a.expr(argument, locals, false)
			hasTransient = hasTransient || values[index].transient
		}
		if hasTransient {
			if len(values) == 2 && values[0].transient &&
				values[1].typ.Kind == compiler.IRInt && isVectorAOTNth(call) {
				if a.written&values[0].origins != 0 {
					a.valid = false
					break
				}
				a.analysis.nths[node] = true
				result.typ = compiler.IRType{Kind: compiler.IRInt}
			} else {
				a.valid = false
			}
		} else if isVectorAOTNumbersCall(call) {
			types := make([]compiler.IRType, len(values))
			for index, value := range values {
				types[index] = value.typ
			}
			if inferred, ok := compiler.InferNumericHostCallType(
				call,
				types,
			); ok {
				a.analysis.numbers[node] = true
				result.typ = inferred
			}
		}

	case ast.OpInvoke:
		invoke := node.Sub.(*ast.InvokeNode)
		values := make([]vectorAOTValue, len(invoke.Args))
		hasTransient := false
		for index, argument := range invoke.Args {
			values[index] = a.expr(argument, locals, false)
			hasTransient = hasTransient || values[index].transient
		}
		if !hasTransient {
			break
		}
		callee := a.vectorCallee(invoke)
		if callee == nil {
			a.valid = false
			break
		}
		calleeAnalysis := callee.vectorAnalysis
		var origins uint32
		for index, value := range values {
			expectedVector :=
				calleeAnalysis.paramMask&(uint32(1)<<index) != 0
			if value.transient != expectedVector ||
				!expectedVector &&
					!calleeAnalysis.paramTypes[index].Accepts(value.typ) {
				a.valid = false
				break
			}
			if expectedVector {
				origins |= value.origins
			}
		}
		if a.valid {
			a.analysis.calls[node] = callee
			a.analysis.mutated |= origins
			a.written |= origins
			result = vectorAOTValue{transient: true, origins: origins}
		}
	case ast.OpVector:
		vector := node.Sub.(*ast.VectorNode)
		for _, item := range vector.Items {
			if a.expr(item, locals, false).transient {
				if !tail {
					a.valid = false
					break
				}
				a.analysis.freeze[item] = true
			}
		}

	default:
		// Keep this a deliberately small, auditable region. New forms are
		// enabled only after their transient identity and evaluation-order
		// behavior have been specified.
		a.valid = false
	}
	a.analysis.values[node] = result
	return result
}

func (a *vectorAOTAnalyzer) combine(
	left, right vectorAOTValue,
) vectorAOTValue {
	if left.control {
		return right
	}
	if right.control {
		return left
	}
	if left.transient != right.transient {
		a.valid = false
		return vectorAOTValue{}
	}
	if left.transient {
		return vectorAOTValue{
			transient: true,
			origins:   left.origins | right.origins,
		}
	}
	if left.typ == right.typ {
		return vectorAOTValue{typ: left.typ}
	}
	if left.typ.Kind == right.typ.Kind {
		return vectorAOTValue{typ: compiler.IRType{Kind: left.typ.Kind}}
	}
	return vectorAOTValue{
		typ: compiler.IRType{Kind: compiler.IRDynamic, Nullable: true},
	}
}

func (a *vectorAOTAnalyzer) findLoop(id *lang.Symbol) *vectorAOTLoop {
	for loop := a.loop; loop != nil; loop = loop.parent {
		if lang.Equals(loop.id, id) {
			return loop
		}
	}
	return nil
}

func (a *vectorAOTAnalyzer) vectorCallee(
	invoke *ast.InvokeNode,
) *aotSpecializationTarget {
	if invoke == nil || invoke.Fn.Op != ast.OpVar {
		return nil
	}
	target := a.targets[invoke.Fn.Sub.(*ast.VarNode).Var]
	if target == nil || target.vectorAnalysis == nil ||
		!target.vectorAnalysis.result.transient ||
		len(invoke.Args) != target.arity {
		return nil
	}
	return target
}

func cloneVectorAOTLocals(
	locals map[*lang.Symbol]vectorAOTValue,
) map[*lang.Symbol]vectorAOTValue {
	result := make(map[*lang.Symbol]vectorAOTValue, len(locals)+1)
	for name, value := range locals {
		result[name] = value
	}
	return result
}

func isVectorAOTNth(call *ast.HostCallNode) bool {
	if call == nil || call.Method == nil ||
		!strings.EqualFold(call.Method.Name(), "Nth") ||
		call.Target == nil || call.Target.Op != ast.OpConst {
		return false
	}
	target := call.Target.Sub.(*ast.ConstNode)
	return target.HostSymbol != nil &&
		target.HostSymbol.String() ==
			"github.com:glojurelang:glojure:pkg:runtime.RT"
}

func isVectorAOTNumbersCall(call *ast.HostCallNode) bool {
	if call == nil || call.Method == nil || call.Target == nil ||
		call.Target.Op != ast.OpConst {
		return false
	}
	target := call.Target.Sub.(*ast.ConstNode)
	if target.Value == lang.Numbers {
		return true
	}
	return target.HostSymbol != nil &&
		target.HostSymbol.String() ==
			"github.com:glojurelang:glojure:pkg:lang.Numbers"
}

func vectorAOTParamTypes(analysis *vectorAOTAnalysis) []string {
	params := make([]string, analysis.target.arity)
	for index := range params {
		if analysis.paramMask&(uint32(1)<<index) != 0 {
			params[index] = "*lang.TransientVector"
		} else {
			params[index] = vectorAOTIRGoType(analysis.paramTypes[index])
		}
	}
	return params
}

func vectorAOTGoType(value vectorAOTValue) string {
	if value.transient {
		return "*lang.TransientVector"
	}
	return vectorAOTIRGoType(value.typ)
}

func vectorAOTIRGoType(typ compiler.IRType) string {
	switch typ.Kind {
	case compiler.IRBool:
		return "bool"
	case compiler.IRInt:
		return "int64"
	case compiler.IRFloat:
		return "float64"
	default:
		return "any"
	}
}

func (g *Generator) generateVectorSpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn || target.vectorAnalysis == nil {
		return false
	}
	analysis := target.vectorAnalysis
	helper := g.allocateTempVar()
	typedParams := make([]string, method.FixedArity)
	paramTypes := vectorAOTParamTypes(analysis)
	for index, name := range paramNames {
		typedParams[index] = name + " " + paramTypes[index]
	}
	returnType := vectorAOTGoType(analysis.result)
	g.writef("var %s func(%s) %s\n",
		helper, strings.Join(typedParams, ", "), returnType)
	g.writef("%s = func(%s) %s {\n",
		helper, strings.Join(typedParams, ", "), returnType)
	previous := g.currentVector
	g.currentVector = analysis
	g.generateFnMethodFixed(method, paramNames)
	g.currentVector = previous
	g.writef("}\n")
	if analysis.result.transient {
		g.writef("%s = %s\n", target.vectorFnVar, helper)
	}

	signature := ""
	if method.FixedArity > 0 {
		signature = strings.Join(paramNames, ", ") + " any"
	}
	g.writef("%s = lang.FnFunc%d(func(%s) any {\n",
		fnVar, method.FixedArity, signature)
	fastArgs := append([]string(nil), paramNames...)
	var guards []string
	for index, param := range paramNames {
		if analysis.paramMask&(uint32(1)<<index) != 0 {
			value := g.allocateTempVar()
			ok := g.allocateTempVar()
			g.writef("%s, %s := %s.(*lang.Vector)\n", value, ok, param)
			g.writef(
				"%s = %s && lang.CanTransientlyUpdateInt64Vector(%s)\n",
				ok,
				ok,
				value,
			)
			fastArgs[index] = value + ".AsTransientForUpdate()"
			guards = append(guards, ok)
		} else if goType := vectorAOTIRGoType(
			analysis.paramTypes[index],
		); goType != "any" {
			value := g.allocateTempVar()
			ok := g.allocateTempVar()
			g.writef("%s, %s := %s.(%s)\n", value, ok, param, goType)
			fastArgs[index] = value
			guards = append(guards, ok)
		}
	}
	g.writef("if %s {\n", strings.Join(guards, " && "))
	result := g.allocateTempVar()
	g.writef("%s := %s(%s)\n",
		result, helper, strings.Join(fastArgs, ", "))
	if analysis.result.transient {
		g.writef("return %s.Persistent()\n", result)
	} else {
		g.writef("return %s\n", result)
	}
	g.writef("}\n")
	g.generateFnMethodFixed(method, paramNames)
	g.writef("})\n")
	return true
}

func (g *Generator) generateVectorAOTInvoke(
	node *ast.Node,
) (string, bool) {
	if g.currentVector == nil {
		return "", false
	}
	target := g.currentVector.calls[node]
	if target == nil {
		return "", false
	}
	invoke := node.Sub.(*ast.InvokeNode)
	args := make([]string, len(invoke.Args))
	for index, argument := range invoke.Args {
		args[index] = g.generateASTNode(argument)
	}
	result := g.allocateTempVar()
	g.writef("%s := %s(%s)\n",
		result,
		target.vectorFnVar,
		strings.Join(args, ", "),
	)
	return result, true
}

func (g *Generator) generateVectorAOTNth(
	node *ast.Node,
) (string, bool) {
	if g.currentVector == nil || !g.currentVector.nths[node] {
		return "", false
	}
	call := node.Sub.(*ast.HostCallNode)
	target := g.generateASTNode(call.Args[0])
	index := g.generateASTNode(call.Args[1])
	result := g.allocateTempVar()
	g.writef("%s := %s.Nth(lang.IntCast(%s)).(int64)\n",
		result, target, index)
	return result, true
}

func (g *Generator) generateVectorAOTAssoc(
	node *ast.Node,
) (string, bool) {
	if g.currentVector == nil || !g.currentVector.assocs[node] {
		return "", false
	}
	assoc := node.Sub.(*ast.AssocNode)
	target := g.generateASTNode(assoc.Target)
	keys := make([]string, len(assoc.Entries))
	values := make([]string, len(assoc.Entries))
	for index, entry := range assoc.Entries {
		keys[index] = g.generateASTNode(entry.Key)
		values[index] = g.generateASTNode(entry.Val)
	}
	for index := range assoc.Entries {
		g.writef("%s.AssocN(lang.IntCast(%s), %s)\n",
			target, keys[index], values[index])
	}
	return target, true
}

func (g *Generator) generateVectorAOTNumber(
	node *ast.Node,
) (string, bool) {
	if g.currentVector == nil || !g.currentVector.numbers[node] {
		return "", false
	}
	call := node.Sub.(*ast.HostCallNode)
	args := make([]string, len(call.Args))
	for index, argument := range call.Args {
		args[index] = g.generateASTNode(argument)
	}
	var expression string
	switch strings.ToLower(call.Method.Name()) {
	case "inc":
		expression = "lang.CheckedAddInt64(" + args[0] + ", 1)"
	case "unchecked_inc":
		expression = "(" + args[0] + " + 1)"
	case "dec":
		expression = "lang.CheckedSubInt64(" + args[0] + ", 1)"
	case "uncheckeddec", "unchecked_dec":
		expression = "(" + args[0] + " - 1)"
	case "iszero":
		expression = "(" + args[0] + " == 0)"
	case "ispos":
		expression = "(" + args[0] + " > 0)"
	case "isneg":
		expression = "(" + args[0] + " < 0)"
	case "lt":
		expression = "(" + args[0] + " < " + args[1] + ")"
	case "lte":
		expression = "(" + args[0] + " <= " + args[1] + ")"
	case "gt":
		expression = "(" + args[0] + " > " + args[1] + ")"
	case "gte":
		expression = "(" + args[0] + " >= " + args[1] + ")"
	case "add":
		expression = "lang.CheckedAddInt64(" + args[0] + ", " + args[1] + ")"
	case "uncheckedadd":
		expression = "(" + args[0] + " + " + args[1] + ")"
	case "minus":
		if len(args) == 1 {
			expression = "lang.CheckedSubInt64(0, " + args[0] + ")"
		} else {
			expression = "lang.CheckedSubInt64(" + args[0] + ", " + args[1] + ")"
		}
	case "unchecked_minus":
		if len(args) == 1 {
			expression = "(-" + args[0] + ")"
		} else {
			expression = "(" + args[0] + " - " + args[1] + ")"
		}
	case "multiply":
		expression = "lang.CheckedMultiplyInt64(" + args[0] + ", " + args[1] + ")"
	case "unchecked_multiply":
		expression = "(" + args[0] + " * " + args[1] + ")"
	case "max":
		expression = "max(" + args[0] + ", " + args[1] + ")"
	case "min":
		expression = "min(" + args[0] + ", " + args[1] + ")"
	}
	if expression == "" {
		return "", false
	}
	result := g.allocateTempVar()
	g.writef("%s := %s\n", result, expression)
	return result, true
}
