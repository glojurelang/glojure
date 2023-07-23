package runtime

import (
	"fmt"
	"strings"

	"github.com/glojurelang/glojure/pkg/compiler"
	compiler2 "github.com/glojurelang/glojure/pkg/compiler2"
	"github.com/glojurelang/glojure/pkg/lang"
	value "github.com/glojurelang/glojure/pkg/lang"
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
		dotExpr := value.NewCons(SymbolDot, value.NewCons(seq.Next().First(), value.NewCons(fieldSym, seq.Next().Next())))
		return env.Macroexpand1(dotExpr)
	}

	macroVar := env.asMacro(sym)
	if macroVar == nil {
		return form, nil
	}

	applyer, ok := macroVar.Get().(value.IFn)
	if !ok {
		return nil, env.errorf(form, "macro %s is not a function (%T)", sym, macroVar.Get())
	}
	res, err := env.applyMacro(applyer, form.(value.ISeq))
	if err != nil {
		return nil, env.errorf(form, "error applying macro: %w", err)
	}
	return res, nil
}

func (env *environment) applyMacro(fn value.IFn, form value.ISeq) (interface{}, error) {
	argList := form.Next()
	// two hidden arguments, $form and $env (nil for now).
	// $form is the form that was passed to the macro
	// $env is the environment that the macro was called in
	return fn.ApplyTo(value.NewCons(form, value.NewCons(nil, argList))), nil
}

func (env *environment) Eval(n interface{}) (interface{}, error) {
	return env.evalInternal(n)
}

func (env *environment) evalInternal(n interface{}) (interface{}, error) {
	kw := value.NewKeyword

	globalEnv := value.NewAtom(nil)
	resetGlobalEnv := func() {
		kwNS := kw("ns")
		kwMappings := kw("mappings")
		kwAliases := kw("aliases")
		namespaces := value.Namespaces()
		namespaceKVs := make([]interface{}, 0, 2*len(namespaces))
		for _, ns := range namespaces {
			aliases := make([]interface{}, 0, 2*value.Count(ns.Aliases()))
			for seq := value.Seq(ns.Aliases()); seq != nil; seq = seq.Next() {
				entry := seq.First().(value.IMapEntry)
				aliases = append(aliases, entry.Key(), entry.Val().(*value.Namespace).Name())
			}
			namespaceKVs = append(namespaceKVs, ns.Name(), value.NewMap(
				kwNS, ns.Name(),
				kwMappings, ns.Mappings(),
				kwAliases, value.NewMap(aliases...),
			))
		}
		globalEnv.Reset(value.NewMap(kw("namespaces"), value.NewMap(namespaceKVs...)))
	}
	resetGlobalEnv()

	if false {
		analyzer := &compiler2.Analyzer{
			Macroexpand1: env.Macroexpand1,
			CreateVar: func(sym *value.Symbol, e compiler2.Env) (interface{}, error) {
				vr := env.CurrentNamespace().Intern(sym)
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
			FindNamespace: lang.FindNamespace,
		}
		_, err := analyzer.Analyze(n, value.NewMap(
			value.KWNS, env.CurrentNamespace().Name(),
		))
		if err != nil {
			panic(err)
		}
	}

	analyzer := &compiler.Analyzer{
		Macroexpand1: env.Macroexpand1,
		CreateVar: func(sym *value.Symbol, e compiler.Env) (interface{}, error) {
			vr := env.CurrentNamespace().Intern(sym)
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
		value.KWNS, env.CurrentNamespace().Name(),
	))
	if err != nil {
		return nil, err
	}
	return env.EvalAST(astNode)
}

func (env *environment) applyFunc(f interface{}, args []interface{}) (interface{}, error) {
	res, err := value.Apply(f, args)
	if err != nil {
		return nil, err
	}
	return res, nil
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
			result = ns.Intern(nameSym)
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
				result = env.CurrentNamespace().Intern(value.NewSymbol(sym.Name()))
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
	return value.NamespaceFor(env.CurrentNamespace(), sym)
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
