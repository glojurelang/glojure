package value

import (
	"errors"
	"fmt"
)

type Fn struct {
	meta IPersistentMap

	astNode interface{}
	env     Environment
}

var (
	_ IObj = (*Fn)(nil)
)

func NewFn(astNode interface{}, env Environment) *Fn {
	return &Fn{astNode: astNode, env: env}
}

func (fn *Fn) Meta() IPersistentMap {
	return fn.meta
}

func (fn *Fn) WithMeta(meta IPersistentMap) interface{} {
	cpy := *fn
	cpy.meta = meta
	return &cpy
}

func (fn *Fn) Invoke(args ...interface{}) interface{} {
	methods := Get(fn.astNode, NewKeyword("methods"))
	variadic := Get(fn.astNode, NewKeyword("variadic?")).(bool)
	maxArity, _ := AsInt(Get(fn.astNode, NewKeyword("max-fixed-arity")))

	if !variadic && len(args) > maxArity {
		panic(fmt.Errorf("too many arguments (%d)", len(args)))
	}

	method, err := fn.findMethod(methods, args)
	if err != nil {
		panic(err)
	}

	fnEnv := fn.env.PushScope()

	fixedArity := Get(method, NewKeyword("fixed-arity")).(int)
	methodVariadic := Get(method, NewKeyword("variadic?")).(bool)
	body := Get(method, NewKeyword("body"))

	bindingValues := args[:fixedArity]

	arity := fixedArity
	var bindingRestValue interface{}
	if len(args) > len(bindingValues) {
		arity++
		bindingRestValue = NewList(args[len(bindingValues):]...)
	}

Recur:

	params := Get(method, NewKeyword("params"))
	for i, paramValue := range bindingValues {
		param := MustNth(params, i)
		fnEnv.BindLocal(Get(param, NewKeyword("name")).(*Symbol), paramValue)
	}
	if bindingRestValue != nil {
		fnEnv.BindLocal(Get(MustNth(params, fixedArity), NewKeyword("name")).(*Symbol), bindingRestValue)
	} else if methodVariadic {
		fnEnv.BindLocal(Get(MustNth(params, fixedArity), NewKeyword("name")).(*Symbol), nil)
	}

	rt := NewRecurTarget()
	recurEnv := fnEnv.WithRecurTarget(rt)
	recurErr := &RecurError{Target: rt}
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
		panic(errorWithStack(err, StackFrame{}))
	}
	return res
}

func (fn *Fn) findMethod(methods interface{}, args []interface{}) (interface{}, error) {
	var variadicMethod interface{}
	for mths := Seq(methods); mths != nil; mths = mths.Next() {
		method := mths.First()
		if Get(method, NewKeyword("variadic?")).(bool) {
			variadicMethod = method
			continue
		}
		if Get(method, NewKeyword("fixed-arity")).(int) == len(args) {
			return method, nil
		}
	}
	if variadicMethod == nil || len(args) < Get(variadicMethod, NewKeyword("fixed-arity")).(int) {
		return nil, fmt.Errorf("wrong number of arguments (%d)", len(args))
	}
	return variadicMethod, nil
}

// TODO: finish migration from Applyer to IFn

func (fn *Fn) ApplyTo(args ISeq) interface{} {
	return fn.Invoke(seqToSlice(args)...)
}

func errorWithStack(err error, stackFrame StackFrame) error {
	if err == nil {
		return nil
	}
	valErr, ok := err.(*Error)
	if !ok {
		return NewError(stackFrame, err)
	}
	return valErr.AddStack(stackFrame)
}
