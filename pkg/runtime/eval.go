package runtime

import (
	"fmt"

	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (env *environment) Macroexpand1(form interface{}) (interface{}, error) {
	seq, ok := form.(lang.ISeq)
	if !ok {
		return form, nil
	}

	op := lang.First(seq)
	sym, ok := op.(*lang.Symbol)
	if !ok {
		return form, nil
	}

	symStr := sym.String()
	if len(symStr) > 1 && symStr[0] == '.' && symStr[1] != '.' {
		fieldSym := lang.NewSymbol(sym.String()[1:])
		// rewrite the expression to a dot expression
		dotExpr := lang.NewCons(SymbolDot, lang.NewCons(seq.Next().First(), lang.NewCons(fieldSym, seq.Next().Next())))
		return env.Macroexpand1(dotExpr)
	}

	macroVar := env.asMacro(sym)
	if macroVar == nil {
		return form, nil
	}

	applyer, ok := macroVar.Get().(lang.IFn)
	if !ok {
		return nil, env.errorf(form, "macro %s is not a function (%T)", sym, macroVar.Get())
	}
	res, err := env.applyMacro(applyer, form.(lang.ISeq))
	if err != nil {
		return nil, env.errorf(form, "error applying macro: %w", err)
	}
	return res, nil
}

func (env *environment) applyMacro(fn lang.IFn, form lang.ISeq) (interface{}, error) {
	argList := form.Next()
	// two hidden arguments, $form and $env (nil for now).
	// $form is the form that was passed to the macro
	// $env is the environment that the macro was called in
	return fn.ApplyTo(lang.NewCons(form, lang.NewCons(nil, argList))), nil
}

func (env *environment) Eval(n interface{}) (interface{}, error) {
	return env.evalInternal(n)
}

func (env *environment) evalInternal(n interface{}) (interface{}, error) {
	analyzer := &compiler.Analyzer{
		Macroexpand1: env.Macroexpand1,
		CreateVar: func(sym *lang.Symbol, e compiler.Env) (interface{}, error) {
			vr := env.CurrentNamespace().Intern(sym)
			return vr, nil
		},
		IsVar: func(v interface{}) bool {
			_, ok := v.(*lang.Var)
			return ok
		},
		Gensym: func(prefix string) *lang.Symbol {
			num := env.nextSymNum()
			return lang.NewSymbol(fmt.Sprintf("%s%d", prefix, num))
		},
		FindNamespace: lang.FindNamespace,
	}
	astNode, err := analyzer.Analyze(n, lang.NewMap(
		lang.KWNS, env.CurrentNamespace().Name(),
	))
	if err != nil {
		return nil, err
	}
	return env.EvalAST(astNode)
}

// Helpers

func (env *environment) lookupVar(sym *lang.Symbol, internNew, registerMacro bool) (*lang.Var, error) {
	// Translated from clojure's Compiler.java
	var result *lang.Var
	switch {
	case sym.Namespace() != "":
		ns := env.namespaceForSymbol(sym)
		if ns == nil {
			return nil, env.errorf(sym, "unable to resolve %s", sym)
		}
		nameSym := lang.NewSymbol(sym.Name())
		if internNew && ns == env.CurrentNamespace() {
			result = ns.Intern(nameSym)
		} else {
			result = ns.FindInternedVar(nameSym)
		}
	case sym.Equals(SymbolNamespace):
		result = env.namespaceVar
	case sym.Equals(SymbolInNamespace):
		result = env.inNamespaceVar
	default:
		// is it mapped?
		v := env.CurrentNamespace().GetMapping(sym)
		if v == nil {
			// introduce a new var in the current ns
			if internNew {
				result = env.CurrentNamespace().Intern(lang.NewSymbol(sym.Name()))
			}
		} else if v, ok := v.(*lang.Var); ok {
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

func (env *environment) namespaceForSymbol(sym *lang.Symbol) *lang.Namespace {
	return lang.NamespaceFor(env.CurrentNamespace(), sym)
}

func (env *environment) registerVar(v *lang.Var) {
	// TODO: implement
}

func (env *environment) asMacro(sym *lang.Symbol) *lang.Var {
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

func seqToSlice(seq lang.ISeq) []interface{} {
	if seq == nil {
		return nil
	}
	var items []interface{}
	for ; seq != nil; seq = seq.Next() {
		items = append(items, seq.First())
	}
	return items
}
