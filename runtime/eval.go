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

	globalEnv := value.NewAtom(nil)
	resetGlobalEnv := func() {
		globalEnv.Reset(value.NewMap(
			kw("namespaces"), value.NewMap(
				value.NewSymbol(currentNSSym.Name()), value.NewMap(
					kw("ns"), currentNSSym,
					kw("mappings"), env.CurrentNamespace().Mappings(),
				))))
	}
	resetGlobalEnv()

	analyzer := &compiler.Analyzer{
		Macroexpand1: env.Macroexpand1,
		CreateVar: func(sym *value.Symbol, e compiler.Env) (interface{}, error) {
			vr := env.CurrentNamespace().Intern(env, sym)
			resetGlobalEnv()
			return vr, nil
		},
		IsVar: func(v interface{}) bool {
			_, ok := v.(*value.Var)
			return ok
		},
		Gensym: func(prefix string) *value.Symbol {
			num := env.nextSymNum()
			return value.NewSymbol(fmt.Sprintf("%s%d", prefix, num))
		},
		GlobalEnv: globalEnv,
	}
	astNode, err := analyzer.Analyze(n, value.NewMap(
		value.NewKeyword("ns"), env.CurrentNamespace().Name(),
	))
	if err != nil {
		return nil, err
	}
	return env.EvalAST(astNode)
}

func (env *environment) applyFunc(f interface{}, args []interface{}) (interface{}, error) {
	res, err := value.Apply(env, f, args)
	if err != nil {
		return nil, err
	}
	return res, nil
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
