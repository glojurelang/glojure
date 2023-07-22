package runtime

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	value "github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"

	// Make it easier to refer to KW globals
	. "github.com/glojurelang/glojure/pkg/lang"
)

var indent = 0

var (
	Debug = false

	SymNS   = value.NewSymbol("ns")
	SymInNS = value.NewSymbol("in-ns")
)

type EvalError struct {
	Err      error
	GLJStack []string
	GoStack  string
}

func (e *EvalError) Error() string {
	sb := strings.Builder{}
	sb.WriteString(e.Err.Error())
	sb.WriteString("\n\n")
	if e.GoStack != "" && false {
		sb.WriteString("Go Stack:\n")
		sb.WriteString(e.GoStack)
		sb.WriteString("\n\n")
	}
	sb.WriteString("GLJ Stack:\n")
	for _, s := range e.GLJStack {
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (env *environment) EvalAST(x interface{}) (ret interface{}, err error) {
	n := x.(ast.Node)

	op := ast.Op(n)
	switch op {
	case KWConst:
		return get(n, KWVal), nil
	case KWDef:
		return env.EvalASTDef(n)
	case KWSetBang:
		return env.EvalASTAssign(n)
	case KWMaybeClass:
		return env.EvalASTMaybeClass(n)
	case KWWithMeta:
		return env.EvalASTWithMeta(n)
	case KWFn:
		return env.EvalASTFn(n)
	case KWMap:
		return env.EvalASTMap(n)
	case KWVector:
		return env.EvalASTVector(n)
	case KWSet:
		return env.EvalASTSet(n)
	case KWDo:
		return env.EvalASTDo(n)
	case KWLet:
		return env.EvalASTLet(n, false)
	case KWLoop:
		return env.EvalASTLet(n, true)
	case KWInvoke:
		return env.EvalASTInvoke(n)
	case KWQuote:
		return get(get(n, KWExpr), KWVal), nil
	case KWVar:
		return env.EvalASTVar(n)
	case KWLocal:
		return env.EvalASTLocal(n)
	case KWHostCall:
		return env.EvalASTHostCall(n)
	case KWHostInterop:
		return env.EvalASTHostInterop(n)
	case KWMaybeHostForm:
		return env.EvalASTMaybeHostForm(n)
	case KWIf:
		return env.EvalASTIf(n)
	case KWCase:
		return env.EvalASTCase(n)
	case KWTheVar:
		return env.EvalASTTheVar(n)
	case KWRecur:
		return env.EvalASTRecur(n)
	case KWNew:
		return env.EvalASTNew(n)
	case KWTry:
		return env.EvalASTTry(n)
	case KWThrow:
		return env.EvalASTThrow(n)
	default:
		panic("unimplemented op: " + value.ToString(op) + "\n" + value.ToString(get(n, KWForm)))
	}
}

func (env *environment) EvalASTDef(n ast.Node) (interface{}, error) {
	init := get(n, KWInit)
	if value.IsNil(init) {
		return get(n, KWVar), nil
	}

	initVal, err := env.EvalAST(init.(ast.Node))
	if err != nil {
		return nil, err
	}
	sym := get(n, KWName).(*value.Symbol)
	// evaluate symbol metadata if present
	meta := get(n, KWMeta)
	if !value.IsNil(meta) {
		metaVal, err := env.EvalAST(meta.(ast.Node))
		if err != nil {
			return nil, err
		}
		s, err := value.WithMeta(sym, metaVal.(value.IPersistentMap))
		if err != nil {
			return nil, err
		}
		sym = s.(*value.Symbol)
	}

	vr := env.DefVar(sym, initVal)
	if RT.BooleanCast(get(vr.Meta(), value.KWDynamic)) {
		vr.SetDynamic()
	}
	return vr, nil
}

func (env *environment) EvalASTAssign(n ast.Node) (interface{}, error) {
	val, err := env.EvalAST(get(n, KWVal).(ast.Node))
	if err != nil {
		return nil, err
	}
	target := get(n, KWTarget)
	switch get(target, KWOp) {
	case KWVar:
		tgtVar := get(target, KWVar).(*value.Var)
		return tgtVar.Set(val), nil
	case KWHostInterop:
		interopTargetVal, err := env.EvalAST(get(target, KWTarget).(ast.Node))
		if err != nil {
			return nil, err
		}
		field := get(target, KWMOrF).(*value.Symbol)

		targetV := reflect.ValueOf(interopTargetVal)
		if targetV.Kind() == reflect.Ptr {
			targetV = targetV.Elem()
		}
		fieldVal := targetV.FieldByName(field.Name())
		if !fieldVal.IsValid() {
			return nil, fmt.Errorf("no such field %s", field.Name())
		}
		if !fieldVal.CanSet() {
			return nil, fmt.Errorf("cannot set field %s", field.Name())
		}
		valV := reflect.ValueOf(val)
		if !valV.IsValid() {
			switch fieldVal.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
				fieldVal.Set(reflect.Zero(fieldVal.Type()))
			default:
				return nil, fmt.Errorf("cannot set field %s to nil", field.Name())
			}
		} else {
			fieldVal.Set(valV)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported assign target: %v", get(target, KWForm))
	}
}

func (env *environment) EvalASTTheVar(n ast.Node) (interface{}, error) {
	return get(n, KWVar), nil
}

// TEMP
// TODO: add a compiler struct
type evalCompiler struct {
	env *environment
}

func (c *evalCompiler) Eval(form interface{}) interface{} {
	res, err := c.env.Eval(form)
	if err != nil {
		panic(err)
	}
	return res
}

func (c *evalCompiler) Macroexpand1(form interface{}) interface{} {
	res, err := c.env.Macroexpand1(form)
	if err != nil {
		panic(err)
	}
	return res
}

func (c *evalCompiler) MaybeResolveIn(ns *value.Namespace, sym *value.Symbol) interface{} {
	switch {
	case sym.Namespace() != "":
		n := value.NamespaceFor(ns, sym)
		if n == nil {
			return nil
		}
		return n.FindInternedVar(value.NewSymbol(sym.Name()))
	case strings.Index(sym.Name(), ".") > 0 && !strings.HasSuffix(sym.Name(), ".") || sym.Name()[0] == '[':
		panic(fmt.Errorf("can't resolve class"))
	case sym.Equal(SymNS):
		return value.VarNS
	case sym.Equal(SymInNS):
		return value.VarInNS
	default:
		return ns.GetMapping(sym)
	}
}

func (env *environment) EvalASTMaybeClass(n ast.Node) (interface{}, error) {
	// TODO: add go values to the namespace (without vars)
	sym := get(n, KWClass).(*value.Symbol)
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
	case "glojure.lang.InternSymbol":
		return value.InternSymbol, nil
	case "glojure.lang.InternKeyword":
		return value.InternKeyword, nil
	case "glojure.lang.IsInteger":
		return value.IsInteger, nil
	case "glojure.lang.AsInt64":
		return value.AsInt64, nil
	case "glojure.lang.Keyword":
		return reflect.TypeOf(value.KWName), nil
	case "glojure.lang.RT":
		return RT, nil
	case "glojure.lang.Numbers":
		return value.Numbers, nil
	case "glojure.lang.NewMultiFn":
		return value.NewMultiFn, nil
	case "glojure.lang.IDrop":
		return reflect.TypeOf((*value.IDrop)(nil)).Elem(), nil
	case "glojure.lang.Compiler":
		return &evalCompiler{env: env}, nil
	case "glojure.lang.Ref":
		return reflect.TypeOf(&value.Ref{}), nil
	case "glojure.lang.NewRef":
		return value.NewRef, nil
	case "glojure.lang.Named":
		return reflect.TypeOf((*value.Named)(nil)).Elem(), nil
	case "glojure.lang.Counted":
		return reflect.TypeOf((*value.Counted)(nil)).Elem(), nil
	case "glojure.lang.FindNamespace":
		return value.FindNamespace, nil
	case "glojure.lang.NewRepeat":
		return value.NewRepeat, nil
	case "glojure.lang.NewRepeatN":
		return value.NewRepeatN, nil
	case "glojure.lang.PushThreadBindings":
		return value.PushThreadBindings, nil
	case "glojure.lang.PopThreadBindings":
		return value.PopThreadBindings, nil
	default:
		v, ok := pkgmap.Get(sym.FullName())
		if ok {
			return v, nil
		}

		return nil, errors.New("unable to resolve symbol: " + value.ToString(get(n, KWClass)))
	}
}

func (env *environment) EvalASTMaybeHostForm(n ast.Node) (interface{}, error) {
	// TODO: implement this for real
	switch get(n, KWClass).(string) {
	case "glojure.lang.PersistentTreeSet":
		switch get(n, KWField).(*value.Symbol).Name() {
		case "create":
			return func(keys interface{}) interface{} {
				var ks []interface{}
				for seq := value.Seq(keys); seq != nil; seq = seq.Next() {
					ks = append(ks, seq.First())
				}
				return value.NewSet(ks...)
			}, nil
		}
	case "go":
		switch get(n, KWField).(*value.Symbol).Name() {
		case "int":
			return reflect.TypeOf(int(0)), nil
		case "byte":
			return reflect.TypeOf(byte(0)), nil
		case "rune":
			return reflect.TypeOf(rune(0)), nil
		case "append":
			return func(slc interface{}, vals ...interface{}) interface{} {
				slcVal := reflect.ValueOf(slc)
				slcTyp := slcVal.Type().Elem()
				valSlc := reflect.MakeSlice(reflect.SliceOf(slcTyp), len(vals), len(vals))
				for i, v := range vals {
					valSlc.Index(i).Set(reflect.ValueOf(v))
				}
				return reflect.AppendSlice(slcVal, valSlc).Interface()
			}, nil
		case "sliceof":
			return func(t reflect.Type, sizeCap ...interface{}) interface{} {
				if len(sizeCap) > 2 {
					panic("go/sliceof: too many arguments")
				}
				l, c := 0, 0
				var ok bool
				if len(sizeCap) > 0 {
					l, ok = value.AsInt(sizeCap[0])
					if !ok {
						panic("go/sliceof: length is not an integer")
					}
				}
				if len(sizeCap) > 1 {
					c, ok = value.AsInt(sizeCap[1])
					if !ok {
						panic("go/sliceof: capacity is not an integer")
					}
				}
				if c < l {
					c = l
				}
				return reflect.MakeSlice(reflect.SliceOf(t), l, c).Interface()
			}, nil
		case "slice":
			return func(sliceOrString interface{}, indices ...interface{}) interface{} {
				if len(indices) == 0 || len(indices) > 2 {
					panic("go/slice: must have 1 or 2 indices")
				}
				var start, end int64 = -1, -1
				if !value.IsNil(indices[0]) {
					start = value.AsInt64(indices[0])
				}
				if len(indices) == 2 && !value.IsNil(indices[1]) {
					end = value.AsInt64(indices[1])
				}
				if str, ok := sliceOrString.(string); ok {
					if start == -1 {
						start = 0
					}
					if end == -1 {
						end = int64(len(str))
					}
					return str[start:end]
				}
				sVal := reflect.ValueOf(sliceOrString)
				if sVal.Kind() != reflect.Slice {
					panic(fmt.Sprintf("go/slice: %v is not a slice or string", sliceOrString))
				}
				if start == -1 {
					start = 0
				}
				if end == -1 {
					end = int64(sVal.Len())
				}
				return sVal.Slice(int(start), int(end)).Interface()
			}, nil
		}
	}

	// TODO: how to handle?
	fmt.Println("EvalASTMaybeHostForm: ", n)
	panic("EvalASTMaybeHostForm: " + get(n, KWClass).(string))
}

func (env *environment) EvalASTHostCall(n ast.Node) (interface{}, error) {
	tgt := get(n, KWTarget)
	method := get(n, KWMethod).(*value.Symbol)
	args := get(n, KWArgs)

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
	methodVal, ok := value.FieldOrMethod(tgtVal, method.Name())
	if !ok {
		return nil, fmt.Errorf("no such field or method on %v (%T): %s", tgtVal, tgtVal, method)
	}
	// if the field is not a function, return an error
	if reflect.TypeOf(methodVal).Kind() != reflect.Func {
		return nil, errors.New("not a method: " + value.ToString(tgtVal) + "." + method.Name())
	}

	return value.Apply(methodVal, argVals)
}

func (env *environment) EvalASTHostInterop(n ast.Node) (interface{}, error) {
	tgt := get(n, KWTarget)
	mOrF := get(n, KWMOrF).(*value.Symbol)

	tgtVal, err := env.EvalAST(tgt.(ast.Node))
	if err != nil {
		return nil, err
	}

	mOrFVal, ok := value.FieldOrMethod(tgtVal, mOrF.Name())
	if !ok {
		return nil, fmt.Errorf("no such field or method on %T: %s", tgtVal, mOrF)
	}
	if mOrFVal == nil {
		// Avoid panic in kind check below and just return if nil. It
		// can't have been a method.
		return mOrFVal, nil
	}
	switch reflect.TypeOf(mOrFVal).Kind() {
	case reflect.Func:
		return value.Apply(mOrFVal, nil)
	default:
		return mOrFVal, nil
	}
}

func (env *environment) EvalASTWithMeta(n ast.Node) (interface{}, error) {
	expr := get(n, KWExpr)
	meta := get(n, KWMeta).(value.IPersistentMap)
	exprVal, err := env.EvalAST(expr.(ast.Node))
	if err != nil {
		return nil, err
	}
	metaVal, err := env.EvalAST(meta)
	if err != nil {
		return nil, err
	}

	return value.WithMeta(exprVal, metaVal.(value.IPersistentMap))
}

func (env *environment) EvalASTFn(n ast.Node) (interface{}, error) {
	return value.NewFn(n, env), nil
}

func (env *environment) EvalASTMap(n ast.Node) (interface{}, error) {
	res := value.NewMap()

	keys := get(n, KWKeys)
	vals := get(n, KWVals)
	for i := 0; i < value.Count(keys); i++ {
		key := value.Get(keys, i)
		keyVal, err := env.EvalAST(key.(ast.Node))
		if err != nil {
			return nil, err
		}
		val := value.Get(vals, i)
		valVal, err := env.EvalAST(val.(ast.Node))
		if err != nil {
			return nil, err
		}
		res = value.Assoc(res, keyVal, valVal).(value.IPersistentMap)
	}

	return res, nil
}

func (env *environment) EvalASTVector(n ast.Node) (interface{}, error) {
	items := get(n, KWItems)
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
	items := get(n, KWItems)
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
	test := get(n, KWTest)
	then := get(n, KWThen)
	els := get(n, KWElse)

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

func (env *environment) EvalASTCase(n ast.Node) (interface{}, error) {
	testVal, err := env.EvalAST(get(n, KWTest))
	if err != nil {
		return nil, err
	}
	for seq := value.Seq(get(n, KWNodes)); seq != nil; seq = value.Next(seq) {
		node := value.First(seq)
		for testSeq := value.Seq(get(node, KWTests)); testSeq != nil; testSeq = value.Next(testSeq) {
			caseTestVal, err := env.EvalAST(value.First(testSeq).(ast.Node))
			if err != nil {
				return nil, err
			}
			if value.Equal(testVal, caseTestVal) {
				res, err := env.EvalAST(get(node, KWThen))
				if err != nil {
					return nil, err
				}
				return res, nil
			}
		}
	}
	return env.EvalAST(get(n, KWDefault))
}

func (env *environment) EvalASTDo(n ast.Node) (interface{}, error) {
	statements := get(n, KWStatements)
	for i := 0; i < value.Count(statements); i++ {
		_, err := env.EvalAST(value.Get(statements, i).(ast.Node))
		if err != nil {
			return nil, err
		}
	}
	ret := get(n, KWRet)
	return env.EvalAST(ret.(ast.Node))
}

func (env *environment) EvalASTLet(n ast.Node, isLoop bool) (interface{}, error) {
	newEnv := env.PushScope().(*environment)

	var bindNameVals []interface{}

	bindings := get(n, KWBindings)
	for i := 0; i < value.Count(bindings); i++ {
		binding := get(bindings, i)
		name := get(binding, KWName)
		init := get(binding, KWInit)
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

	res, err := recurEnv.EvalAST(get(n, KWBody).(ast.Node))
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

	exprs := get(n, KWExprs)
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

func (env *environment) EvalASTInvoke(n ast.Node) (res interface{}, err error) {
	defer func() {
		meta, ok := get(n, KWMeta).(value.IPersistentMap)
		if !ok {
			return
		}
		var gljFrame string
		if r := recover(); r != nil {
			// TODO: dynamically set pr-on to nil to avoid infinite
			// recursion; need to use go-only stringification for errors.
			gljFrame = fmt.Sprintf("%s:%d:%d: %s\n", value.Get(meta, KWFile), value.Get(meta, KWLine), value.Get(meta, KWColumn), get(n, KWForm))
			switch r := r.(type) {
			case *EvalError:
				r.GLJStack = append(r.GLJStack, gljFrame)
				if r.GoStack == "" {
					r.GoStack = string(debug.Stack())
				}
				err = r
			case error:
				err = &EvalError{
					Err:      r,
					GLJStack: []string{gljFrame},
					GoStack:  string(debug.Stack()),
				}
			default:
				err = &EvalError{
					Err:      fmt.Errorf("%v", r),
					GLJStack: []string{gljFrame},
					GoStack:  string(debug.Stack()),
				}
			}
		}
	}()
	fn := get(n, KWFn)
	args := get(n, KWArgs)
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

	return value.Apply(fnVal, argVals)
}

func (env *environment) EvalASTVar(n ast.Node) (interface{}, error) {
	return get(n, KWVar).(*value.Var).Get(), nil
}

func (env *environment) EvalASTLocal(n ast.Node) (interface{}, error) {
	sym := get(n, KWName).(*value.Symbol)
	v, ok := env.lookup(sym)
	if !ok {
		return nil, env.errorf(get(n, KWForm), "unable to resolve local symbol: %s", sym)
	}
	return v, nil
}

func (env *environment) EvalASTNew(n ast.Node) (interface{}, error) {
	classVal, err := env.EvalAST(get(n, KWClass))
	if err != nil {
		return nil, err
	}
	if value.Count(get(n, KWArgs)) > 0 {
		return nil, errors.New("new with args unsupported")
	}
	classValTyp, ok := classVal.(reflect.Type)
	if !ok {
		return nil, fmt.Errorf("new value must be a reflect.Type, got %T", classVal)
	}
	return reflect.New(classValTyp).Interface(), nil
}

func (env *environment) EvalASTTry(n ast.Node) (res interface{}, err error) {
	if finally := get(n, KWFinally); finally != nil {
		defer func() {
			_, ferr := env.EvalAST(finally.(ast.Node))
			if ferr != nil {
				err = ferr
			}
		}()
	}
	// TODO: catch
	return env.EvalAST(get(n, KWBody))
}

func (env *environment) EvalASTThrow(n ast.Node) (interface{}, error) {
	exception, err := env.EvalAST(get(n, KWException))
	if err != nil {
		return nil, err
	}
	panic(exception)
}

func get(x interface{}, key interface{}) interface{} {
	return value.Get(x, key)
}
