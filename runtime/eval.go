package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/compiler"
	"github.com/glojurelang/glojure/value"
)

func (env *environment) Macroexpand1(form interface{}) (interface{}, error) {
	seq, ok := form.(value.ISeq)
	if !ok {
		return form, nil
	}

	op := value.First(seq)
	sym, ok := op.(*value.Symbol)
	if !ok {
		return form, nil
	}

	if strings.HasPrefix(sym.String(), ".") && len(sym.String()) > 1 {
		fieldSym := value.NewSymbol(sym.String()[1:])
		// rewrite the expression to a dot expression
		newList := seqToList(value.NewCons(SymbolDot, value.NewCons(seq.Next().First(), value.NewCons(fieldSym, seq.Next().Next()))))
		return env.Macroexpand1(newList)
	}

	macroVar := env.asMacro(sym)
	if macroVar == nil {
		return form, nil
	}

	applyer, ok := macroVar.Get().(value.Applyer)
	if !ok {
		return nil, env.errorf(form, "macro %s is not a function", sym)
	}
	res, err := env.applyMacro1(applyer, form.(value.ISeq))
	if err != nil {
		return nil, env.errorf(form, "error applying macro: %w", err)
	}
	return res, nil
}

func (env *environment) applyMacro1(fn value.Applyer, form value.ISeq) (interface{}, error) {
	argList := form.Next()
	// two hidden arguments, $form and $env.
	// $form is the form that was passed to the macro
	// $env is the environment that the macro was called in
	args := append([]interface{}{form, nil}, seqToSlice(argList)...)

	return env.applyFunc(fn, args)
}

func (env *environment) Eval(n interface{}) (interface{}, error) {
	currentNSSym := env.CurrentNamespace().Name()
	kw := value.NewKeyword
	analyzer := &compiler.Analyzer{
		Macroexpand1: env.Macroexpand1,
		CreateVar: func(sym *value.Symbol, e compiler.Env) (interface{}, error) {
			return env.CurrentNamespace().Intern(env, sym), nil
		},
		IsVar: func(v interface{}) bool {
			_, ok := v.(*value.Var)
			return ok
		},
		Gensym: func(prefix string) *value.Symbol {
			num := env.nextSymNum()
			return value.NewSymbol(fmt.Sprintf("%s%d", prefix, num))
		},
		GlobalEnv: value.NewAtom(value.NewMap(
			kw("namespaces"), value.NewMap(
				value.NewSymbol(currentNSSym.Name()), value.NewMap(
					kw("ns"), currentNSSym,
					kw("mappings"), env.CurrentNamespace().Mappings(),
				),
			),
		)),
	}
	astNode, err := analyzer.Analyze(n, value.NewMap(
		value.NewKeyword("ns"), env.CurrentNamespace().Name(),
	))
	if err != nil {
		return nil, err
	}
	return env.EvalAST(astNode)

	switch v := n.(type) {
	case *value.Vector:
		return env.evalVector(v)
	case *value.List: // TODO: should apply to any seq...
		return env.evalList(v)
	case value.ISeq:
		// convert to a list
		var elements []interface{}
		for ; v != nil; v = v.Next() {
			elements = append(elements, v.First())
		}
		if len(elements) == 0 {
			// TODO: eval sequences, not lists. this is a hack now that
			// EmptyList is different from List
			return value.NewList(nil), nil
		}
		return env.evalList(value.NewList(elements...).(*value.List))
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
		case "do":
			return env.evalDo(n)
		case "if":
			return env.evalIf(n)
		case "case*":
			return env.evalCase(n)
		case "fn*":
			return env.evalFn(n)
		case "quote":
			return env.evalQuote(n)
		case "let*":
			return env.evalLet(n, false)
		case "loop*":
			return env.evalLet(n, true)

		case "var":
			return env.evalVar(n)

		case "recur":
			return env.evalRecur(n)

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
	for cur := value.Seq(n); cur != nil; cur = cur.Next() {
		item := cur.First()
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

	v, err := value.WithMeta(value.NewVector(res...), n.Meta())
	if err != nil {
		// this should never happen - vectors can have metadata
		panic(err)
	}
	return v, nil
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

func (env *environment) evalFn(n value.ISeq) (interface{}, error) {
	return value.ParseFunc(env, n)
}

func (env *environment) evalDo(n *value.List) (interface{}, error) {
	var res interface{}
	var err error
	for cur := n.Next(); cur != nil; cur = cur.Next() {
		res, err = env.Eval(cur.First())
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (env *environment) evalIf(n *value.List) (interface{}, error) {
	listLength := n.Count()
	if listLength < 3 || listLength > 4 {
		return nil, env.errorf(n, "invalid if, need `cond ifExp [elseExp]`")
	}
	cond, err := env.Eval(n.Next().First())
	if err != nil {
		return nil, err
	}

	if value.IsTruthy(cond) {
		return env.Eval(n.Next().Next().First())
	}

	if listLength == 4 {
		return env.Eval(n.Next().Next().Next().First())
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
	cond, err := env.Eval(n.Next().First())
	if err != nil {
		return nil, err
	}

	cases := make([]interface{}, 0, listLength-2)
	for cur := n.Next().Next(); cur != nil; cur = cur.Next() {
		cases = append(cases, cur.First())
	}

	for len(cases) >= 2 {
		test, result := cases[0], cases[1]
		cases = cases[2:]

		testItems := []interface{}{test}
		testList, ok := test.(value.ISeq)
		if ok {
			var testItems []interface{}
			for cur := testList; cur != nil; cur = cur.Next() {
				testItems = append(testItems, cur.First())
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

	return n.Next().First(), nil
}

func (env *environment) evalLet(n *value.List, isLoop bool) (interface{}, error) {
	items := seqToSlice(n)
	if len(items) < 3 {
		return nil, env.errorf(n, "invalid let, need bindings and body")
	}

	var bindNameVals []interface{}
	var err error
	bindings, ok := items[1].(*value.Vector)
	if !ok {
		return nil, env.errorf(n, "invalid let, bindings must be a vector")
	}

	bindNameVals, err = env.evalBindings(bindings)
	if err != nil {
		return nil, err
	}
	// create a new environment with the bindings
	newEnv := env.PushScope().(*environment)

Recur:
	for i := 0; i < len(bindNameVals); i += 2 {
		name := bindNameVals[i].(string)
		val := bindNameVals[i+1]
		newEnv.BindLocal(value.NewSymbol(name), val)
	}

	// evaluate the body
	for _, item := range items[2 : len(items)-1] {
		_, err = newEnv.Eval(item)
		if err != nil {
			return nil, err
		}
	}

	rt := value.NewRecurTarget()
	recurEnv := newEnv.WithRecurTarget(rt)
	recurErr := &value.RecurError{Target: rt}

	res, err := recurEnv.Eval(items[len(items)-1])
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

func (env *environment) evalBindings(bindings *value.Vector) ([]interface{}, error) {
	if bindings.Count()%2 != 0 {
		return nil, env.errorf(bindings, "invalid let, bindings must be a vector of even length")
	}

	newEnv := env.PushScope().(*environment)
	var bindingNameVals []interface{}
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
			newEnv.BindLocal(name, binds[i+1])
			bindingNameVals = append(bindingNameVals, name.String(), binds[i+1])
		}
	}

	return bindingNameVals, nil
}

func (env *environment) evalDefMacro(n *value.List) (interface{}, error) {
	// fnList is a transformed version of the macro that looks like a fn
	// form, which is nearly the same as the defmacro form but without
	// the docstring or metadata.
	var fnList value.ISeq = n
	if n.Count() > 3 {
		_, ok := value.MustNth(n, 2).(string)
		if ok {
			argBody := n.Next().Next().Next()
			fnList = seqToList(value.NewCons(value.MustNth(n, 0), value.NewCons(value.MustNth(n, 1), argBody)))
			//fnList = argBody.Conj(value.MustNth(n, 1)).Conj(value.MustNth(n, 0)).(*value.List)
			// TODO: store the docstring somewhere
		}
	}

	fn, err := env.evalFn(fnList)
	if err != nil {
		return nil, err
	}

	sym, ok := value.MustNth(n, 1).(*value.Symbol)
	if !ok {
		return nil, env.errorf(n.Next().First(), "invalid defmacro, name must be a symbol")
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
		return nil, env.errorf(n.Next().First(), "invalid var, name must be a symbol")
	}

	return env.lookupVar(sym, false, true)
}

func (env *environment) applyMacro(fn value.Applyer, form value.ISeq) (interface{}, error) {
	argList := form.Next()
	// two hidden arguments, $form and $env.
	// $form is the form that was passed to the macro
	// $env is the environment that the macro was called in
	args := append([]interface{}{form, nil}, seqToSlice(argList)...)

	exp, err := env.applyFunc(fn, args)
	if err != nil {
		return nil, err
	}
	return env.Eval(exp)
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
	for cur := n.Next().Next(); cur != nil; cur = cur.Next().Next() {
		fieldName, ok := cur.First().(value.Keyword)
		if !ok {
			return nil, env.errorf(cur.First(), "invalid new expression, field name must be a keyword")
		}
		fieldValue, err := env.Eval(cur.Next().First())
		if err != nil {
			return nil, err
		}
		fieldNameStr := fieldName.Name()[1:]
		field := val.Elem().FieldByName(fieldNameStr)
		if !field.IsValid() {
			return nil, env.errorf(cur.First(), "invalid new expression, unknown field (%v)", fieldNameStr)
		}
		if !field.CanSet() {
			return nil, env.errorf(cur.First(), "invalid new expression, field is not settable (%v)", fieldNameStr)
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

	// TODO: bother with this? GoValuer interface probably isn't necessary
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

func (env *environment) evalRecur(n *value.List) (interface{}, error) {
	if env.recurTarget == nil {
		return nil, env.errorf(n, "invalid recur expression, not in a recur target")
	}

	// TODO: ensure that the recur is in a tail position!

	var args []interface{}
	noRecurEnv := env.WithRecurTarget(nil)
	for cur := n.Next(); cur != nil; cur = cur.Next() {
		arg, err := noRecurEnv.Eval(cur.First())
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	return nil, &value.RecurError{
		Target: env.recurTarget,
		Args:   args,
	}
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

func (env *environment) evalDot(n value.ISeq) (interface{}, error) {
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

	dotCount := value.Count(n)

	if dotCount < 3 {
		return nil, env.errorf(n, "invalid expression, expecting (. target member ...)")
	}

	target, err := env.Eval(n.Next().First())
	if err != nil {
		return nil, err
	}

	memberExpr := n.Next().Next().First()
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
		sym, ok := v.First().(*value.Symbol)
		if !ok {
			return nil, env.errorf(v.First(), "invalid expression, method name must be a symbol")
		}

		method := value.FieldOrMethod(target, sym.String())
		if method == nil {
			return nil, env.errorf(sym, "%T has no such method (%s)", target, sym)
		}
		var args []interface{}
		for cur := v.Next(); cur != nil; cur = cur.Next() {
			v, err := env.Eval(cur.First())
			if err != nil {
				return nil, err
			}
			args = append(args, v)
		}
		return value.Apply(env, method, args)
	}

	return nil, fmt.Errorf("unimplemented")
}

func (env *environment) evalFieldOrMethod(n *value.List) (interface{}, error) {
	if n.Next() == nil {
		return nil, env.errorf(n, "invalid expression, expecting (.method target ...)")
	}

	sym := n.First().(*value.Symbol)
	fieldSym := value.NewSymbol(sym.String()[1:])

	// rewrite the expression to a dot expression
	newList := seqToList(value.NewCons(SymbolDot, value.NewCons(n.Next().First(), value.NewCons(fieldSym, n.Next().Next()))))
	//newList := n.Next().Next().Conj(fieldSym).Conj(value.MustNth(n, 1)).Conj(SymbolDot).(*value.List)
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

func seqToList(seq value.ISeq) value.IPersistentList {
	return value.NewList(seqToSlice(seq)...)
}

func seqToSlice(seq value.ISeq) []interface{} {
	if seq == nil {
		return nil
	}
	var items []interface{}
	for ; seq != nil; seq = seq.Next() {
		items = append(items, seq.First())
	}
	return items
}

func isNilableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}
