package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// A positive count of a materialized slice needs neither numeric boxing nor
// the general count dispatcher. Restrict the argument to a local read so the
// guarded path cannot repeat argument side effects when it falls back.
func (c threadedEvalCompiler) compilePositiveCount(call *ast.HostCallNode, fallback evalFn) evalFn {
	if !isNumbersCall(call) || !strings.EqualFold(call.Method.Name(), "gt") || len(call.Args) != 2 {
		return fallback
	}
	zero, ok := int64ConstValue(call.Args[1])
	if !ok || zero != 0 || call.Args[0].Op != ast.OpInvoke {
		return fallback
	}
	count := call.Args[0].Sub.(*ast.InvokeNode)
	if count.Fn.Op != ast.OpVar || len(count.Args) != 1 || count.Args[0].Op != ast.OpLocal {
		return fallback
	}
	variable := count.Fn.Sub.(*ast.VarNode).Var
	if variable.Namespace() != lang.NSCore || variable.Symbol().Name() != "count" {
		return fallback
	}
	version := variable.RootVersion()
	if !IsDefaultCoreVar(variable) || variable.RootVersion() != version {
		return fallback
	}
	arg := c.compile(count.Args[0])
	if arg == nil {
		return fallback
	}
	return func(env *environment) (any, error) {
		if variable.RootVersion() == version && !variable.IsMacro() && !variable.IsDynamic() {
			value, err := arg(env)
			if err != nil {
				return nil, err
			}
			if value == nil {
				return false, nil
			}
			if seq, ok := value.(*lang.SliceSeq); ok && seq.HasValue() {
				return true, nil
			}
		}
		return fallback(env)
	}
}

func (c threadedEvalCompiler) compileCoreInvoke(node *ast.Node, invoke *ast.InvokeNode) evalFn {
	if invoke.Fn.Op != ast.OpVar {
		return nil
	}
	variable := invoke.Fn.Sub.(*ast.VarNode).Var
	if variable.Namespace() != lang.NSCore {
		return nil
	}
	name := variable.Symbol().Name()
	var unary func(any) any
	var binary func(any, any) any
	switch {
	case len(invoke.Args) == 1:
		switch name {
		case "count":
			unary = func(value any) any { return lang.Count(value) }
		case "seq":
			unary = func(value any) any { return lang.Seq(value) }
		case "first":
			unary = lang.First
		case "next":
			unary = func(value any) any { return lang.Next(value) }
		case "rest":
			unary = lang.Rest
		}
	case len(invoke.Args) == 2 && name == "=":
		binary = func(a, b any) any { return lang.Equiv(a, b) }
	}
	if unary == nil && binary == nil {
		return nil
	}
	version := variable.RootVersion()
	if !IsDefaultCoreVar(variable) || variable.RootVersion() != version {
		return nil
	}
	args := c.compileArgs(invoke.Args)
	if args == nil {
		return nil
	}
	// Locals and constants cannot invoke user code while being read. For these
	// arguments, recover only around calls that can actually panic.
	plainArgs := true
	for _, arg := range invoke.Args {
		if arg.Op != ast.OpConst && arg.Op != ast.OpLocal {
			plainArgs = false
		}
	}
	if unary != nil {
		arg := args[0]
		evaluate := func(env *environment) (result interface{}, err error) {
			// Choose the callee before evaluating arguments, which may
			// redefine this Var or introduce a dynamic binding.
			direct := variable.RootVersion() == version &&
				!variable.IsMacro() && !variable.IsDynamic()
			var fallback interface{}
			if !direct {
				fallback = variable.Get()
			}
			value, err := arg(env)
			if err != nil {
				return nil, err
			}
			if direct {
				if seq, ok := value.(*lang.SliceSeq); ok && seq.HasValue() {
					switch name {
					case "count":
						return seq.Count(), nil
					case "seq":
						return seq, nil
					case "first":
						return seq.First(), nil
					case "next":
						return seq.Next(), nil
					case "rest":
						return seq.More(), nil
					}
				}
				return invokeCoreUnary(env, node, unary, value)
			}
			return invokeCoreUnary(env, node, func(value any) any {
				return lang.Apply1(fallback, value)
			}, value)
		}
		if plainArgs {
			return evaluate
		}
		return func(env *environment) (result interface{}, err error) {
			defer env.recoverInvoke(node, &result, &err)
			return evaluate(env)
		}
	}
	arg0, arg1 := args[0], args[1]
	evaluate := func(env *environment) (result interface{}, err error) {
		direct := variable.RootVersion() == version &&
			!variable.IsMacro() && !variable.IsDynamic()
		var fallback interface{}
		if !direct {
			fallback = variable.Get()
		}
		a, err := arg0(env)
		if err != nil {
			return nil, err
		}
		b, err := arg1(env)
		if err != nil {
			return nil, err
		}
		if direct {
			if left, ok := a.(int64); ok {
				if right, ok := b.(int64); ok {
					return left == right, nil
				}
			}
			return invokeCoreBinary(env, node, binary, a, b)
		}
		return invokeCoreBinary(env, node, func(a, b any) any {
			return lang.Apply2(fallback, a, b)
		}, a, b)
	}
	if plainArgs {
		return evaluate
	}
	return func(env *environment) (result interface{}, err error) {
		defer env.recoverInvoke(node, &result, &err)
		return evaluate(env)
	}
}

func invokeCoreUnary(env *environment, node *ast.Node, fn func(any) any, value any) (result any, err error) {
	defer env.recoverInvoke(node, &result, &err)
	return fn(value), nil
}

func invokeCoreBinary(env *environment, node *ast.Node, fn func(any, any) any, a, b any) (result any, err error) {
	defer env.recoverInvoke(node, &result, &err)
	return fn(a, b), nil
}
