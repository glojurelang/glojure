package runtime

import (
	"fmt"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// evalFn is a pre-threaded AST evaluator. compileEval deliberately supports a
// conservative set of stable expression forms and returns nil for anything
// else, preserving EvalAST as the complete semantic fallback.
type evalFn func(*environment) (interface{}, error)

type threadedEvalCompiler struct {
	localSlots  map[*lang.Symbol]localSlot
	nextLetSlot int
	typedLoop   bool
}

type localSlotKind uint8

const (
	fnLocalSlot localSlotKind = iota
	loopLocalSlot
	letLocalSlot
)

type localSlot struct {
	index       int
	kind        localSlotKind
	numericKind numericKind
}

func boxInt64(value int64) interface{} {
	return lang.BoxInt64(value)
}

func compileEval(n *ast.Node) evalFn {
	evaluator := (threadedEvalCompiler{}).compile(n)
	if evaluator == nil {
		return nil
	}
	return func(env *environment) (interface{}, error) {
		var frame evalFrame
		evalEnv := *env
		evalEnv.evalFrame = &frame
		return evaluator(&evalEnv)
	}
}

func compileMethodEval(body *ast.Node, params []*ast.Node) evalFn {
	slots := make(map[*lang.Symbol]localSlot, min(len(params), len(fnFrame{}.args)))
	for i, param := range params {
		if i == len(fnFrame{}.args) {
			break
		}
		slots[param.Sub.(*ast.BindingNode).Name] = localSlot{index: i}
	}
	return (threadedEvalCompiler{localSlots: slots}).compile(body)
}

func compileLoopEval(body *ast.Node, bindings []*ast.Node) evalFn {
	slots := make(map[*lang.Symbol]localSlot, min(len(bindings), len(loopFrame{}.args)))
	compiler := threadedEvalCompiler{localSlots: slots, typedLoop: true}
	for i, binding := range bindings {
		if i == len(loopFrame{}.args) {
			break
		}
		bindingNode := binding.Sub.(*ast.BindingNode)
		slots[bindingNode.Name] = localSlot{
			index:       i,
			kind:        loopLocalSlot,
			numericKind: compiler.inferNumericKind(bindingNode.Init),
		}
	}
	return compiler.compile(body)
}

func (c threadedEvalCompiler) compile(n *ast.Node) evalFn {
	switch n.Op {
	case ast.OpConst:
		value := n.Sub.(*ast.ConstNode).Value
		return func(*environment) (interface{}, error) { return value, nil }

	case ast.OpLocal:
		sym := n.Sub.(*ast.LocalNode).Name
		if slot, ok := c.localSlots[sym]; ok {
			switch slot.kind {
			case loopLocalSlot:
				return func(env *environment) (interface{}, error) {
					return env.loopFrame.args[slot.index], nil
				}
			case letLocalSlot:
				return func(env *environment) (interface{}, error) {
					return env.evalFrame.args[slot.index], nil
				}
			default:
				return func(env *environment) (interface{}, error) {
					return env.fnFrame.args[slot.index], nil
				}
			}
		}
		name := sym.String()
		return func(env *environment) (interface{}, error) {
			value, ok := env.scope.lookupName(name)
			if !ok {
				return nil, env.errorf(n.Form, "unable to resolve local symbol: %s", sym)
			}
			return value, nil
		}

	case ast.OpVar:
		vr := n.Sub.(*ast.VarNode).Var
		if vr.IsMacro() {
			return nil
		}
		return func(*environment) (interface{}, error) { return vr.Get(), nil }

	case ast.OpIf:
		ifNode := n.Sub.(*ast.IfNode)
		test := c.compile(ifNode.Test)
		then := c.compile(ifNode.Then)
		els := c.compile(ifNode.Else)
		if test == nil || then == nil || els == nil {
			return nil
		}
		return func(env *environment) (interface{}, error) {
			value, err := test(env)
			if err != nil {
				return nil, err
			}
			if lang.IsTruthy(value) {
				return then(env)
			}
			return els(env)
		}

	case ast.OpDo:
		doNode := n.Sub.(*ast.DoNode)
		statements := make([]evalFn, len(doNode.Statements))
		for i, statement := range doNode.Statements {
			statements[i] = c.compile(statement)
			if statements[i] == nil {
				return nil
			}
		}
		ret := c.compile(doNode.Ret)
		if ret == nil {
			return nil
		}
		return func(env *environment) (interface{}, error) {
			for _, statement := range statements {
				if _, err := statement(env); err != nil {
					return nil, err
				}
			}
			return ret(env)
		}

	case ast.OpLet:
		letNode := n.Sub.(*ast.LetNode)
		inits := make([]evalFn, len(letNode.Bindings))
		slots := make([]int, len(letNode.Bindings))
		letCompiler := c
		for i, binding := range letNode.Bindings {
			bindingNode := binding.Sub.(*ast.BindingNode)
			inits[i] = letCompiler.compile(bindingNode.Init)
			if inits[i] == nil {
				return nil
			}
			if letCompiler.nextLetSlot == len(evalFrame{}.args) {
				return nil
			}
			slots[i] = letCompiler.nextLetSlot
			numericKind := unknownNumericKind
			if letCompiler.typedLoop {
				numericKind = letCompiler.inferNumericKind(bindingNode.Init)
			}
			letCompiler = letCompiler.withLocalSlot(
				bindingNode.Name,
				localSlot{
					index:       slots[i],
					kind:        letLocalSlot,
					numericKind: numericKind,
				},
			)
			letCompiler.nextLetSlot++
		}
		body := letCompiler.compile(letNode.Body)
		if body == nil {
			return nil
		}
		return func(env *environment) (interface{}, error) {
			for i, init := range inits {
				value, err := init(env)
				if err != nil {
					return nil, err
				}
				env.evalFrame.args[slots[i]] = value
			}
			return body(env)
		}

	case ast.OpHostCall:
		hostCall := n.Sub.(*ast.HostCallNode)
		if hostCall.ResolvedMethod == nil {
			return nil
		}
		args := c.compileArgs(hostCall.Args)
		if args == nil {
			return nil
		}
		if numeric := compileNumberCall(hostCall, args); numeric != nil {
			if c.typedLoop {
				return c.compileNumericRegion(hostCall, numeric)
			}
			return numeric
		}
		return compileResolvedCall(hostCall.ResolvedMethod, args)

	case ast.OpInvoke:
		invoke := n.Sub.(*ast.InvokeNode)
		if pipeline := c.compileReducePipeline(n, invoke); pipeline != nil {
			return pipeline
		}
		fn := c.compile(invoke.Fn)
		args := c.compileArgs(invoke.Args)
		if fn == nil || args == nil {
			return nil
		}
		if invoke.Fn.Op == ast.OpVar && len(args) == 1 {
			vr := invoke.Fn.Sub.(*ast.VarNode).Var
			arg := args[0]
			return func(env *environment) (res interface{}, err error) {
				defer env.recoverInvoke(n, &res, &err)
				fnValue := vr.Get()
				a0, err := arg(env)
				if err != nil {
					return nil, err
				}
				if direct, ok := fnValue.(*Fn); ok {
					return direct.Invoke1(a0), nil
				}
				return lang.Apply1(fnValue, a0), nil
			}
		}
		if invoke.Fn.Op == ast.OpVar && len(args) == 2 {
			vr := invoke.Fn.Sub.(*ast.VarNode).Var
			arg0, arg1 := args[0], args[1]
			return func(env *environment) (res interface{}, err error) {
				defer env.recoverInvoke(n, &res, &err)
				fnValue := vr.Get()
				a0, err := arg0(env)
				if err != nil {
					return nil, err
				}
				a1, err := arg1(env)
				if err != nil {
					return nil, err
				}
				if direct, ok := fnValue.(*Fn); ok {
					return direct.Invoke2(a0, a1), nil
				}
				return lang.Apply2(fnValue, a0, a1), nil
			}
		}
		if invoke.Fn.Op == ast.OpVar && len(args) == 3 {
			vr := invoke.Fn.Sub.(*ast.VarNode).Var
			arg0, arg1, arg2 := args[0], args[1], args[2]
			return func(env *environment) (res interface{}, err error) {
				defer env.recoverInvoke(n, &res, &err)
				fnValue := vr.Get()
				a0, err := arg0(env)
				if err != nil {
					return nil, err
				}
				a1, err := arg1(env)
				if err != nil {
					return nil, err
				}
				a2, err := arg2(env)
				if err != nil {
					return nil, err
				}
				if direct, ok := fnValue.(*Fn); ok {
					return direct.Invoke3(a0, a1, a2), nil
				}
				return lang.Apply3(fnValue, a0, a1, a2), nil
			}
		}
		return func(env *environment) (res interface{}, err error) {
			defer env.recoverInvoke(n, &res, &err)
			fnValue, err := fn(env)
			if err != nil {
				return nil, err
			}
			return evalCompiledCall(env, fnValue, args)
		}

	case ast.OpKeywordLookup:
		lookup := n.Sub.(*ast.KeywordLookupNode)
		target := c.compile(lookup.Target)
		if target == nil {
			return nil
		}
		var fallback evalFn
		if lookup.Default != nil {
			fallback = c.compile(lookup.Default)
			if fallback == nil {
				return nil
			}
		}
		return func(env *environment) (res interface{}, err error) {
			defer env.recoverOptimizedInvoke(
				lookup.Meta,
				n.Form,
				&res,
				&err,
			)
			value, err := target(env)
			if err != nil {
				return nil, err
			}
			if fallback == nil {
				return lookup.Keyword.Invoke1(value), nil
			}
			defaultValue, err := fallback(env)
			if err != nil {
				return nil, err
			}
			return lookup.Keyword.Invoke2(value, defaultValue), nil
		}

	case ast.OpAssoc:
		assoc := n.Sub.(*ast.AssocNode)
		target := c.compile(assoc.Target)
		if target == nil {
			return nil
		}
		keys := make([]evalFn, len(assoc.Entries))
		values := make([]evalFn, len(assoc.Entries))
		for i, entry := range assoc.Entries {
			keys[i] = c.compile(entry.Key)
			values[i] = c.compile(entry.Val)
			if keys[i] == nil || values[i] == nil {
				return nil
			}
		}
		return func(env *environment) (res interface{}, err error) {
			defer env.recoverOptimizedInvoke(
				assoc.Meta,
				n.Form,
				&res,
				&err,
			)
			result, err := target(env)
			if err != nil {
				return nil, err
			}
			keyValues := make([]any, len(keys)*2)
			for i := range keys {
				keyValues[i*2], err = keys[i](env)
				if err != nil {
					return nil, err
				}
				keyValues[i*2+1], err = values[i](env)
				if err != nil {
					return nil, err
				}
			}
			for i := range keys {
				result = lang.Assoc(
					result,
					keyValues[i*2],
					keyValues[i*2+1],
				)
			}
			return result, nil
		}

	case ast.OpReplaceLast:
		replace := n.Sub.(*ast.ReplaceLastNode)
		collection := c.compile(replace.Collection)
		value := c.compile(replace.Value)
		if collection == nil || value == nil {
			return nil
		}
		return func(env *environment) (res interface{}, err error) {
			defer env.recoverReplaceLast(n, &res, &err)
			coll, err := collection(env)
			if err != nil {
				return nil, err
			}
			plan := PrepareReplaceLast(coll)
			replacement, err := value(env)
			if err != nil {
				return nil, err
			}
			return plan.Finish(replacement), nil
		}

	case ast.OpRecur:
		recur := n.Sub.(*ast.RecurNode)
		exprs := c.compileArgs(recur.Exprs)
		if exprs == nil {
			return nil
		}
		return func(env *environment) (interface{}, error) {
			recurErr := env.recurErr
			if recurErr == nil {
				recurErr = &lang.RecurError{Target: env.recurTarget}
			}
			if cap(recurErr.Args) < len(exprs) {
				recurErr.Args = make([]interface{}, len(exprs))
			} else {
				recurErr.Args = recurErr.Args[:len(exprs)]
			}

			target := env.recurTarget
			previousErr := env.recurErr
			env.recurTarget = nil
			env.recurErr = nil
			for i, expr := range exprs {
				value, err := expr(env)
				if err != nil {
					env.recurTarget = target
					env.recurErr = previousErr
					return nil, err
				}
				recurErr.Args[i] = value
			}
			env.recurTarget = target
			env.recurErr = previousErr
			return nil, recurErr
		}
	}
	return nil
}

func (c threadedEvalCompiler) withLocalSlot(sym *lang.Symbol, slot localSlot) threadedEvalCompiler {
	slots := make(map[*lang.Symbol]localSlot, len(c.localSlots)+1)
	for existing, existingSlot := range c.localSlots {
		slots[existing] = existingSlot
	}
	slots[sym] = slot
	c.localSlots = slots
	return c
}

func compileNumberCall(call *ast.HostCallNode, args []evalFn) evalFn {
	if !isNumbersCall(call) {
		return nil
	}
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		arg := args[0]
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec":
			return func(env *environment) (interface{}, error) {
				value, err := arg(env)
				if err != nil {
					return nil, err
				}
				if n, ok := value.(int64); ok {
					switch name {
					case "inc":
						return boxInt64(checkedInt64Add(n, 1)), nil
					case "dec":
						return boxInt64(checkedInt64Sub(n, 1)), nil
					case "unchecked_inc":
						return boxInt64(n + 1), nil
					default:
						return boxInt64(n - 1), nil
					}
				}
				return lang.Apply1(call.ResolvedMethod, value), nil
			}
		case "iszero", "ispos", "isneg":
			return func(env *environment) (interface{}, error) {
				value, err := arg(env)
				if err != nil {
					return nil, err
				}
				if n, ok := value.(int64); ok {
					switch name {
					case "iszero":
						return n == 0, nil
					case "ispos":
						return n > 0, nil
					default:
						return n < 0, nil
					}
				}
				return lang.Apply1(call.ResolvedMethod, value), nil
			}
		}
		return nil
	}
	if len(args) != 2 {
		return nil
	}

	left, right := args[0], args[1]
	switch name {
	case "add", "uncheckedadd", "minus", "unchecked_minus",
		"multiply", "unchecked_multiply", "quotient", "remainder",
		"lt", "lte", "gt", "gte", "equiv":
		return func(env *environment) (interface{}, error) {
			a, err := left(env)
			if err != nil {
				return nil, err
			}
			b, err := right(env)
			if err != nil {
				return nil, err
			}
			ai, aok := a.(int64)
			bi, bok := b.(int64)
			if aok && bok {
				switch name {
				case "add":
					return boxInt64(checkedInt64Add(ai, bi)), nil
				case "uncheckedadd":
					return boxInt64(ai + bi), nil
				case "minus":
					return boxInt64(checkedInt64Sub(ai, bi)), nil
				case "unchecked_minus":
					return boxInt64(ai - bi), nil
				case "multiply":
					return boxInt64(checkedInt64Multiply(ai, bi)), nil
				case "unchecked_multiply":
					return boxInt64(ai * bi), nil
				case "quotient":
					return boxInt64(checkedInt64Quotient(ai, bi)), nil
				case "remainder":
					return boxInt64(checkedInt64Remainder(ai, bi)), nil
				case "lt":
					return ai < bi, nil
				case "lte":
					return ai <= bi, nil
				case "gt":
					return ai > bi, nil
				case "gte":
					return ai >= bi, nil
				case "equiv":
					return ai == bi, nil
				}
			}
			return lang.Apply2(call.ResolvedMethod, a, b), nil
		}
	}
	return nil
}

func (c threadedEvalCompiler) compileArgs(nodes []*ast.Node) []evalFn {
	// Keep a non-nil empty slice so zero-argument calls remain compilable.
	args := make([]evalFn, len(nodes))
	for i, node := range nodes {
		args[i] = c.compile(node)
		if args[i] == nil {
			return nil
		}
	}
	return args
}

func compileResolvedCall(fn interface{}, args []evalFn) evalFn {
	switch len(args) {
	case 0:
		return func(*environment) (interface{}, error) {
			return lang.Apply0(fn), nil
		}
	case 1:
		if direct, ok := fn.(lang.FnFunc1); ok {
			return func(env *environment) (interface{}, error) {
				a0, err := args[0](env)
				if err != nil {
					return nil, err
				}
				return direct(a0), nil
			}
		}
	case 2:
		if direct, ok := fn.(lang.FnFunc2); ok {
			return func(env *environment) (interface{}, error) {
				a0, err := args[0](env)
				if err != nil {
					return nil, err
				}
				a1, err := args[1](env)
				if err != nil {
					return nil, err
				}
				return direct(a0, a1), nil
			}
		}
	}
	return func(env *environment) (interface{}, error) {
		return evalCompiledCall(env, fn, args)
	}
}

func evalCompiledCall(env *environment, fn interface{}, args []evalFn) (interface{}, error) {
	switch len(args) {
	case 0:
		if direct, ok := fn.(*Fn); ok {
			return direct.Invoke0(), nil
		}
		return lang.Apply0(fn), nil
	case 1:
		a0, err := args[0](env)
		if err != nil {
			return nil, err
		}
		if direct, ok := fn.(*Fn); ok {
			return direct.Invoke1(a0), nil
		}
		return lang.Apply1(fn, a0), nil
	case 2:
		a0, err := args[0](env)
		if err != nil {
			return nil, err
		}
		a1, err := args[1](env)
		if err != nil {
			return nil, err
		}
		if direct, ok := fn.(*Fn); ok {
			return direct.Invoke2(a0, a1), nil
		}
		return lang.Apply2(fn, a0, a1), nil
	case 3:
		a0, err := args[0](env)
		if err != nil {
			return nil, err
		}
		a1, err := args[1](env)
		if err != nil {
			return nil, err
		}
		a2, err := args[2](env)
		if err != nil {
			return nil, err
		}
		if direct, ok := fn.(*Fn); ok {
			return direct.Invoke3(a0, a1, a2), nil
		}
		return lang.Apply3(fn, a0, a1, a2), nil
	case 4:
		a0, err := args[0](env)
		if err != nil {
			return nil, err
		}
		a1, err := args[1](env)
		if err != nil {
			return nil, err
		}
		a2, err := args[2](env)
		if err != nil {
			return nil, err
		}
		a3, err := args[3](env)
		if err != nil {
			return nil, err
		}
		if direct, ok := fn.(*Fn); ok {
			return direct.Invoke4(a0, a1, a2, a3), nil
		}
		return lang.Apply4(fn, a0, a1, a2, a3), nil
	default:
		values := make([]interface{}, len(args))
		for i, arg := range args {
			value, err := arg(env)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		if !lang.CanApply(fn) {
			return nil, fmt.Errorf("cannot apply non-function %T", fn)
		}
		return lang.Apply(fn, values), nil
	}
}
