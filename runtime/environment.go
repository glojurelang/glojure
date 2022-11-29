package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/value"
)

type environment struct {
	ctx context.Context

	scope *scope

	// global vars scope
	vars *scope

	macros map[string]*value.Func

	gensymCounter int

	stdout io.Writer

	loadPath []string
}

func newEnvironment(ctx context.Context, stdout io.Writer) *environment {
	e := &environment{
		ctx:    ctx,
		scope:  newScope(),
		vars:   newScope(),
		macros: make(map[string]*value.Func),
		stdout: stdout,
	}
	addBuiltins(e)
	return e
}

func (env *environment) Context() context.Context {
	return env.ctx
}

func (env *environment) String() string {
	return fmt.Sprintf("environment:\nScope:\n%v", env.scope.printIndented("  "))
}

func (env *environment) Define(name string, value interface{}) {
	// TODO: define should define globally!
	env.scope.define(name, value)
}

func (env *environment) DefVar(name string, value interface{}) {
	env.vars.define(name, value)
}

func (env *environment) DefineMacro(name string, fn *value.Func) {
	env.macros[name] = fn
}

func (env *environment) lookup(name string) (interface{}, bool) {
	v, ok := env.scope.lookup(name)
	if ok {
		return v, true
	}
	return env.vars.lookup(name)
}

func (env *environment) PushScope() value.Environment {
	wrappedEnv := *env
	newEnv := &wrappedEnv
	newEnv.scope = newEnv.scope.push()
	return newEnv
}

func (env *environment) Stdout() io.Writer {
	return env.stdout
}

func (env *environment) PushLoadPaths(paths []string) value.Environment {
	newEnv := &(*env)
	newEnv.loadPath = append(paths, newEnv.loadPath...)
	return newEnv
}

func (env *environment) ResolveFile(filename string) (string, bool) {
	if filepath.IsAbs(filename) {
		return filename, true
	}

	for _, path := range env.loadPath {
		fullPath := filepath.Join(path, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, true
		}
	}
	return "", false
}

type poser interface {
	Pos() value.Pos
}

func (env *environment) errorf(n interface{}, format string, args ...interface{}) error {
	var pos value.Pos
	if n, ok := n.(poser); ok {
		pos = n.Pos()
	}
	filename := "?"
	line := "?"
	col := "?"
	if pos.Valid() {
		if pos.Filename != "" {
			filename = pos.Filename
		}
		line = fmt.Sprintf("%d", pos.Line)
		col = fmt.Sprintf("%d", pos.Column)
	}
	location := fmt.Sprintf("%s:%s:%s", filename, line, col)

	return fmt.Errorf("%s: %s", location, fmt.Sprintf(format, args...))
}

func (env *environment) Eval(n interface{}) (interface{}, error) {
	var result interface{}
	var err error
	continuation := func() (interface{}, value.Continuation, error) {
		return env.eval(n)
	}

	for {
		result, continuation, err = continuation()
		if err != nil {
			return nil, err
		}
		if continuation == nil {
			return result, nil
		}
	}
}

func (env *environment) ContinuationEval(n interface{}) (interface{}, value.Continuation, error) {
	return env.eval(n)
}

func (env *environment) eval(n interface{}) (interface{}, value.Continuation, error) {
	switch v := n.(type) {
	case *value.List:
		return env.evalList(v)
	case *value.Vector:
		res, err := env.evalVector(v)
		return res, nil, err
	default:
		res, err := env.evalScalar(n)
		return res, nil, err
	}
}

func asContinuationResult(v interface{}, err error) (interface{}, value.Continuation, error) {
	return v, nil, err
}

func (env *environment) evalList(n *value.List) (interface{}, value.Continuation, error) {
	if n.IsEmpty() {
		return nil, nil, nil
	}

	first := n.Item()
	if sym, ok := first.(*value.Symbol); ok {
		// handle special forms
		switch sym.Value {
		case "def":
			return asContinuationResult(env.evalDef(n))
		case "if":
			return env.evalIf(n)
		case "case":
			return env.evalCase(n)
		case "fn":
			return asContinuationResult(env.evalFn(n))
		case "quote":
			return asContinuationResult(env.evalQuote(n))
		case "quasiquote":
			return asContinuationResult(env.evalQuasiquote(n))
		case "let":
			return env.evalLet(n)
		case "defmacro":
			return asContinuationResult(env.evalDefMacro(n))

			// Go interop special forms
		case ".":
			return asContinuationResult(env.evalDot(n))
		case "new":
			return asContinuationResult(env.evalNew(n))
		case "set!":
			return asContinuationResult(env.evalSet(n))
		}

		// handle macros
		if macro, ok := env.macros[sym.Value]; ok {
			return env.applyMacro(macro, n.Next())
		}
	}

	// otherwise, handle a function call
	var res []interface{}
	for cur := n; !cur.IsEmpty(); cur = cur.Next() {
		item := cur.Item()
		v, err := env.Eval(item)
		if err != nil {
			return nil, nil, err
		}
		res = append(res, v)
	}

	return nil, func() (interface{}, value.Continuation, error) {
		return env.applyFunc(res[0], res[1:])
	}, nil
}

func (env *environment) evalVector(n *value.Vector) (interface{}, error) {
	var res []interface{}
	for i := 0; i < n.Count(); i++ {
		item := n.ValueAt(i)
		v, err := env.Eval(item)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return value.NewVector(res, value.WithSection(n.Section)), nil
}

func (env *environment) evalScalar(n interface{}) (interface{}, error) {
	switch v := n.(type) {
	case *value.Symbol:
		if val, ok := env.lookup(v.Value); ok {
			return val, nil
		}
		return nil, env.errorf(n, "undefined symbol: %s", v.Value)
	default:
		// else, it's a literal
		return v, nil
	}
}

func (env *environment) applyFunc(f interface{}, args []interface{}) (interface{}, value.Continuation, error) {
	cfn, ok := f.(value.ContinuationApplyer)
	if ok {
		return cfn.ContinuationApply(env, args)
	}

	fn, ok := f.(value.Applyer)
	if !ok {
		// TODO: the error's location should indicate the call site, not
		// the location at which the function value was defined.
		return nil, nil, env.errorf(f, "value is not a function: %v", f)
	}
	res, err := fn.Apply(env, args)
	return res, nil, err
}

// Special forms

type nopApplyer struct{}

func (na *nopApplyer) Apply(env *environment, args []interface{}) (interface{}, error) {
	return nil, nil
}

func (env *environment) evalDef(n *value.List) (interface{}, error) {
	listLength := n.Count()
	if listLength < 3 {
		return nil, env.errorf(n, "too few arguments to def")
	}
	if listLength > 4 {
		return nil, env.errorf(n, "too many arguments to def")
	}

	v, ok := value.MustNth(n, 1).(*value.Symbol)
	if !ok {
		return nil, env.errorf(n, "invalid def, first item is not a symbol")
	}
	valIndex := 2
	if listLength == 4 {
		_, ok := value.MustNth(n, 2).(string)
		if !ok {
			return nil, env.errorf(n, "too many arguments to def")
		}
		// TODO: store docstring
		valIndex = 3
	}

	val, err := env.Eval(value.MustNth(n, valIndex))
	if err != nil {
		return nil, err
	}
	env.DefVar(v.Value, val)
	return nil, nil
}

func (env *environment) evalFn(n *value.List) (interface{}, error) {
	listLength := n.Count()
	items := make([]interface{}, 0, listLength-1)
	for cur := n.Next(); !cur.IsEmpty(); cur = cur.Next() {
		items = append(items, cur.Item())
	}

	if len(items) < 2 {
		return nil, env.errorf(n, "invalid fn expression, need args and body")
	}

	var fnName string
	if sym, ok := items[0].(*value.Symbol); ok {
		// if the first child is not a list, it's the name of the
		// function. this can be used for recursion.
		fnName = sym.Value
		items = items[1:]
	}

	if len(items) == 0 {
		return nil, env.errorf(n, "invalid fn expression, need args and body")
	}

	const errorString = "invalid fn expression, expected (fn ([bindings0] body0) ([bindings1] body1) ...) or (fn [bindings] body)"

	arities := make([]*value.List, 0, len(items))
	if _, ok := items[0].(*value.Vector); ok {
		// if the next child is a vector, it's the bindings, and we only
		// have one arity.
		arities = append(arities, value.NewList(items, value.WithSection(n.Section)))
	} else {
		// otherwise, every remaining child must be a list of function
		// bindings and bodies for each arity.
		for _, item := range items {
			list, ok := item.(*value.List)
			if !ok {
				return nil, env.errorf(n, errorString)
			}
			arities = append(arities, list)
		}
	}

	arityValues := make([]value.FuncArity, len(arities))
	for i, arity := range arities {
		bindings, ok := arity.Item().(*value.Vector)
		if !ok {
			return nil, env.errorf(n, errorString)
		}
		if !value.IsValidBinding(bindings) {
			return nil, env.errorf(n, "invalid fn expression, invalid binding (%v). Must be valid destructure form", bindings)
		}

		body := arity.Next()

		arityValues[i] = value.FuncArity{
			BindingForm: bindings,
			Exprs:       body,
		}
	}
	return &value.Func{
		Section:    n.Section,
		LambdaName: fnName,
		Env:        env,
		Arities:    arityValues,
	}, nil
}

func (env *environment) evalIf(n *value.List) (interface{}, value.Continuation, error) {
	listLength := n.Count()
	if listLength < 3 || listLength > 4 {
		return nil, nil, env.errorf(n, "invalid if, need `cond ifExp [elseExp]`")
	}
	cond, err := env.Eval(n.Next().Item())
	if err != nil {
		return nil, nil, err
	}

	if value.IsTruthy(cond) {
		return nil, func() (interface{}, value.Continuation, error) {
			return env.eval(n.Next().Next().Item())
		}, nil
	}

	if listLength == 4 {
		return nil, func() (interface{}, value.Continuation, error) {
			return env.eval(n.Next().Next().Next().Item())
		}, nil
	}
	return nil, nil, nil
}

// cases use syntax and most of the semantics of Clojure's case (not Scheme's).
// see https://clojuredocs.org/clojure.core/case
func (env *environment) evalCase(n *value.List) (interface{}, value.Continuation, error) {
	listLength := n.Count()
	if listLength < 4 {
		return nil, nil, env.errorf(n, "invalid case, need `case caseExp & caseClauses`")
	}
	cond, err := env.Eval(n.Next().Item())
	if err != nil {
		return nil, nil, err
	}

	//cases := n.Items[2:]
	cases := make([]interface{}, 0, listLength-2)
	for cur := n.Next().Next(); !cur.IsEmpty(); cur = cur.Next() {
		cases = append(cases, cur.Item())
	}

	for len(cases) >= 2 {
		test, result := cases[0], cases[1]
		cases = cases[2:]

		testItems := []interface{}{test}
		testList, ok := test.(*value.List)
		if ok {
			testItems = make([]interface{}, 0, testList.Count())
			for cur := testList; !cur.IsEmpty(); cur = cur.Next() {
				testItems = append(testItems, cur.Item())
			}
		}

		for _, testItem := range testItems {
			if value.Equal(testItem, cond) {
				return nil, func() (interface{}, value.Continuation, error) {
					return env.eval(result)
				}, nil
			}
		}
	}
	if len(cases) == 1 {
		return nil, func() (interface{}, value.Continuation, error) {
			return env.eval(cases[0])
		}, nil
	}
	return nil, nil, nil
}

func toBool(v interface{}) bool {
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

func toBoolContinuation(v interface{}, c value.Continuation, err error) (interface{}, value.Continuation, error) {
	if err != nil {
		return nil, nil, err
	}
	if c == nil {
		return toBool(v), nil, nil
	}
	return nil, func() (interface{}, value.Continuation, error) {
		return toBoolContinuation(c())
	}, nil
}

func (env *environment) evalQuote(n *value.List) (interface{}, error) {
	listLength := n.Count()
	if listLength != 2 {
		return nil, env.errorf(n, "invalid quote, need 1 argument")
	}

	return n.Next().Item(), nil
}

func (env *environment) evalQuasiquote(n *value.List) (interface{}, error) {
	listLength := n.Count()
	if listLength != 2 {
		return nil, env.errorf(n, "invalid quasiquote, need 1 argument")
	}

	// symbolNameMap tracks the names of symbols that have been renamed.
	// symbols that end with a '#' have '#' replaced with a unique
	// suffix.
	symbolNameMap := make(map[string]string)
	return env.evalQuasiquoteItem(symbolNameMap, n.Next().Item())
}

func (env *environment) evalQuasiquoteItem(symbolNameMap map[string]string, item interface{}) (interface{}, error) {
	switch item := item.(type) {
	case *value.List:
		if item.IsEmpty() {
			return item, nil
		}
		if value.Equal(item.Item(), value.SymbolUnquote) {
			return env.Eval(item.Next().Item())
		}
		if value.Equal(item.Item(), value.SymbolSpliceUnquote) {
			return nil, env.errorf(item, "splice-unquote not in list")
		}

		var resultValues []interface{}
		for cur := item; !cur.IsEmpty(); cur = cur.Next() {
			if lst, ok := cur.Item().(*value.List); ok && !lst.IsEmpty() && value.Equal(lst.Item(), value.SymbolSpliceUnquote) {
				res, err := env.Eval(lst.Next().Item())
				if err != nil {
					return nil, err
				}
				vals, ok := res.(value.Nther)
				if !ok {
					return nil, env.errorf(lst, "splice-unquote did not return an enumerable")
				}
				for i := 0; ; i++ {
					v, ok := vals.Nth(i)
					if !ok {
						break
					}
					resultValues = append(resultValues, v)
				}
				continue
			}

			result, err := env.evalQuasiquoteItem(symbolNameMap, cur.Item())
			if err != nil {
				return nil, err
			}
			resultValues = append(resultValues, result)
		}
		return value.NewList(resultValues), nil
	case *value.Vector:
		if item.Count() == 0 {
			return item, nil
		}

		var resultValues []interface{}
		for i := 0; i < item.Count(); i++ {
			cur := item.ValueAt(i)
			if lst, ok := cur.(*value.List); ok && !lst.IsEmpty() && value.Equal(lst.Item(), value.SymbolSpliceUnquote) {
				res, err := env.Eval(lst.Next().Item())
				if err != nil {
					return nil, err
				}
				vals, ok := res.(value.Nther)
				if !ok {
					return nil, env.errorf(lst, "splice-unquote did not return an enumerable")
				}
				for j := 0; ; j++ {
					v, ok := vals.Nth(j)
					if !ok {
						break
					}
					resultValues = append(resultValues, v)
				}
				continue
			}

			result, err := env.evalQuasiquoteItem(symbolNameMap, cur)
			if err != nil {
				return nil, err
			}
			resultValues = append(resultValues, result)
		}
		return value.NewVector(resultValues), nil
	case *value.Symbol:
		if !strings.HasSuffix(item.Value, "#") {
			return item, nil
		}
		newName, ok := symbolNameMap[item.Value]
		if !ok {
			newName = item.Value[:len(item.Value)-1] + "__" + strconv.Itoa(env.gensymCounter) + "__auto__"
			symbolNameMap[item.Value] = newName
			env.gensymCounter++
		}
		return value.NewSymbol(newName), nil
	default:
		return item, nil
	}
}

// essential syntax: let <bindings> <body>
//
// Two forms are supported. The first is the standard let form:
//==============================================================================
// Syntax: <bindings> should have the form:
//
// ((<symbol 1> <init 1>) ...),
//
// where each <init> is an expression, and <body> should be a sequence
// of one or more expressions. It is an error for a <symbol> to
// appear more than once in the list of symbols being bound.
//
// Semantics: The <init>s are evaluated in the current environment (in
// some unspecified order), the <symbol>s are bound to fresh
// locations holding the results, the <body> is evaluated in the
// extended environment, and the value of the last expression of
// <body> is returned. Each binding of a <symbol> has <body> as its
// region.
//==============================================================================
// The second form is the clojure
// (https://clojuredocs.org/clojure.core/let) let form:
//
// Syntax: <bindings> should have the form:
//
// [<symbol 1> <init 1> ... <symbol n> <init n>]
//
// Semantics: Semantics are as above, except that the <init>s are
// evaluated in order, and all preceding <init>s are available to
// subsequent <init>s.
func (env *environment) evalLet(n *value.List) (interface{}, value.Continuation, error) {
	items := listAsSlice(n)
	if len(items) < 3 {
		return nil, nil, env.errorf(n, "invalid let, need bindings and body")
	}

	var bindingsMap map[string]interface{}
	var err error
	switch bindings := items[1].(type) {
	case *value.Vector:
		bindingsMap, err = env.evalVectorBindings(bindings)
	default:
		return nil, nil, env.errorf(n, "invalid let, bindings must be a list or vector")
	}
	if err != nil {
		return nil, nil, err
	}

	// create a new environment with the bindings
	newEnv := env.PushScope().(*environment)
	for name, val := range bindingsMap {
		newEnv.Define(name, val)
	}

	// evaluate the body
	for _, item := range items[2 : len(items)-1] {
		_, err = newEnv.Eval(item)
		if err != nil {
			return nil, nil, err
		}
	}
	// return a continuation for the last item
	return nil, func() (interface{}, value.Continuation, error) {
		return newEnv.eval(items[len(items)-1])
	}, nil
}

func (env *environment) evalVectorBindings(bindings *value.Vector) (map[string]interface{}, error) {
	if bindings.Count()%2 != 0 {
		return nil, env.errorf(bindings, "invalid let, bindings must be a vector of even length")
	}

	newEnv := env.PushScope().(*environment)
	bindingsMap := make(map[string]interface{})
	for i := 0; i < bindings.Count(); i += 2 {
		nameValue := bindings.ValueAt(i)
		name, ok := nameValue.(*value.Symbol)
		if !ok {
			return nil, env.errorf(bindings, "invalid let, binding name must be a symbol")
		}
		if _, ok := bindingsMap[name.Value]; ok {
			return nil, env.errorf(nameValue, "invalid let, duplicate binding name")
		}
		val, err := newEnv.Eval(bindings.ValueAt(i + 1))
		if err != nil {
			return nil, err
		}
		bindingsMap[name.Value] = val
		newEnv.Define(name.Value, val)
	}

	return bindingsMap, nil
}

func (env *environment) evalDefMacro(n *value.List) (interface{}, error) {
	// fnList is a transformed version of the macro that looks like a fn
	// form, which is nearly the same as the defmacro form but without
	// the docstring or metadata.
	fnList := n
	if n.Count() > 3 {
		_, ok := value.MustNth(n, 2).(string)
		if ok {
			fnList = n.Next().Next().Next().Conj(value.MustNth(n, 1), value.MustNth(n, 0)).(*value.List)
			// TODO: store the docstring somewhere
		}
	}

	fn, err := env.evalFn(fnList)
	if err != nil {
		return nil, err
	}

	sym, ok := value.MustNth(n, 1).(*value.Symbol)
	if !ok {
		return nil, env.errorf(n.Next().Item(), "invalid defmacro, name must be a symbol")
	}

	env.DefineMacro(sym.Value, fn.(*value.Func))
	return nil, nil
}

func (env *environment) applyMacro(fn *value.Func, argList *value.List) (interface{}, value.Continuation, error) {
	args := listAsSlice(argList)
	res, c, err := env.applyFunc(fn, args)
	if err != nil {
		return nil, nil, err
	}
	if c == nil {
		return nil, func() (interface{}, value.Continuation, error) {
			return env.eval(res)
		}, nil
	}

	// continue evaluating until we get a result
	for c != nil && err == nil {
		res, c, err = c()
	}
	if err != nil {
		return nil, nil, err
	}

	return nil, func() (interface{}, value.Continuation, error) {
		return env.eval(res)
	}, nil
}

func (env *environment) evalNew(n *value.List) (interface{}, error) {
	argCount := n.Count() - 1
	if argCount < 1 {
		return nil, env.errorf(n, "invalid expression, expected (new <type> <field_value>*)")
	}
	if (argCount-1)%2 != 0 {
		return nil, env.errorf(n, "invalid expression, expected (new <type> <field> <value> ...)")
	}

	typeValIfc, err := env.Eval(value.MustNth(n, 1))
	if err != nil {
		return nil, err
	}
	typeValue, ok := typeValIfc.(*value.GoTyp)
	if !ok {
		return nil, env.errorf(value.MustNth(n, 1), "invalid expression, expected (new <type> <field_value>*)")
	}

	val := typeValue.New()
	for cur := n.Next().Next(); !cur.IsEmpty(); cur = cur.Next().Next() {
		fieldName, ok := cur.Item().(*value.Keyword)
		if !ok {
			return nil, env.errorf(cur.Item(), "invalid new expression, field name must be a keyword")
		}
		fieldValue, err := env.Eval(cur.Next().Item())
		if err != nil {
			return nil, err
		}
		field := val.Elem().FieldByName(fieldName.Value)
		if !field.IsValid() {
			return nil, env.errorf(cur.Item(), "invalid new expression, unknown field (%v)", fieldName.Value)
		}
		if !field.CanSet() {
			return nil, env.errorf(cur.Item(), "invalid new expression, field is not settable (%v)", fieldName.Value)
		}
		goVal := fieldValue
		if fieldValue, ok := fieldValue.(value.GoValuer); ok {
			goVal = fieldValue.GoValue()
		}

		reflectVal := reflect.ValueOf(goVal)
		if goVal == nil && isNilableKind(field.Kind()) {
			// reflect.Value.Set panics if the value is an untyped nil
			reflectVal = reflect.Zero(field.Type())
		}

		field.Set(reflectVal)
	}

	return value.NewGoVal(val.Interface()), nil
}

func (env *environment) evalSet(n *value.List) (interface{}, error) {
	argCount := n.Count() - 1
	if argCount != 2 {
		return nil, env.errorf(n, "invalid expression, expected (set! (. go-value-expr field-symbol) expr)")
	}
	// evaluate the go-value-expr, which should be a *value.GoVal.
	// check that the field-symbol is a symbol
	// evaluate the expression

	fieldExpr, ok := value.MustNth(n, 1).(*value.List)
	{ // Validation
		if !ok {
			return nil, env.errorf(value.MustNth(n, 1), "invalid expression, expected (set! (. go-value-expr field-symbol) expr)")
		}
		if fieldExpr.Count() != 3 {
			return nil, env.errorf(fieldExpr, "invalid expression, expected (set! (. go-value-expr field-symbol) expr)")
		}
		if dotSym, ok := value.MustNth(fieldExpr, 0).(*value.Symbol); !ok || dotSym.Value != "." {
			return nil, env.errorf(fieldExpr, "invalid expression, expected (set! (. go-value-expr field-symbol) expr)")
		}
	}

	fieldSym, ok := value.MustNth(fieldExpr, 2).(*value.Symbol)
	if !ok {
		return nil, env.errorf(value.MustNth(fieldExpr, 2), "invalid set! expression, expected a symbol")
	}
	targetExpr, err := env.Eval(value.MustNth(fieldExpr, 1))
	if err != nil {
		return nil, err
	}
	targetGoVal, ok := targetExpr.(*value.GoVal)
	if !ok {
		return nil, env.errorf(value.MustNth(fieldExpr, 1), "invalid set! expression, expected a go value")
	}

	expr, err := env.Eval(value.MustNth(n, 2))
	if err != nil {
		return nil, err
	}

	var goVal interface{}

	if goValuer, ok := expr.(value.GoValuer); ok {
		goVal = goValuer.GoValue()
	} else {
		goVal = expr
	}

	if err := targetGoVal.SetField(fieldSym.Value, goVal); err != nil {
		return nil, env.errorf(n, "invalid set! expression, %v", err)
	}

	return expr, nil
}

func (env *environment) evalDot(n *value.List) (interface{}, error) {
	// the dot form is a port of clojure's dot special form, giving
	// access to Go host functions and values.
	//
	// as described in https://clojure.org/reference/java_interop#dot,
	// the following variations are supported in Java:
	//
	// 1. (. instance-expr member-symbol)
	// 2. (. Classname-symbol member-symbol)
	// 3. (. instance-expr -field-symbol)
	// 4. (. instance-expr (method-symbol args*)) or (. instance-expr method-symbol args*)
	// 5. (. Classname-symbol (method-symbol args*)) or (. Classname-symbol method-symbol args*)
	//
	// TODO: form 3

	dotCount := n.Count()

	if dotCount < 3 {
		return nil, env.errorf(n, "invalid expression, expecting (. target member ...)")
	}

	target, err := env.Eval(n.Next().Item())
	if err != nil {
		return nil, err
	}

	// TODO: how to handle glojure types when treating them as dot
	// targets? Clojure just exposes them as the underlying Java
	// classes, e.g. clojure.lang.PersistentVector. if we take that
	// route, we need to be confident in our representation of
	// primitives.

	var goValTarget *value.GoVal
	if gv, ok := target.(*value.GoVal); ok {
		goValTarget = gv
	} else if gv, ok := target.(value.GoValuer); ok {
		goValTarget = value.NewGoVal(gv.GoValue())
	} else {
		goValTarget = value.NewGoVal(target)
	}

	memberExpr := n.Next().Next().Item()
	if dotCount > 3 {
		// must be a convenience form for a method call, e.g. (. foo bar baz)
		// use the tail as the member expression.
		memberExpr = n.Next().Next()
	}

	if v, ok := memberExpr.(*value.Symbol); ok {
		field := goValTarget.FieldOrMethod(v.Value)
		if field == nil {
			return nil, env.errorf(v, "unknown field or method (%v)", v.Value)
		}

		fieldVal := field.Value()
		reflectVal := reflect.ValueOf(fieldVal)

		// if the field is not a function, or it has an arity greater than
		// zero, return the field value.
		if reflectVal.Type().Kind() != reflect.Func || reflectVal.Type().NumIn() > 0 {
			return fieldVal, nil
		}

		// otherwise, the field is a function with no arguments, so we
		// drop down to the function call case below. This is a variant of
		// form 2 above, where the field is a function with no arguments.
		memberExpr = n.Next().Next()
	}

	if v, ok := memberExpr.(*value.List); ok {
		sym, ok := v.Item().(*value.Symbol)
		if !ok {
			return nil, env.errorf(v.Item(), "invalid expression, method name must be a symbol")
		}

		method := goValTarget.FieldOrMethod(sym.Value)
		args := make([]interface{}, v.Next().Count())
		for i := range args {
			v, err := env.Eval(value.MustNth(v, i+1))
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return method.Apply(env, args)
	}

	return nil, fmt.Errorf("unimplemented")
}

// Helpers

func nodeAsStringList(n *value.List) ([]string, error) {
	var res []string
	for cur := n; !cur.IsEmpty(); cur = cur.Next() {
		item := cur.Item()
		sym, ok := item.(*value.Symbol)
		if !ok {
			return nil, fmt.Errorf("invalid argument list: %v", n)
		}
		res = append(res, sym.Value)
	}
	return res, nil
}

// TODO: don't use this for function application. avoid copying and
// just take lists or sequences.
func listAsSlice(lst *value.List) []interface{} {
	var res []interface{}
	for cur := lst; !cur.IsEmpty(); cur = cur.Next() {
		res = append(res, cur.Item())
	}
	return res
}

func isNilableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}
