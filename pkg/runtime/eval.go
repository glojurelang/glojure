package runtime

import (
	"fmt"
	"io"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

// Core forms resolve these exports during analysis. Keep compiler intrinsics
// independent of the large, optional Go export registry.
const (
	numbersHostExport            = "github.com:glojurelang:glojure:pkg:lang.Numbers"
	findNamespaceHostExport      = "github.com:glojurelang:glojure:pkg:lang.FindNamespace"
	pushThreadBindingsHostExport = "github.com:glojurelang:glojure:pkg:lang.PushThreadBindings"
	popThreadBindingsHostExport  = "github.com:glojurelang:glojure:pkg:lang.PopThreadBindings"
	lockingTransactionHostExport = "github.com:glojurelang:glojure:pkg:lang.LockingTransaction"
)

func resolveHost(sym *lang.Symbol) (interface{}, bool) {
	switch sym.String() {
	case numbersHostExport:
		return lang.Numbers, true
	case findNamespaceHostExport:
		return lang.FindNamespace, true
	case pushThreadBindingsHostExport:
		return lang.PushThreadBindings, true
	case popThreadBindingsHostExport:
		return lang.PopThreadBindings, true
	case lockingTransactionHostExport:
		return lang.LockingTransaction, true
	default:
		return pkgmap.Get(sym.String())
	}
}

func (env *environment) Macroexpand1(form interface{}) (interface{}, error) {
	return env.macroexpand1(form, env.CurrentNamespace())
}

func (env *environment) macroexpand1(
	form interface{},
	currentNS *lang.Namespace,
) (interface{}, error) {
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
		return env.macroexpand1(dotExpr, currentNS)
	}

	macroVar := env.asMacroInNamespace(sym, currentNS)
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
	if env.astDumpEnabled() {
		return env.evalInternalInNamespace(n, env.CurrentNamespace())
	}
	if directSelfEvaluating(n) {
		return n, nil
	}
	currentNS := env.CurrentNamespace()
	if result, ok, err := env.evalDirectInt64ReducePipeline(n, currentNS); ok {
		return result, err
	}
	if result, ok, err := env.evalDirectInvoke(n, currentNS); ok {
		return result, err
	}
	return env.evalInternalInNamespace(n, currentNS)
}

func directSelfEvaluating(form interface{}) bool {
	switch form.(type) {
	case *lang.Symbol,
		lang.IPersistentVector,
		lang.IPersistentMap,
		lang.IPersistentSet,
		lang.ISeq:
		return false
	default:
		return true
	}
}

func (env *environment) evalInternal(n interface{}) (interface{}, error) {
	currentNS := env.CurrentNamespace()
	return env.evalInternalInNamespace(n, currentNS)
}

func (env *environment) evalInternalInNamespace(
	n interface{},
	currentNS *lang.Namespace,
) (interface{}, error) {
	astNode, err := env.analyzeInternalInNamespace(n, currentNS)
	if err != nil {
		return nil, err
	}
	return env.EvalAST(astNode)
}

func (env *environment) analyzeInternalInNamespace(
	n interface{},
	currentNS *lang.Namespace,
) (*ast.Node, error) {
	analyzer := &compiler.Analyzer{
		Macroexpand1: func(form interface{}) (interface{}, error) {
			return env.macroexpand1(form, currentNS)
		},
		CreateVar: func(sym *lang.Symbol, e compiler.Env) (interface{}, error) {
			vr := currentNS.Intern(sym)
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
		ResolveHost:   resolveHost,
		Optimizer: compiler.NewDefaultOptimizer(compiler.OptimizationOptions{
			DirectLinking: directLinkEnabled(),
		}),
	}
	node, err := analyzer.Analyze(n, lang.NewMap(
		lang.KWNS, currentNS.Name(),
	))
	if err != nil {
		return nil, err
	}
	if err := env.dumpAST(node); err != nil {
		return nil, err
	}
	return node, nil
}

// EvalWithASTDump analyzes form, writes the post-optimization AST, and
// evaluates that exact tree. It is intended for compiler diagnostics.
func EvalWithASTDump(form interface{}, w io.Writer) (interface{}, error) {
	env, ok := lang.GlobalEnv.(*environment)
	if !ok {
		return nil, fmt.Errorf("runtime: AST dumping requires the Glojure environment")
	}
	node, err := env.analyzeInternalInNamespace(form, env.CurrentNamespace())
	if err != nil {
		return nil, err
	}
	if err := ast.Dump(w, node); err != nil {
		return nil, err
	}
	return env.EvalAST(node)
}

// SetASTDumpWriter enables post-optimization AST dumping for all evaluations
// in the global Glojure environment, including forms loaded recursively.
// Passing nil disables dumping. The returned function restores the old writer.
func SetASTDumpWriter(w io.Writer) (func(), error) {
	env, ok := lang.GlobalEnv.(*environment)
	if !ok {
		return nil, fmt.Errorf("runtime: AST dumping requires the Glojure environment")
	}
	if env.astDump == nil {
		env.astDump = &astDumpState{}
	}
	env.astDump.mu.Lock()
	old := env.astDump.writer
	env.astDump.writer = w
	env.astDump.mu.Unlock()
	return func() {
		env.astDump.mu.Lock()
		env.astDump.writer = old
		env.astDump.mu.Unlock()
	}, nil
}

func (env *environment) astDumpEnabled() bool {
	if env.astDump == nil {
		return false
	}
	env.astDump.mu.Lock()
	defer env.astDump.mu.Unlock()
	return env.astDump.writer != nil
}

func (env *environment) dumpAST(node *ast.Node) error {
	if env.astDump == nil {
		return nil
	}
	env.astDump.mu.Lock()
	defer env.astDump.mu.Unlock()
	if env.astDump.writer == nil {
		return nil
	}
	return ast.Dump(env.astDump.writer, node)
}

func (env *environment) evalDirectInvoke(
	form interface{},
	currentNS *lang.Namespace,
) (result interface{}, ok bool, err error) {
	seq, ok := form.(lang.ISeq)
	if !ok || seq == nil {
		return nil, false, nil
	}
	op, ok := seq.First().(*lang.Symbol)
	if !ok {
		return nil, false, nil
	}
	vr := directInvokeVar(currentNS, op)
	if vr == nil || vr.IsMacro() {
		return nil, false, nil
	}
	fn, callable := vr.Get().(lang.IFn)
	if !callable {
		return nil, false, nil
	}

	var args [4]interface{}
	arity := 0
	for argSeq := seq.Next(); argSeq != nil; argSeq = argSeq.Next() {
		if arity == len(args) {
			return nil, false, nil
		}
		value, constant := directInvokeConstant(argSeq.First())
		if !constant {
			return nil, false, nil
		}
		args[arity] = value
		arity++
	}

	ok = true
	defer env.recoverDirectInvoke(form, &result, &err)
	switch arity {
	case 0:
		result = lang.Apply0(fn)
	case 1:
		result = lang.Apply1(fn, args[0])
	case 2:
		result = lang.Apply2(fn, args[0], args[1])
	case 3:
		result = lang.Apply3(fn, args[0], args[1], args[2])
	case 4:
		result = lang.Apply4(fn, args[0], args[1], args[2], args[3])
	}
	return result, true, nil
}

func directInvokeVar(currentNS *lang.Namespace, sym *lang.Symbol) *lang.Var {
	if sym.Namespace() == "" {
		vr, _ := currentNS.GetMapping(sym).(*lang.Var)
		return vr
	}
	ns := lang.NamespaceFor(currentNS, sym)
	if ns == nil {
		return nil
	}
	vr := ns.FindInternedVar(lang.NewSymbol(sym.Name()))
	if vr == nil || ns != currentNS && !vr.IsPublic() {
		return nil
	}
	return vr
}

func directInvokeConstant(form interface{}) (interface{}, bool) {
	seq, ok := form.(lang.ISeq)
	if !ok || seq == nil {
		return nil, false
	}
	op, ok := seq.First().(*lang.Symbol)
	if !ok || op.String() != "quote" && op.String() != "clojure.core/quote" {
		return nil, false
	}
	args := seq.Next()
	if args == nil || args.Next() != nil {
		return nil, false
	}
	return args.First(), true
}

// Helpers

func (env *environment) lookupVarInNamespace(
	sym *lang.Symbol,
	internNew, registerMacro bool,
	currentNS *lang.Namespace,
) (*lang.Var, error) {
	// Translated from clojure's Compiler.java
	var result *lang.Var
	switch {
	case sym.Namespace() != "":
		ns := lang.NamespaceFor(currentNS, sym)
		if ns == nil {
			return nil, env.errorf(sym, "unable to resolve %s", sym)
		}
		nameSym := lang.NewSymbol(sym.Name())
		if internNew && ns == currentNS {
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
		v := currentNS.GetMapping(sym)
		if v == nil {
			// introduce a new var in the current ns
			if internNew {
				result = currentNS.Intern(lang.NewSymbol(sym.Name()))
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

func (env *environment) registerVar(v *lang.Var) {
	// TODO: implement
}

func (env *environment) asMacroInNamespace(sym *lang.Symbol, currentNS *lang.Namespace) *lang.Var {
	vr, err := env.lookupVarInNamespace(sym, false, false, currentNS)
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
