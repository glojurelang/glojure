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

// TODO: rename to Invoke
func (fn *Fn) Apply(env Environment, args []interface{}) (interface{}, error) {
	methods := Get(fn.astNode, NewKeyword("methods"))
	variadic := Get(fn.astNode, NewKeyword("variadic?")).(bool)
	maxArity, _ := AsInt(Get(fn.astNode, NewKeyword("max-fixed-arity")))

	if !variadic && len(args) > maxArity {
		return nil, fmt.Errorf("too many arguments (%d)", len(args))
	}

	method, err := fn.findMethod(methods, args)
	if err != nil {
		return nil, err
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
		return nil, errorWithStack(err, StackFrame{})
	}
	return res, nil
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
	var argSlice []interface{}
	for seq := Seq(args); seq != nil; seq = seq.Next() {
		argSlice = append(argSlice, seq.First())
	}
	return fn.Invoke(argSlice...)
}

func (fn *Fn) Invoke(args ...interface{}) interface{} {
	res, err := fn.Apply(nil, args) // TODO: global/singleton env
	if err != nil {
		panic(err)
	}
	return res
}
