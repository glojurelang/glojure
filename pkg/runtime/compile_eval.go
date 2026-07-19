package runtime

import (
	"fmt"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// evalFn is a pre-threaded AST evaluator. compileEval deliberately supports a
// conservative set of stable expression forms and returns nil for anything
// else, preserving EvalAST as the complete semantic fallback.
type evalFn func(*environment) (interface{}, error)

func compileEval(n *ast.Node) evalFn {
	switch n.Op {
	case ast.OpConst:
		value := n.Sub.(*ast.ConstNode).Value
		return func(*environment) (interface{}, error) { return value, nil }

	case ast.OpLocal:
		sym := n.Sub.(*ast.LocalNode).Name
		return func(env *environment) (interface{}, error) {
			value, ok := env.scope.lookup(sym)
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
		test := compileEval(ifNode.Test)
		then := compileEval(ifNode.Then)
		els := compileEval(ifNode.Else)
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
			statements[i] = compileEval(statement)
			if statements[i] == nil {
				return nil
			}
		}
		ret := compileEval(doNode.Ret)
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

	case ast.OpHostCall:
		hostCall := n.Sub.(*ast.HostCallNode)
		if hostCall.ResolvedMethod == nil {
			return nil
		}
		args := compileEvalArgs(hostCall.Args)
		if args == nil {
			return nil
		}
		return compileResolvedCall(hostCall.ResolvedMethod, args)

	case ast.OpInvoke:
		invoke := n.Sub.(*ast.InvokeNode)
		fn := compileEval(invoke.Fn)
		args := compileEvalArgs(invoke.Args)
		if fn == nil || args == nil {
			return nil
		}
		return func(env *environment) (res interface{}, err error) {
			defer env.recoverInvoke(n, &res, &err)
			fnValue, err := fn(env)
			if err != nil {
				return nil, err
			}
			return evalCompiledCall(env, fnValue, args)
		}
	}
	return nil
}

func compileEvalArgs(nodes []*ast.Node) []evalFn {
	// Keep a non-nil empty slice so zero-argument calls remain compilable.
	args := make([]evalFn, len(nodes))
	for i, node := range nodes {
		args[i] = compileEval(node)
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
		return lang.Apply0(fn), nil
	case 1:
		a0, err := args[0](env)
		if err != nil {
			return nil, err
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
