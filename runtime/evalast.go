package runtime

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/ast"
	"github.com/glojurelang/glojure/value"
)

// TODO: replace all usage of kw() with global vars

var indent = 0

const debug = true

func (env *environment) EvalAST(x interface{}) (ret interface{}, err error) {
	n := x.(ast.Node)

	if debug {
		fmt.Println(strings.Repeat(" ", indent), "BEG EvalAST", get(n, kw("op")), value.ToString(get(n, kw("form"))))
		indent += 2
		defer func() {
			indent -= 2
			fmt.Println(strings.Repeat(" ", indent), "END EvalAST", get(n, kw("op")), "->", ret, ",", err)
		}()
	}

	op := ast.Op(n)
	switch op {
	case kw("const"):
		return get(n, kw("val")), nil
	case kw("def"):
		return env.EvalASTDef(n)
	case kw("maybe-class"):
		return env.EvalASTMaybeClass(n)
	case kw("with-meta"):
		return env.EvalASTWithMeta(n)
	case kw("fn"):
		return env.EvalASTFn(n)
	case kw("map"):
		return env.EvalASTMap(n)
	case kw("vector"):
		return env.EvalASTVector(n)
	case kw("set"):
		return env.EvalASTSet(n)
	case kw("do"):
		return env.EvalASTDo(n)
	case kw("let"):
		return env.EvalASTLet(n, false)
	case kw("loop"):
		return env.EvalASTLet(n, true)
	case kw("invoke"):
		return env.EvalASTInvoke(n)
	case kw("quote"):
		return get(get(n, kw("expr")), kw("val")), nil
	case kw("var"):
		return env.EvalASTVar(n)
	case kw("local"):
		return env.EvalASTLocal(n)
	case kw("host-call"):
		return env.EvalASTHostCall(n)
	case kw("host-interop"):
		return env.EvalASTHostInterop(n)
	case kw("maybe-host-form"):
		return env.EvalASTMaybeHostForm(n)
	case kw("if"):
		return env.EvalASTIf(n)
	case kw("the-var"):
		return env.EvalASTTheVar(n)
	case kw("recur"):
		return env.EvalASTRecur(n)
	case kw("new"):
		return env.EvalASTNew(n)
	default:
		panic("unimplemented op: " + value.ToString(op) + "\n" + value.ToString(get(n, kw("form"))))
	}
}

func (env *environment) EvalASTDef(n ast.Node) (interface{}, error) {
	init := get(n, kw("init"))
	if init == nil {
		return get(n, kw("var")), nil
	}

	initVal, err := env.EvalAST(init.(ast.Node))
	if err != nil {
		return nil, err
	}

	return env.DefVar(get(n, kw("name")).(*value.Symbol), initVal), nil
}

func (env *environment) EvalASTTheVar(n ast.Node) (interface{}, error) {
	return get(n, kw("var")), nil
}

func (env *environment) EvalASTMaybeClass(n ast.Node) (interface{}, error) {
	// TODO: add go values to the namespace (without vars)
	sym := get(n, kw("class")).(*value.Symbol)
	name := sym.Name()
	if v, ok := env.scope.lookup(sym); ok {
		return v, nil
	}
	switch name {
	case "os.Exit":
		return os.Exit, nil
	case "fmt.Println":
		return fmt.Println, nil
	case "glojure.lang.NewList":
		return value.NewList, nil
	case "glojure.lang.WithMeta":
		return value.WithMeta, nil
	case "glojure.lang.NewCons":
		return value.NewCons, nil
	case "glojure.lang.NewLazilyPersistentVector":
		return value.NewLazilyPersistentVector, nil
	case "glojure.lang.Symbol":
		return reflect.TypeOf(value.NewSymbol("")), nil
	case "glojure.lang.IsInteger":
		return value.IsInteger, nil
	case "glojure.lang.AsInt64":
		return value.AsInt64, nil
	case "glojure.lang.Keyword":
		return reflect.TypeOf(value.NewKeyword("nop")), nil
	case "glojure.lang.RT":
		return value.RT, nil
	case "glojure.lang.Numbers":
		return value.Numbers, nil
	case "glojure.lang.NewMultiFn":
		return value.NewMultiFn, nil
	case "glojure.lang.IDrop":
		return reflect.TypeOf((*value.IDrop)(nil)).Elem(), nil
	default:
		return nil, errors.New("unknown Go value: " + value.ToString(get(n, kw("class"))))
	}
}

func (env *environment) EvalASTMaybeHostForm(n ast.Node) (interface{}, error) {
	// TODO: how to handle?
	panic("EvalASTMaybeHostForm")
}

func (env *environment) EvalASTHostCall(n ast.Node) (interface{}, error) {
	tgt := get(n, kw("target"))
	method := get(n, kw("method")).(*value.Symbol)
	args := get(n, kw("args"))

	tgtVal, err := env.EvalAST(tgt.(ast.Node))
	if err != nil {
		return nil, err
	}
	var argVals []interface{}
	for i := 0; i < value.Count(args); i++ {
		arg := get(args, i)
		argVal, err := env.EvalAST(arg.(ast.Node))
		if err != nil {
			return nil, err
		}
		argVals = append(argVals, argVal)
	}
	methodVal := value.FieldOrMethod(tgtVal, method.Name())
	if methodVal == nil {
		return nil, fmt.Errorf("no such field or method on %T: %s", tgtVal, method)
	}
	// if the field is not a function, return an error
	if reflect.TypeOf(methodVal).Kind() != reflect.Func {
		return nil, errors.New("not a method: " + value.ToString(tgtVal) + "." + method.Name())
	}

	return value.Apply(env, methodVal, argVals)
}

func (env *environment) EvalASTHostInterop(n ast.Node) (interface{}, error) {
	tgt := get(n, kw("target"))
	mOrF := get(n, kw("m-or-f")).(*value.Symbol)

	tgtVal, err := env.EvalAST(tgt.(ast.Node))
	if err != nil {
		return nil, err
	}

	mOrFVal := value.FieldOrMethod(tgtVal, mOrF.Name())
	switch reflect.TypeOf(mOrFVal).Kind() {
	case reflect.Func:
		return value.Apply(env, mOrFVal, nil)
	default:
		panic("uniimplemented")
	}
}

func (env *environment) EvalASTWithMeta(n ast.Node) (interface{}, error) {
	expr := get(n, kw("expr"))
	meta := get(n, kw("meta")).(value.IPersistentMap)
	exprVal, err := env.EvalAST(expr.(ast.Node))
	if err != nil {
		return nil, err
	}

	return value.WithMeta(exprVal, meta)
}

func (env *environment) EvalASTFn(n ast.Node) (interface{}, error) {
	return value.NewFn(n, env), nil
}

func (env *environment) EvalASTMap(n ast.Node) (interface{}, error) {
	res := value.NewMap()

	keys := get(n, kw("keys"))
	vals := get(n, kw("vals"))
	for i := 0; i < value.Count(keys); i++ {
		key := value.Get(keys, i)
		val := value.Get(vals, i)
		valVal, err := env.EvalAST(val.(ast.Node))
		if err != nil {
			return nil, err
		}
		value.Assoc(res, key, valVal)
	}

	return res, nil
}

func (env *environment) EvalASTVector(n ast.Node) (interface{}, error) {
	items := get(n, kw("items"))
	var vals []interface{}
	for i := 0; i < value.Count(items); i++ {
		item := get(items, i)
		itemVal, err := env.EvalAST(item.(ast.Node))
		if err != nil {
			return nil, err
		}
		vals = append(vals, itemVal)
	}
	return value.NewVector(vals...), nil
}

func (env *environment) EvalASTSet(n ast.Node) (interface{}, error) {
	items := get(n, kw("items"))
	var vals []interface{}
	for i := 0; i < value.Count(items); i++ {
		item := get(items, i)
		itemVal, err := env.EvalAST(item.(ast.Node))
		if err != nil {
			return nil, err
		}
		vals = append(vals, itemVal)
	}
	return value.NewSet(vals...), nil
}

func (env *environment) EvalASTIf(n ast.Node) (interface{}, error) {
	test := get(n, kw("test"))
	then := get(n, kw("then"))
	els := get(n, kw("else"))

	testVal, err := env.EvalAST(test.(ast.Node))
	if err != nil {
		return nil, err
	}
	if value.IsTruthy(testVal) {
		return env.EvalAST(then.(ast.Node))
	} else {
		return env.EvalAST(els.(ast.Node))
	}
}

func (env *environment) EvalASTDo(n ast.Node) (interface{}, error) {
	statements := get(n, kw("statements"))
	for i := 0; i < value.Count(statements); i++ {
		_, err := env.EvalAST(value.Get(statements, i).(ast.Node))
		if err != nil {
			return nil, err
		}
	}
	ret := get(n, kw("ret"))
	return env.EvalAST(ret.(ast.Node))
}

func (env *environment) EvalASTLet(n ast.Node, isLoop bool) (interface{}, error) {
	newEnv := env.PushScope().(*environment)

	var bindNameVals []interface{}

	bindings := get(n, kw("bindings"))
	for i := 0; i < value.Count(bindings); i++ {
		binding := get(bindings, i)
		name := get(binding, kw("name"))
		init := get(binding, kw("init"))
		initVal, err := newEnv.EvalAST(init.(ast.Node))
		if err != nil {
			return nil, err
		}
		// TODO: this should not mutate in-place!
		newEnv.BindLocal(name.(*value.Symbol), initVal)

		bindNameVals = append(bindNameVals, name, initVal)
	}

Recur:
	for i := 0; i < len(bindNameVals); i += 2 {
		name := bindNameVals[i].(*value.Symbol)
		val := bindNameVals[i+1]
		newEnv.BindLocal(name, val)
	}

	rt := value.NewRecurTarget()
	recurEnv := newEnv.WithRecurTarget(rt).(*environment)
	recurErr := &value.RecurError{Target: rt}

	res, err := recurEnv.EvalAST(get(n, kw("body")).(ast.Node))
	if isLoop && errors.As(err, &recurErr) {
		newVals := recurErr.Args
		if len(newVals) != len(bindNameVals)/2 {
			return nil, env.errorf(n, "invalid recur, expected %d arguments, got %d", len(bindNameVals)/2, len(newVals))
		}
		for i := 0; i < len(bindNameVals); i += 2 {
			newValsIndex := i / 2
			val := newVals[newValsIndex]
			bindNameVals[i+1] = val
		}
		goto Recur
	}
	return res, err
}

func (env *environment) EvalASTRecur(n ast.Node) (interface{}, error) {
	if env.recurTarget == nil {
		panic("recur outside of loop")
	}

	exprs := get(n, kw("exprs"))
	vals := make([]interface{}, 0, value.Count(exprs))
	noRecurEnv := env.WithRecurTarget(nil).(*environment)
	for seq := value.Seq(exprs); seq != nil; seq = seq.Next() {
		val, err := noRecurEnv.EvalAST(seq.First().(ast.Node))
		if err != nil {
			return nil, err
		}
		vals = append(vals, val)
	}
	return nil, &value.RecurError{
		Target: env.recurTarget,
		Args:   vals,
	}
}

func (env *environment) EvalASTInvoke(n ast.Node) (interface{}, error) {
	fn := get(n, kw("fn"))
	args := get(n, kw("args"))
	fnVal, err := env.EvalAST(fn.(ast.Node))
	if err != nil {
		return nil, err
	}

	var argVals []interface{}
	for i := 0; i < value.Count(args); i++ {
		arg := get(args, i)
		argVal, err := env.EvalAST(arg.(ast.Node))
		if err != nil {
			return nil, err
		}
		argVals = append(argVals, argVal)
	}

	return value.Apply(env, fnVal, argVals)
}

func (env *environment) EvalASTVar(n ast.Node) (interface{}, error) {
	return get(n, kw("var")).(*value.Var).Get(), nil
}

func (env *environment) EvalASTLocal(n ast.Node) (interface{}, error) {
	sym := get(n, kw("name")).(*value.Symbol)
	v, ok := env.scope.lookup(sym)
	if !ok {
		return nil, env.errorf(get(n, kw("form")), "unable to resolve symbol: %s", sym)
	}
	return v, nil
}

func (env *environment) EvalASTNew(n ast.Node) (interface{}, error) {
	classVal, err := env.EvalAST(get(n, kw("class")))
	if err != nil {
		return nil, err
	}
	if value.Count(get(n, kw("args"))) > 0 {
		return nil, errors.New("new with args unsupported")
	}
	classValTyp, ok := classVal.(reflect.Type)
	if !ok {
		return nil, fmt.Errorf("new value must be a reflect.Type, got %T", classVal)
	}
	return reflect.New(classValTyp).Interface(), nil
}

func kw(s string) value.Keyword {
	return value.NewKeyword(s)
}

func get(x interface{}, key interface{}) interface{} {
	return value.Get(x, key)
}
