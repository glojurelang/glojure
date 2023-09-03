package runtime

import (
	"errors"
	"fmt"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type Fn struct {
	meta lang.IPersistentMap

	astNode *ast.Node
	env     lang.Environment
}

var (
	_ lang.IObj = (*Fn)(nil)
)

func NewFn(astNode *ast.Node, env lang.Environment) *Fn {
	return &Fn{astNode: astNode, env: env}
}

func (fn *Fn) Meta() lang.IPersistentMap {
	return fn.meta
}

func (fn *Fn) WithMeta(meta lang.IPersistentMap) interface{} {
	cpy := *fn
	cpy.meta = meta
	return &cpy
}

func (fn *Fn) Invoke(args ...interface{}) interface{} {
	fnNode := fn.astNode.Sub.(*ast.FnNode)

	methods := fnNode.Methods
	variadic := fnNode.IsVariadic
	maxArity := fnNode.MaxFixedArity

	if !variadic && len(args) > maxArity {
		panic(lang.NewIllegalArgumentError(fmt.Sprintf("too many arguments (%d)", len(args))))
	}

	method, err := fn.findMethod(methods, args)
	if err != nil {
		panic(err)
	}

	fnEnv := fn.env.PushScope()
	if fnNode.Local != nil {
		localNode := fnNode.Local.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(localNode.Name, fn)
	}

	methodNode := method.Sub.(*ast.FnMethodNode)

	fixedArity := methodNode.FixedArity
	methodVariadic := methodNode.IsVariadic
	body := methodNode.Body

	bindingValues := args[:fixedArity]

	arity := fixedArity
	var bindingRestValue interface{}
	if len(args) > len(bindingValues) {
		arity++
		bindingRestValue = lang.NewList(args[len(bindingValues):]...)
	}

Recur:

	params := methodNode.Params
	for i, paramValue := range bindingValues {
		param := params[i]
		paramNode := param.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(paramNode.Name, paramValue)
	}
	if bindingRestValue != nil {
		param := params[len(params)-1]
		paramNode := param.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(paramNode.Name, bindingRestValue)
	} else if methodVariadic {
		param := params[len(params)-1]
		paramNode := param.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(paramNode.Name, nil)
	}

	rt := lang.NewRecurTarget()
	recurEnv := fnEnv.WithRecurTarget(rt)
	recurErr := &lang.RecurError{Target: rt}
	res, err := recurEnv.EvalAST(body)
	if errors.As(err, &recurErr) {
		if len(recurErr.Args) != arity {
			panic("wrong number of arguments to recur")
		}
		bindingRestValue = nil
		bindingValues = recurErr.Args[:fixedArity]
		if len(recurErr.Args) > fixedArity {
			bindingRestValue = recurErr.Args[fixedArity]
		}
		goto Recur
	}
	if err != nil {
		panic(errorWithStack(err, lang.StackFrame{})) // TODO: think through error stacks
	}
	return res
}

func (fn *Fn) findMethod(methods []*ast.Node, args []interface{}) (*ast.Node, error) {
	var variadicMethod *ast.Node
	for _, method := range methods {
		methodNode := method.Sub.(*ast.FnMethodNode)
		if methodNode.IsVariadic {
			variadicMethod = method
			continue
		}
		if methodNode.FixedArity == len(args) {
			return method, nil
		}
	}
	if variadicMethod == nil || len(args) < variadicMethod.Sub.(*ast.FnMethodNode).FixedArity {
		return nil, lang.NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments (%d)", len(args)))
	}
	return variadicMethod, nil
}

func (fn *Fn) ApplyTo(args lang.ISeq) interface{} {
	return fn.Invoke(seqToSlice(args)...)
}

func errorWithStack(err error, stackFrame lang.StackFrame) error {
	if err == nil {
		return nil
	}
	valErr, ok := err.(*lang.Error)
	if !ok {
		return lang.NewError(stackFrame, err)
	}
	return valErr.AddStack(stackFrame)
}
