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
	"sync"

	"github.com/glojurelang/glojure/value"
)

var (
	SymbolUnquote       = value.NewSymbol("clojure.core/unquote") // TODO: rename to glojure.core/unquote
	SymbolSpliceUnquote = value.NewSymbol("splice-unquote")
	SymbolNamespace     = value.NewSymbol("ns")
	SymbolInNamespace   = value.NewSymbol("in-ns")
	SymbolUserNamespace = value.NewSymbol("user")
	SymbolDot           = value.NewSymbol(".")
)

type environment struct {
	ctx context.Context

	scope *scope

	currentNamespaceVar *value.Var
	namespaces          map[string]*value.Namespace
	nsMtx               *sync.RWMutex

	// some well-known vars
	namespaceVar   *value.Var // ns
	inNamespaceVar *value.Var // in-ns

	// counter for gensym (symbol generator)
	gensymCounter int

	stdout io.Writer
	stderr io.Writer

	loadPath []string
}

func newEnvironment(ctx context.Context, stdout, stderr io.Writer) *environment {
	e := &environment{
		ctx:        ctx,
		scope:      newScope(),
		namespaces: make(map[string]*value.Namespace),
		nsMtx:      &sync.RWMutex{},
		stdout:     stdout,
		stderr:     stderr,
	}
	coreNS := e.FindOrCreateNamespace(value.SymbolCoreNamespace)
	e.currentNamespaceVar = value.NewVarWithRoot(coreNS, value.NewSymbol("*ns*"), coreNS)

	// bootstrap some vars
	e.namespaceVar = value.NewVarWithRoot(coreNS, SymbolNamespace,
		value.ApplyerFunc(func(env value.Environment, args []interface{}) (interface{}, error) { return nil, nil }))
	e.namespaceVar.SetMacro()

	e.inNamespaceVar = value.NewVarWithRoot(coreNS, SymbolInNamespace, false)

	addBuiltins(e) // TODO: remove this
	return e
}

func (env *environment) Context() context.Context {
	return env.ctx
}

func (env *environment) String() string {
	return fmt.Sprintf("environment:\nScope:\n%v", env.scope.printIndented("  "))
}

func (env *environment) Define(sym *value.Symbol, val interface{}) {
	// TODO: define should define globally!
	env.scope.define(sym, val)
}

func (env *environment) DefVar(sym *value.Symbol, val interface{}) *value.Var {
	// TODO: match clojure implementation more closely
	v := env.CurrentNamespace().InternWithValue(env, sym, val, true /* replace root */)
	if meta := sym.Meta(); meta != nil {
		v.SetMeta(meta)
	}
	return v
}

func (env *environment) DefineMacro(name string, fn value.Applyer) {
	vr := env.DefVar(value.NewSymbol(name), fn)
	vr.SetMacro()
	// // TODO: should simply set macro flag on var
	// env.macros[name] = fn
}

func (env *environment) lookup(sym *value.Symbol) (interface{}, bool) {
	v, ok := env.scope.lookup(sym)
	if ok {
		return v, true
	}

	{ // HACKHACK
		// TODO: implement *ns* as a normal var
		if sym.String() == "*ns*" {
			return env.CurrentNamespace(), true
		}
	}

	ns := env.CurrentNamespace()
	if sym.Namespace() != "" {
		ns = env.FindNamespace(value.NewSymbol(sym.Namespace()))
		sym = value.NewSymbol(sym.Name())
	}
	if ns == nil {
		return nil, false
	}
	vr, ok := ns.Mappings().ValueAt(sym)
	if !ok {
		return nil, false
	}
	// TODO: can these only be vars?
	return vr.(*value.Var).Get(), true
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

func (env *environment) Stderr() io.Writer {
	return env.stderr
}

func (env *environment) FindNamespace(sym *value.Symbol) *value.Namespace {
	env.nsMtx.RLock()
	defer env.nsMtx.RUnlock()
	return env.namespaces[sym.String()]
}

func (env *environment) FindOrCreateNamespace(sym *value.Symbol) *value.Namespace {
	ns := env.FindNamespace(sym)
	if ns != nil {
		return ns
	}
	env.nsMtx.Lock()
	defer env.nsMtx.Unlock()
	ns = env.namespaces[sym.String()]
	if ns != nil {
		return ns
	}
	ns = value.NewNamespace(sym)
	env.namespaces[sym.String()] = ns
	return ns
}

func (env *environment) CurrentNamespace() *value.Namespace {
	return env.currentNamespaceVar.Get().(*value.Namespace)
}

func (env *environment) SetCurrentNamespace(ns *value.Namespace) {
	env.currentNamespaceVar.BindRoot(ns)
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

	return fmt.Errorf("%s: "+format, append([]interface{}{location}, args...)...)
}

func (env *environment) Eval(n interface{}) (interface{}, error) {
	switch v := n.(type) {
	case *value.List:
		return env.evalList(v)
	case *value.Vector:
		return env.evalVector(v)
	default:
		return env.evalScalar(n)
	}
}

func (env *environment) evalList(n *value.List) (interface{}, error) {
	if n.IsEmpty() {
		return n, nil
	}

	first := n.Item()
	if sym, ok := first.(*value.Symbol); ok {
		// handle special forms
		switch sym.String() {
		case "def":
			return env.evalDef(n)
		case "if":
			return env.evalIf(n)
		case "case*":
			return env.evalCase(n)
		case "fn*":
			return env.evalFn(n)
		case "quote":
			return env.evalQuote(n)
		case "quasiquote":
			return env.evalQuasiquote(n)
		case "let*":
			return env.evalLet(n)

		case "var":
			return env.evalVar(n)

		case "throw":
			return env.evalThrow(n)

			// Go interop special forms
		case ".":
			return env.evalDot(n)
		case "new":
			return env.evalNew(n)
		case "set!":
			return env.evalSet(n)
		}

		// handle field or method call shorthand syntax (symbol beginning with a dot)
		if strings.HasPrefix(sym.String(), ".") {
			return env.evalFieldOrMethod(n)
		}

		// handle macros
		if macroVar := env.asMacro(sym); macroVar != nil {
			applyer, ok := macroVar.Get().(value.Applyer)
			if !ok {
				return nil, env.errorf(n, "macro %s is not a function", sym)
			}
			res, err := env.applyMacro(applyer, n)
			if err != nil {
				return nil, env.errorf(n, "error applying macro: %w", err)
			}
			return res, nil
		}
	}

	// otherwise, handle a function call
	var res []interface{}
	for cur := n; !cur.IsEmpty(); cur = cur.Next() {
		item := cur.Item()
		v, err := env.Eval(item)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}

	// TODO: construct the error here, or pass metadata, for better
	// error localization
	x, err := env.applyFunc(res[0], res[1:])
	if err != nil {
		return nil, env.errorf(n, "%w", err)
	}
	return x, nil
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
		if val, ok := env.lookup(v); ok {
			return val, nil
		}
		return nil, env.errorf(n, "undefined symbol: %s", v)
	default:
		// else, it's a literal
		return v, nil
	}
}

func (env *environment) applyFunc(f interface{}, args []interface{}) (interface{}, error) {
	res, err := value.Apply(env, f, args)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Special forms

func (env *environment) evalDef(n *value.List) (interface{}, error) {
	listLength := n.Count()
	if listLength < 2 {
		return nil, env.errorf(n, "too few arguments to def")
	}
	sym, ok := value.MustNth(n, 1).(*value.Symbol)
	if !ok {
		return nil, env.errorf(n, "invalid def, first item is not a symbol")
	}
	if listLength == 2 {
		vr := env.CurrentNamespace().Intern(env, sym)
		return vr, nil
	}

	if listLength < 3 {
		return nil, env.errorf(n, "too few arguments to def")
	}
	if listLength > 4 {
		return nil, env.errorf(n, "too many arguments to def")
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

	return env.DefVar(sym, val), nil
}

func (env *environment) evalFn(n *value.List) (interface{}, error) {
	listLength := n.Count()
	items := make([]interface{}, 0, listLength-1)
	for cur := n.Next(); !cur.IsEmpty(); cur = cur.Next() {
		items = append(items, cur.Item())
	}

	if len(items) < 1 {
		return nil, env.errorf(n, "invalid fn expression")
	}

	var fnName *value.Symbol
	if sym, ok := items[0].(*value.Symbol); ok {
		// if the first child is not a list, it's the name of the
		// function. this can be used for recursion.
		fnName = sym
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

func (env *environment) evalIf(n *value.List) (interface{}, error) {
	listLength := n.Count()
	if listLength < 3 || listLength > 4 {
		return nil, env.errorf(n, "invalid if, need `cond ifExp [elseExp]`")
	}
	cond, err := env.Eval(n.Next().Item())
	if err != nil {
		return nil, err
	}

	if value.IsTruthy(cond) {
		return env.Eval(n.Next().Next().Item())
	}

	if listLength == 4 {
		return env.Eval(n.Next().Next().Next().Item())
	}
	return nil, nil
}

// cases use syntax and most of the semantics of Clojure's case (not Scheme's).
// see https://clojuredocs.org/clojure.core/case
func (env *environment) evalCase(n *value.List) (interface{}, error) {
	listLength := n.Count()
	if listLength < 4 {
		return nil, env.errorf(n, "invalid case, need `case caseExp & caseClauses`")
	}
	cond, err := env.Eval(n.Next().Item())
	if err != nil {
		return nil, err
	}

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
				return env.Eval(result)
			}
		}
	}
	if len(cases) == 1 {
		return env.Eval(cases[0])
	}
	return nil, nil
}

func toBool(v interface{}) bool {
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
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
		if value.Equal(item.Item(), SymbolUnquote) {
			return env.Eval(item.Next().Item())
		}
		if value.Equal(item.Item(), SymbolSpliceUnquote) {
			return nil, env.errorf(item, "splice-unquote not in list")
		}

		var resultValues []interface{}
		for cur := item; !cur.IsEmpty(); cur = cur.Next() {
			if lst, ok := cur.Item().(*value.List); ok && !lst.IsEmpty() && value.Equal(lst.Item(), SymbolSpliceUnquote) {
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
			if lst, ok := cur.(*value.List); ok && !lst.IsEmpty() && value.Equal(lst.Item(), SymbolSpliceUnquote) {
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
		if !strings.HasSuffix(item.Name(), "#") {
			return item, nil
		}
		symStr := item.String()
		newName, ok := symbolNameMap[symStr]
		if !ok {
			newName = symStr[:len(symStr)-1] + "__" + strconv.Itoa(env.gensymCounter) + "__auto__"
			symbolNameMap[symStr] = newName
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
// ==============================================================================
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
// ==============================================================================
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
func (env *environment) evalLet(n *value.List) (interface{}, error) {
	items := listAsSlice(n)
	if len(items) < 3 {
		return nil, env.errorf(n, "invalid let, need bindings and body")
	}

	var bindingsMap map[string]interface{}
	var err error
	switch bindings := items[1].(type) {
	case *value.Vector:
		bindingsMap, err = env.evalBindings(bindings)
	default:
		return nil, env.errorf(n, "invalid let, bindings must be a list or vector")
	}
	if err != nil {
		return nil, err
	}

	// create a new environment with the bindings
	newEnv := env.PushScope().(*environment)
	for name, val := range bindingsMap {
		newEnv.Define(value.NewSymbol(name), val)
	}

	// evaluate the body
	for _, item := range items[2 : len(items)-1] {
		_, err = newEnv.Eval(item)
		if err != nil {
			return nil, err
		}
	}
	return newEnv.Eval(items[len(items)-1])
}

func (env *environment) evalBindings(bindings *value.Vector) (map[string]interface{}, error) {
	if bindings.Count()%2 != 0 {
		return nil, env.errorf(bindings, "invalid let, bindings must be a vector of even length")
	}

	newEnv := env.PushScope().(*environment)
	bindingsMap := make(map[string]interface{})
	for i := 0; i < bindings.Count(); i += 2 {
		pattern := bindings.ValueAt(i)
		val, err := newEnv.Eval(bindings.ValueAt(i + 1))
		if err != nil {
			return nil, err
		}
		// TODO: replace with macro
		binds, err := value.Bind(pattern, val)
		if err != nil {
			return nil, env.errorf(bindings, "invalid let: %w", err)
		}

		for i := 0; i < len(binds); i += 2 {
			name, ok := binds[i].(*value.Symbol)
			if !ok {
				return nil, env.errorf(bindings, "invalid let, binding name must be a symbol")
			}
			newEnv.Define(name, binds[i+1])
			bindingsMap[name.String()] = binds[i+1]
		}
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

	env.DefineMacro(sym.String(), fn.(*value.Func))
	return nil, nil
}

func (env *environment) evalVar(n *value.List) (interface{}, error) {
	if n.Count() != 2 {
		return nil, env.errorf(n, "invalid var, need name")
	}

	sym, ok := value.MustNth(n, 1).(*value.Symbol)
	if !ok {
		return nil, env.errorf(n.Next().Item(), "invalid var, name must be a symbol")
	}

	return env.lookupVar(sym, false, true)
}

func (env *environment) applyMacro(fn value.Applyer, form *value.List) (interface{}, error) {
	argList := form.Next()
	// two hidden arguments, $form and $env.
	// $form is the form that was passed to the macro
	// $env is the environment that the macro was called in
	args := append([]interface{}{form, nil}, listAsSlice(argList)...)
	exp, err := env.applyFunc(fn, args)
	if err != nil {
		return nil, err
	}
	res := exp
	switch exp := exp.(type) {
	case *value.List:
		// ignore
	case value.ISeq:
		// convert to a list
		var elements []interface{}
		for !exp.IsEmpty() {
			elements = append(elements, exp.First())
			exp = exp.Rest()
		}
		res = value.NewList(elements)
	}
	return env.Eval(res)
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
	typeValue, ok := typeValIfc.(reflect.Type)
	if !ok {
		return nil, env.errorf(value.MustNth(n, 1), "invalid expression, expected (new <type> <field_value>*)")
	}

	val := reflect.New(typeValue)
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

	return val.Interface(), nil
}

func (env *environment) evalSet(n *value.List) (interface{}, error) {
	argCount := n.Count() - 1
	if argCount != 2 {
		return nil, env.errorf(n, "invalid expression, expected (set! (. go-value-expr field-symbol) expr)")
	}
	// evaluate the go-value-expr, which should be a go struct.
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
		if dotSym, ok := value.MustNth(fieldExpr, 0).(*value.Symbol); !ok || dotSym.String() != "." {
			return nil, env.errorf(fieldExpr, "invalid expression, expected (set! (. go-value-expr field-symbol) expr)")
		}
	}

	fieldSym, ok := value.MustNth(fieldExpr, 2).(*value.Symbol)
	if !ok {
		return nil, env.errorf(value.MustNth(fieldExpr, 2), "invalid set! expression, expected a symbol")
	}
	targetVal, err := env.Eval(value.MustNth(fieldExpr, 1))
	if err != nil {
		return nil, err
	}

	expr, err := env.Eval(value.MustNth(n, 2))
	if err != nil {
		return nil, err
	}

	// TODO: bother with this?
	var goVal interface{}
	if goValuer, ok := expr.(value.GoValuer); ok {
		goVal = goValuer.GoValue()
	} else {
		goVal = expr
	}

	if err := value.SetField(targetVal, fieldSym.String(), goVal); err != nil {
		return nil, env.errorf(n, "invalid set! expression, %v", err)
	}

	return expr, nil
}

func (env *environment) evalThrow(n *value.List) (interface{}, error) {
	if n.Count() != 2 {
		return nil, env.errorf(n, "too many arguments to throw")
	}
	expr, err := env.Eval(value.MustNth(n, 1))
	if err != nil {
		return nil, err
	}

	if err, ok := expr.(error); ok {
		return nil, env.errorf(n, "%w", err)
	}

	return nil, env.errorf(n, "invalid throw expression, expected an error")
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

	memberExpr := n.Next().Next().Item()
	if dotCount > 3 {
		// must be a convenience form for a method call, e.g. (. foo bar baz)
		// use the tail as the member expression.
		memberExpr = n.Next().Next()
	}

	if v, ok := memberExpr.(*value.Symbol); ok {
		fieldVal := value.FieldOrMethod(target, v.String())
		if fieldVal == nil {
			return nil, env.errorf(v, "%T has no such field or method (%s)", target, v)
		}

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

		method := value.FieldOrMethod(target, sym.String())
		if method == nil {
			return nil, env.errorf(sym, "%T has no such method (%s)", target, sym)
		}
		args := make([]interface{}, v.Next().Count())
		for i := range args {
			v, err := env.Eval(value.MustNth(v, i+1))
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return value.Apply(env, method, args)
	}

	return nil, fmt.Errorf("unimplemented")
}

func (env *environment) evalFieldOrMethod(n *value.List) (interface{}, error) {
	if n.Next().IsEmpty() {
		return nil, env.errorf(n, "invalid expression, expecting (.method target ...)")
	}

	sym := n.Item().(*value.Symbol)
	fieldSym := value.NewSymbol(sym.String()[1:])

	// rewrite the expression to a dot expression
	newList := n.Next().Next().Conj(fieldSym, value.MustNth(n, 1), SymbolDot).(*value.List)
	// newList := value.NewList([]interface{}{
	// 	SymbolDot,
	// 	value.MustNth(n, 1),
	// 	fieldSym,
	// })
	return env.evalDot(newList)
}

// Helpers

func (env *environment) lookupVar(sym *value.Symbol, internNew, registerMacro bool) (*value.Var, error) {
	// Translated from clojure's Compiler.java
	var result *value.Var
	switch {
	case sym.Namespace() != "":
		ns := env.namespaceForSymbol(sym)
		if ns == nil {
			return nil, env.errorf(sym, "unable to resolve %s", sym)
		}
		nameSym := value.NewSymbol(sym.Name())
		if internNew && ns == env.CurrentNamespace() {
			result = ns.Intern(env, nameSym)
		} else {
			result = ns.FindInternedVar(nameSym)
		}
	case sym.Equal(SymbolNamespace):
		result = env.namespaceVar
	case sym.Equal(SymbolInNamespace):
		result = env.inNamespaceVar
	default:
		// is it mapped?
		v := env.CurrentNamespace().GetMapping(sym)
		if v == nil {
			// introduce a new var in the current ns
			if internNew {
				result = env.CurrentNamespace().Intern(env, value.NewSymbol(sym.Name()))
			}
		} else if v, ok := v.(*value.Var); ok {
			result = v
		} else {
			return nil, env.errorf(sym, "expecting var, but %s is mapped to %T", sym, v)
		}
	}
	if result != nil && (!result.IsMacro() || registerMacro) {
		env.registerVar(result)
	}
	return result, nil
}

func (env *environment) namespaceForSymbol(sym *value.Symbol) *value.Namespace {
	return env.namespaceFor(env.CurrentNamespace(), sym)
}

func (env *environment) namespaceFor(inns *value.Namespace, sym *value.Symbol) *value.Namespace {
	//note, presumes non-nil sym.ns
	// first check against currentNS' aliases...
	nsSym := value.NewSymbol(sym.Namespace())
	ns := inns.LookupAlias(nsSym)
	if ns != nil {
		return ns
	}

	return env.FindNamespace(nsSym)
}

func (env *environment) registerVar(v *value.Var) {
	// TODO: implement
}

func (env *environment) asMacro(sym *value.Symbol) *value.Var {
	vr, err := env.lookupVar(sym, false, false)
	if vr == nil || err != nil {
		return nil
	}
	// TODO: implement check for public/private
	if vr.IsMacro() {
		return vr
	}
	return nil
}

// Misc. helpers

func nodeAsStringList(n *value.List) ([]string, error) {
	var res []string
	for cur := n; !cur.IsEmpty(); cur = cur.Next() {
		item := cur.Item()
		sym, ok := item.(*value.Symbol)
		if !ok {
			return nil, fmt.Errorf("invalid argument list: %v", n)
		}
		res = append(res, sym.String())
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
