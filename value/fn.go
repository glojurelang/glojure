package value

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// MaxPositionalArity is the maximum number of positional arguments
	// that can be passed to a function.
	MaxPositionalArity = 20
)

// Func is a function.
type Func struct {
	Section
	name           *Symbol
	env            Environment // TODO: only track closure bindings
	methods        []*FuncMethod
	variadicMethod *FuncMethod
}

type FuncMethod struct {
	requiredParams []*Symbol
	restParam      *Symbol
	body           ISeq
}

func (f *FuncMethod) recurParamLength() int {
	if f.restParam == nil {
		return len(f.requiredParams)
	}
	return len(f.requiredParams) + 1
}

func (f *FuncMethod) String() string {
	b := strings.Builder{}
	b.WriteRune('[')
	for i, sym := range f.requiredParams {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(sym.String())
	}
	if f.restParam != nil {
		if len(f.requiredParams) > 0 {
			b.WriteString(" ")
		}
		b.WriteString("& ")
		b.WriteString(f.restParam.String())
	}
	b.WriteString("] ")
	for cur := f.body; cur != nil; cur = cur.Next() {
		if cur != f.body {
			b.WriteString(" ")
		}
		b.WriteString(ToString(cur.First()))
	}
	return b.String()
}

func ParseFunc(env Environment, form ISeq) (*Func, error) {
	if form == nil {
		return nil, env.Errorf(form, "fn requires a form")
	}
	rest := form.Next()
	name, ok := rest.First().(*Symbol)
	if ok {
		rest = rest.Next()
	}
	if rest == nil {
		return nil, env.Errorf(form, "invalid fn expression, expected function definition")
	}

	// if the first remaining element is a vector, then we have a single
	// signature. to simplify the code below, we'll just wrap it in a
	// sequence and proceed to parse a sequence of sequences.
	if _, ok := rest.First().(*Vector); ok {
		rest = NewList(rest)
	}

	fn := &Func{
		name: name,
		env:  env,
	}

	for cur := rest; cur != nil; cur = cur.Next() {
		seq, ok := cur.First().(ISeq)
		if !ok {
			return nil, env.Errorf(cur.First(), "invalid fn expression, expected sequence of parameters and body, got %T", cur.First())
		}
		meth, err := parseMethod(env, seq)
		if err != nil {
			return nil, err
		}
		if meth.restParam != nil {
			if fn.variadicMethod != nil {
				return nil, env.Errorf(form, "fn can only have one variadic method")
			}
			fn.variadicMethod = meth
		} else {
			fn.methods = append(fn.methods, meth)
		}
	}

	// validate that all methods have different numbers of required
	// params and no more than any variadic method.
	paramCounts := make(map[int]bool)
	for _, meth := range fn.methods {
		if fn.variadicMethod != nil && len(meth.requiredParams) > len(fn.variadicMethod.requiredParams) {
			return nil, env.Errorf(form, "can't have fixed arity function with more params than variadic function")
		}
		if paramCounts[len(meth.requiredParams)] {
			return nil, env.Errorf(form, "fn cannot have multiple methods with the same number of required parameters")
		}
		paramCounts[len(meth.requiredParams)] = true
	}

	return fn, nil
}

func parseMethod(env Environment, form ISeq) (*FuncMethod, error) {
	params, ok := form.First().(*Vector)
	if !ok {
		return nil, env.Errorf(form.First(), "invalid fn expression, expected vector of parameters")
	}
	rest := form.Next()
	meth := &FuncMethod{
		body: rest,
	}

	expectRest := false
	for i := 0; i < params.Count(); i++ {
		sym, ok := MustNth(params, i).(*Symbol)
		if !ok {
			return nil, env.Errorf(MustNth(params, i), "fn params must be symbols")
		}
		if sym.Namespace() != "" {
			return nil, env.Errorf(sym, "fn params must not be qualified")
		}
		if sym.Name() == "&" {
			if expectRest {
				return nil, env.Errorf(sym, "invalid parameter list")
			}
			expectRest = true
			continue
		}
		if expectRest {
			if i != params.Count()-1 {
				return nil, env.Errorf(sym, "unexpected parameter")
			}
			meth.restParam = sym
		} else {
			meth.requiredParams = append(meth.requiredParams, sym)
		}
	}

	if len(meth.requiredParams) > MaxPositionalArity {
		return nil, env.Errorf(params, "fn cannot have more than %d required parameters", MaxPositionalArity)
	}

	return meth, nil
}

func (f *Func) String() string {
	b := strings.Builder{}
	b.WriteString("(fn*")
	if f.name != nil {
		b.WriteString(" ")
		b.WriteString(f.name.String())
	}
	b.WriteRune(' ')
	numMethods := len(f.methods)
	if f.variadicMethod != nil {
		numMethods++
	}
	for i, meth := range f.methods {
		if numMethods > 1 {
			b.WriteRune('(')
		}
		b.WriteString(meth.String())
		if numMethods > 1 {
			b.WriteRune(')')
		}
		if i < numMethods-1 {
			b.WriteRune(' ')
		}
	}
	if f.variadicMethod != nil {
		if numMethods > 1 {
			b.WriteRune('(')
		}
		b.WriteString(f.variadicMethod.String())
		if numMethods > 1 {
			b.WriteRune(')')
		}
	}
	b.WriteString(")")
	return b.String()
}

func (f *Func) Equal(v interface{}) bool {
	other, ok := v.(*Func)
	if !ok {
		return false
	}
	return f == other
}

func errorWithStack(err error, stackFrame StackFrame) error {
	if err == nil {
		return nil
	}
	valErr, ok := err.(*Error)
	if !ok {
		return NewError(stackFrame, err)
	}
	return valErr.AddStack(stackFrame)
}

func (f *Func) Apply(env Environment, args []interface{}) (interface{}, error) {
	// function name for error messages
	fnName := "<anonymous function>"
	fnEnv := f.env.PushScope()
	if f.name != nil {
		fnName = f.name.String()
		// Define the function name in the environment.
		fnEnv.BindLocal(f.name, f)
	}

	// Find the correct method
	var method *FuncMethod
	for _, m := range f.methods {
		if len(args) == len(m.requiredParams) {
			method = m
			break
		}
	}
	if method == nil && f.variadicMethod != nil {
		if len(args) >= len(f.variadicMethod.requiredParams) {
			method = f.variadicMethod
		}
	}
	if method == nil {
		return nil, errorWithStack(fmt.Errorf("wrong number of arguments (%d) passed to %s", len(args), fnName), StackFrame{
			FunctionName: fnName,
			Pos:          f.Pos(),
		})
	}

	bindingValues := args[:len(method.requiredParams)]
	var bindingRestValue interface{}
	if len(args) > len(method.requiredParams) {
		bindingRestValue = NewList(args[len(method.requiredParams):]...)
	}

Recur:
	for i := 0; i < len(method.requiredParams); i++ {
		fnEnv.BindLocal(method.requiredParams[i], bindingValues[i])
	}
	if method.restParam != nil {
		fnEnv.BindLocal(method.restParam, bindingRestValue)
	}

	var exprs []interface{}
	for cur := method.body; cur != nil; cur = cur.Next() {
		exprs = append(exprs, cur.First())
	}
	if len(exprs) == 0 {
		panic("empty function body")
	}

	for _, expr := range exprs[:len(exprs)-1] {
		_, err := fnEnv.Eval(expr)
		if err != nil {
			errPos := f.Pos()
			if expr, ok := expr.(interface{ Pos() Pos }); ok {
				errPos = expr.Pos()
			}
			return nil, errorWithStack(err, StackFrame{
				FunctionName: fnName,
				Pos:          errPos,
			})
		}
	}

	rt := NewRecurTarget()
	recurEnv := fnEnv.WithRecurTarget(rt)
	recurErr := &RecurError{Target: rt}

	lastExpr := exprs[len(exprs)-1]
	v, err := recurEnv.Eval(lastExpr)
	if errors.As(err, &recurErr) {
		if len(recurErr.Args) != method.recurParamLength() {
			// error. TODO: check this at compile time
			return nil, errorWithStack(fmt.Errorf("wrong number of arguments (%d) passed to recur", len(recurErr.Args)), StackFrame{
				FunctionName: fnName,
				Pos:          f.Pos(),
			})
		}
		bindingRestValue = nil
		bindingValues = recurErr.Args[:len(method.requiredParams)]
		if len(recurErr.Args) > len(method.requiredParams) {
			bindingRestValue = recurErr.Args[len(method.requiredParams)]
		}
		goto Recur
	}

	if err != nil {
		errPos := f.Pos()
		if expr, ok := lastExpr.(interface{ Pos() Pos }); ok {
			errPos = expr.Pos()
		}
		return nil, errorWithStack(err, StackFrame{
			FunctionName: fnName,
			Pos:          errPos,
		})
	}
	return v, nil
}

// BuiltinFunc is a builtin function.
type BuiltinFunc struct {
	Section
	Applyer
	Name     string
	variadic bool
	argNames []string
}

func (f *BuiltinFunc) String() string {
	return fmt.Sprintf("*builtin %s*", f.Name)
}

func (f *BuiltinFunc) Equal(v interface{}) bool {
	other, ok := v.(*BuiltinFunc)
	if !ok {
		return false
	}
	return f == other
}

func (f *BuiltinFunc) Apply(env Environment, args []interface{}) (interface{}, error) {
	val, err := f.Applyer.Apply(env, args)
	if err != nil {
		return nil, NewError(StackFrame{
			FunctionName: "* builtin " + f.Name + " *",
			Pos:          f.Section.Pos(),
		}, err)
	}
	return val, nil
}
