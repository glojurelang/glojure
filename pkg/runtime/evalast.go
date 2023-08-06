package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
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
	n := x.(*ast.Node)
	switch n.Op {
	case ast.OpConst:
		return n.Sub.(*ast.ConstNode).Value, nil
	case ast.OpDef:
		return env.EvalASTDef(n)
	case ast.OpSetBang:
		return env.EvalASTAssign(n)
	case ast.OpMaybeClass:
		return env.EvalASTMaybeClass(n)
	case ast.OpWithMeta:
		return env.EvalASTWithMeta(n)
	case ast.OpFn:
		return env.EvalASTFn(n)
	case ast.OpMap:
		return env.EvalASTMap(n)
	case ast.OpVector:
		return env.EvalASTVector(n)
	case ast.OpSet:
		return env.EvalASTSet(n)
	case ast.OpDo:
		return env.EvalASTDo(n)
	case ast.OpLet:
		return env.EvalASTLet(n, false)
	case ast.OpLoop:
		return env.EvalASTLet(n, true)
	case ast.OpInvoke:
		return env.EvalASTInvoke(n)
	case ast.OpQuote:
		return n.Sub.(*ast.QuoteNode).Expr.Sub.(*ast.ConstNode).Value, nil
	case ast.OpVar:
		return env.EvalASTVar(n)
	case ast.OpLocal:
		return env.EvalASTLocal(n)
	case ast.OpGoBuiltin:
		return n.Sub.(*ast.GoBuiltinNode).Value, nil
	case ast.OpHostCall:
		return env.EvalASTHostCall(n)
	case ast.OpHostInterop:
		return env.EvalASTHostInterop(n)
	case ast.OpMaybeHostForm:
		return env.EvalASTMaybeHostForm(n)
	case ast.OpIf:
		return env.EvalASTIf(n)
	case ast.OpCase:
		return env.EvalASTCase(n)
	case ast.OpTheVar:
		return env.EvalASTTheVar(n)
	case ast.OpRecur:
		return env.EvalASTRecur(n)
	case ast.OpNew:
		return env.EvalASTNew(n)
	case ast.OpTry:
		return env.EvalASTTry(n)
	case ast.OpThrow:
		return env.EvalASTThrow(n)
	default:
		panic(fmt.Errorf("unimplemented op: %d. Form: %s", n.Op, value.ToString(n.Form)))
	}
}

func (env *environment) EvalASTDef(n *ast.Node) (interface{}, error) {
	defNode := n.Sub.(*ast.DefNode)
	init := defNode.Init
	if value.IsNil(init) {
		return defNode.Var, nil
	}

	initVal, err := env.EvalAST(init)
	if err != nil {
		return nil, err
	}
	sym := defNode.Name

	// evaluate symbol metadata if present
	meta := defNode.Meta
	if !value.IsNil(meta) {
		metaVal, err := env.EvalAST(meta)
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
	if RT.BooleanCast(lang.Get(vr.Meta(), value.KWDynamic)) {
		vr.SetDynamic()
	}
	return vr, nil
}

func (env *environment) EvalASTAssign(n *ast.Node) (interface{}, error) {
	setBangNode := n.Sub.(*ast.SetBangNode)

	val, err := env.EvalAST(setBangNode.Val)
	if err != nil {
		return nil, err
	}
	target := setBangNode.Target
	switch target.Op {
	case ast.OpVar:
		tgtVar := target.Sub.(*ast.VarNode).Var
		return tgtVar.Set(val), nil
	case ast.OpHostInterop:
		interopNode := target.Sub.(*ast.HostInteropNode)
		tgt := interopNode.Target
		interopTargetVal, err := env.EvalAST(tgt)
		if err != nil {
			return nil, err
		}
		field := interopNode.MOrF

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
		return nil, fmt.Errorf("unsupported assign target: %v", target.Form)
	}
}

func (env *environment) EvalASTTheVar(n *ast.Node) (interface{}, error) {
	return n.Sub.(*ast.TheVarNode).Var, nil
}

// TODO: this is a bit of a mess
type evalCompiler struct{}

var (
	Compiler = &evalCompiler{}
)

func (c *evalCompiler) Eval(form interface{}) interface{} {
	res, err := lang.GlobalEnv.Eval(form)
	if err != nil {
		panic(err)
	}
	return res
}

func (c *evalCompiler) Macroexpand1(form interface{}) interface{} {
	res, err := lang.GlobalEnv.(*environment).Macroexpand1(form)
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

func (env *environment) EvalASTMaybeClass(n *ast.Node) (interface{}, error) {
	sym := n.Sub.(*ast.MaybeClassNode).Class.(*value.Symbol)
	v, ok := pkgmap.Get(sym.FullName())
	if ok {
		return v, nil
	}

	return nil, errors.New("unable to resolve symbol: " + value.ToString(sym))
}

func (env *environment) EvalASTMaybeHostForm(n *ast.Node) (interface{}, error) {
	hostFormNode := n.Sub.(*ast.MaybeHostFormNode)
	field := hostFormNode.Field
	// TODO: implement this for real
	switch hostFormNode.Class {
	case "glojure.lang.PersistentTreeSet":
		switch field.Name() {
		case "create":
			return func(keys interface{}) interface{} {
				var ks []interface{}
				for seq := value.Seq(keys); seq != nil; seq = seq.Next() {
					ks = append(ks, seq.First())
				}
				return value.NewSet(ks...)
			}, nil
		}
	}

	// TODO: how to handle?
	panic("EvalASTMaybeHostForm: " + hostFormNode.Class + "/" + field.Name())
}

func (env *environment) EvalASTHostCall(n *ast.Node) (interface{}, error) {
	hostCallNode := n.Sub.(*ast.HostCallNode)

	tgt := hostCallNode.Target
	method := hostCallNode.Method
	args := hostCallNode.Args

	tgtVal, err := env.EvalAST(tgt)
	if err != nil {
		return nil, err
	}
	var argVals []interface{}
	for _, arg := range args {
		argVal, err := env.EvalAST(arg)
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

	return value.Apply(methodVal, argVals), nil
}

func (env *environment) EvalASTHostInterop(n *ast.Node) (interface{}, error) {
	hostInteropNode := n.Sub.(*ast.HostInteropNode)

	tgt := hostInteropNode.Target
	mOrF := hostInteropNode.MOrF

	tgtVal, err := env.EvalAST(tgt)
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
		return value.Apply(mOrFVal, nil), nil
	default:
		return mOrFVal, nil
	}
}

func (env *environment) EvalASTWithMeta(n *ast.Node) (interface{}, error) {
	wmNode := n.Sub.(*ast.WithMetaNode)

	expr := wmNode.Expr
	meta := wmNode.Meta
	exprVal, err := env.EvalAST(expr)
	if err != nil {
		return nil, err
	}
	metaVal, err := env.EvalAST(meta)
	if err != nil {
		return nil, err
	}

	return value.WithMeta(exprVal, metaVal.(value.IPersistentMap))
}

func (env *environment) EvalASTFn(n *ast.Node) (interface{}, error) {
	return NewFn(n, env), nil
}

func (env *environment) EvalASTMap(n *ast.Node) (interface{}, error) {
	res := value.NewMap()

	mapNode := n.Sub.(*ast.MapNode)

	keys := mapNode.Keys
	vals := mapNode.Vals
	for i, key := range keys {
		keyVal, err := env.EvalAST(key)
		if err != nil {
			return nil, err
		}
		val := vals[i]
		valVal, err := env.EvalAST(val)
		if err != nil {
			return nil, err
		}
		res = lang.Assoc(res, keyVal, valVal).(lang.IPersistentMap)
	}

	return res, nil
}

func (env *environment) EvalASTVector(n *ast.Node) (interface{}, error) {
	vectorNode := n.Sub.(*ast.VectorNode)

	items := vectorNode.Items

	var vals []interface{}
	for _, item := range items {
		itemVal, err := env.EvalAST(item)
		if err != nil {
			return nil, err
		}
		vals = append(vals, itemVal)
	}
	return value.NewVector(vals...), nil
}

func (env *environment) EvalASTSet(n *ast.Node) (interface{}, error) {
	setNode := n.Sub.(*ast.SetNode)

	items := setNode.Items

	var vals []interface{}
	for _, item := range items {
		itemVal, err := env.EvalAST(item)
		if err != nil {
			return nil, err
		}
		vals = append(vals, itemVal)
	}
	return value.NewSet(vals...), nil
}

func (env *environment) EvalASTIf(n *ast.Node) (interface{}, error) {
	ifNode := n.Sub.(*ast.IfNode)

	test := ifNode.Test
	then := ifNode.Then
	els := ifNode.Else

	testVal, err := env.EvalAST(test)
	if err != nil {
		return nil, err
	}
	if value.IsTruthy(testVal) {
		return env.EvalAST(then)
	} else {
		return env.EvalAST(els)
	}
}

func (env *environment) EvalASTCase(n *ast.Node) (interface{}, error) {
	caseNode := n.Sub.(*ast.CaseNode)

	testVal, err := env.EvalAST(caseNode.Test)
	if err != nil {
		return nil, err
	}

	for _, node := range caseNode.Nodes {
		caseNodeNode := node.Sub.(*ast.CaseNodeNode)
		tests := caseNodeNode.Tests
		for _, test := range tests {
			caseTestVal, err := env.EvalAST(test)
			if err != nil {
				return nil, err
			}
			if value.Equal(testVal, caseTestVal) {
				res, err := env.EvalAST(caseNodeNode.Then)
				if err != nil {
					return nil, err
				}
				return res, nil
			}
		}
	}
	return env.EvalAST(caseNode.Default)
}

func (env *environment) EvalASTDo(n *ast.Node) (interface{}, error) {
	doNode := n.Sub.(*ast.DoNode)

	statements := doNode.Statements
	for _, statement := range statements {
		_, err := env.EvalAST(statement)
		if err != nil {
			return nil, err
		}
	}
	ret := doNode.Ret
	return env.EvalAST(ret)
}

func (env *environment) EvalASTLet(n *ast.Node, isLoop bool) (interface{}, error) {
	letNode := n.Sub.(*ast.LetNode)

	newEnv := env.PushScope().(*environment)

	var bindNameVals []interface{}

	bindings := letNode.Bindings
	for _, binding := range bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)

		name := bindingNode.Name
		init := bindingNode.Init
		initVal, err := newEnv.EvalAST(init)
		if err != nil {
			return nil, err
		}
		// TODO: this should not mutate in-place!
		newEnv.BindLocal(name, initVal)

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

	res, err := recurEnv.EvalAST(letNode.Body)
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

func (env *environment) EvalASTRecur(n *ast.Node) (interface{}, error) {
	if env.recurTarget == nil {
		panic("recur outside of loop")
	}

	recurNode := n.Sub.(*ast.RecurNode)

	exprs := recurNode.Exprs
	vals := make([]interface{}, 0, value.Count(exprs))
	noRecurEnv := env.WithRecurTarget(nil).(*environment)
	for _, expr := range exprs {
		val, err := noRecurEnv.EvalAST(expr)
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

func (env *environment) EvalASTInvoke(n *ast.Node) (res interface{}, err error) {
	invokeNode := n.Sub.(*ast.InvokeNode)
	defer func() {
		meta := invokeNode.Meta
		var gljFrame string
		if r := recover(); r != nil {
			// TODO: dynamically set pr-on to nil to avoid infinite
			// recursion; need to use go-only stringification for errors.
			gljFrame = fmt.Sprintf("%s:%d:%d: %s\n", value.Get(meta, KWFile), value.Get(meta, KWLine), value.Get(meta, KWColumn), n.Form)
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

	fn := invokeNode.Fn
	args := invokeNode.Args
	fnVal, err := env.EvalAST(fn)
	if err != nil {
		return nil, err
	}

	var argVals []interface{}
	for _, arg := range args {
		argVal, err := env.EvalAST(arg)
		if err != nil {
			return nil, err
		}
		argVals = append(argVals, argVal)
	}

	return value.Apply(fnVal, argVals), nil
}

func (env *environment) EvalASTVar(n *ast.Node) (interface{}, error) {
	return n.Sub.(*ast.VarNode).Var.Get(), nil
}

func (env *environment) EvalASTLocal(n *ast.Node) (interface{}, error) {
	localNode := n.Sub.(*ast.LocalNode)

	sym := localNode.Name
	v, ok := env.lookup(sym)
	if !ok {
		return nil, env.errorf(n.Form, "unable to resolve local symbol: %s", sym)
	}
	return v, nil
}

func (env *environment) EvalASTNew(n *ast.Node) (interface{}, error) {
	newNode := n.Sub.(*ast.NewNode)

	classVal, err := env.EvalAST(newNode.Class)
	if err != nil {
		return nil, err
	}
	if len(newNode.Args) > 0 {
		return nil, errors.New("new with args unsupported")
	}
	classValTyp, ok := classVal.(reflect.Type)
	if !ok {
		return nil, fmt.Errorf("new value must be a reflect.Type, got %T", classVal)
	}
	return reflect.New(classValTyp).Interface(), nil
}

func (env *environment) EvalASTTry(n *ast.Node) (res interface{}, err error) {
	tryNode := n.Sub.(*ast.TryNode)

	if finally := tryNode.Finally; finally != nil {
		defer func() {
			_, ferr := env.EvalAST(finally)
			if ferr != nil {
				err = ferr
			}
		}()
	}
	// TODO: catch
	return env.EvalAST(tryNode.Body)
}

func (env *environment) EvalASTThrow(n *ast.Node) (interface{}, error) {
	throwNode := n.Sub.(*ast.ThrowNode)

	exception, err := env.EvalAST(throwNode.Exception)
	if err != nil {
		return nil, err
	}
	panic(exception)
}
