package value

import (
	"fmt"
	"strings"
)

// Func is a function.
type Func struct {
	Section
	LambdaName string
	Env        Environment
	Arities    []FuncArity
}

type FuncArity struct {
	BindingForm *Vector
	Exprs       *List
}

func (f *Func) String() string {
	b := strings.Builder{}
	b.WriteString("(fn")
	if f.LambdaName != "" {
		b.WriteString(" ")
		b.WriteString(f.LambdaName)
	}
	b.WriteRune(' ')
	for i, arity := range f.Arities {
		if len(f.Arities) > 1 {
			b.WriteRune('(')
		}
		b.WriteString(arity.BindingForm.String())
		b.WriteRune(' ')
		for cur := arity.Exprs; !cur.IsEmpty(); cur = cur.Next() {
			if cur != arity.Exprs {
				b.WriteString(" ")
			}
			b.WriteString(ToString(cur.Item()))
		}
		if len(f.Arities) > 1 {
			b.WriteRune(')')
		}
		if i < len(f.Arities)-1 {
			b.WriteRune(' ')
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
	var res interface{}
	var err error
	continuation := func() (interface{}, Continuation, error) {
		return f.ContinuationApply(env, args)
	}
	for {
		res, continuation, err = continuation()
		if err != nil {
			return nil, err
		}
		if continuation == nil {
			return res, nil
		}
	}
}

func (f *Func) ContinuationApply(env Environment, args []interface{}) (interface{}, Continuation, error) {
	// function name for error messages
	fnName := f.LambdaName
	if fnName == "" {
		fnName = "<anonymous function>"
	}

	fnEnv := f.Env.PushScope()
	if f.LambdaName != "" {
		// Define the function name in the environment.
		fnEnv.Define(f.LambdaName, f)
	}

	var bindings []interface{}
	var err error
	var exprList *List

	// Find the correct arity.
	for _, arity := range f.Arities {
		minArg, maxArg := MinMaxArgumentCount(arity.BindingForm)
		if len(args) < minArg || (len(args) > maxArg && maxArg != -1) {
			err = fmt.Errorf("Wrong number of args (%d) for %s", len(args), fnName)
			continue
		}

		bindings, err = Bind(arity.BindingForm, NewList(args))
		if err == nil {
			exprList = arity.Exprs
			break
		}
	}
	if err != nil {
		return nil, nil, errorWithStack(err, StackFrame{
			FunctionName: fnName,
			Pos:          f.Pos(),
		})
	}

	for i := 0; i < len(bindings); i += 2 {
		sym := bindings[i].(*Symbol)
		fnEnv.Define(sym.Value, bindings[i+1])
	}

	var exprs []interface{}
	for cur := exprList; !cur.IsEmpty(); cur = cur.next {
		exprs = append(exprs, cur.item)
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
			return nil, nil, errorWithStack(err, StackFrame{
				FunctionName: fnName,
				Pos:          errPos,
			})
		}
	}
	// return the last expression as a continuation
	lastExpr := exprs[len(exprs)-1]
	return nil, func() (interface{}, Continuation, error) {
		v, c, err := fnEnv.ContinuationEval(lastExpr)
		if err != nil {
			errPos := f.Pos()
			if expr, ok := lastExpr.(interface{ Pos() Pos }); ok {
				errPos = expr.Pos()
			}
			return nil, nil, errorWithStack(err, StackFrame{
				FunctionName: fnName,
				Pos:          errPos,
			})
		}
		return v, c, nil
	}, nil
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
