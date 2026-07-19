package runtime

import (
	"errors"
	"fmt"
	"sync"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

type Fn struct {
	meta lang.IPersistentMap

	astNode *ast.Node
	env     lang.Environment
	frames  *sync.Pool

	methodsByArity map[int]*ast.Node
	variadicMethod *ast.Node
	methodRecurs   map[*ast.Node]bool
	methodEvals    map[*ast.Node]evalFn
	singleMethod   *ast.Node
	singleRecurs   bool
	singleEval     evalFn
}

type fnFrame struct {
	env      environment
	scope    scope
	recur    lang.RecurTarget
	recurErr lang.RecurError
	captured bool
}

var (
	_ lang.IObj = (*Fn)(nil)
)

func NewFn(astNode *ast.Node, env lang.Environment) *Fn {
	pool := &sync.Pool{
		New: func() interface{} {
			return &fnFrame{
				scope: scope{syms: make(map[string]interface{})},
			}
		},
	}
	fn := &Fn{
		astNode:        astNode,
		env:            env,
		frames:         pool,
		methodsByArity: make(map[int]*ast.Node),
		methodRecurs:   make(map[*ast.Node]bool),
		methodEvals:    make(map[*ast.Node]evalFn),
	}
	fnNode := astNode.Sub.(*ast.FnNode)
	for _, method := range fnNode.Methods {
		methodNode := method.Sub.(*ast.FnMethodNode)
		if methodNode.IsVariadic {
			fn.variadicMethod = method
		} else {
			fn.methodsByArity[methodNode.FixedArity] = method
		}
		if methodNode.LoopID != nil {
			fn.methodRecurs[method] = nodeRecurs(methodNode.Body, methodNode.LoopID.Name())
		}
		fn.methodEvals[method] = compileEval(methodNode.Body)
	}
	if len(fnNode.Methods) == 1 {
		fn.singleMethod = fnNode.Methods[0]
		fn.singleRecurs = fn.methodRecurs[fn.singleMethod]
		fn.singleEval = fn.methodEvals[fn.singleMethod]
	}
	return fn
}

func (fn *Fn) Meta() lang.IPersistentMap {
	return fn.meta
}

func (fn *Fn) WithMeta(meta lang.IPersistentMap) interface{} {
	cpy := *fn
	cpy.meta = meta
	return &cpy
}

func (fn *Fn) ASTNode() *ast.Node {
	return fn.astNode
}

// GetEnvironment returns the captured environment for this function.
// This is used by the codegen system to access captured values.
func (fn *Fn) GetEnvironment() lang.Environment {
	return fn.env
}

func (fn *Fn) Invoke(args ...interface{}) interface{} {
	return fn.invoke(args)
}

func (fn *Fn) Invoke0() interface{} {
	return fn.invoke(nil)
}

func (fn *Fn) Invoke1(a0 interface{}) interface{} {
	args := [1]interface{}{a0}
	return fn.invoke(args[:])
}

func (fn *Fn) Invoke2(a0, a1 interface{}) interface{} {
	args := [2]interface{}{a0, a1}
	return fn.invoke(args[:])
}

func (fn *Fn) Invoke3(a0, a1, a2 interface{}) interface{} {
	args := [3]interface{}{a0, a1, a2}
	return fn.invoke(args[:])
}

func (fn *Fn) Invoke4(a0, a1, a2, a3 interface{}) interface{} {
	args := [4]interface{}{a0, a1, a2, a3}
	return fn.invoke(args[:])
}

func (fn *Fn) invoke(args []interface{}) interface{} {
	fnNode := fn.astNode.Sub.(*ast.FnNode)

	variadic := fnNode.IsVariadic
	maxArity := fnNode.MaxFixedArity

	if !variadic && len(args) > maxArity {
		panic(lang.NewIllegalArgumentError(fmt.Sprintf("too many arguments (%d)", len(args))))
	}

	method, err := fn.findMethod(args)
	if err != nil {
		panic(err)
	}

	baseEnv, ok := fn.env.(*environment)
	if !ok {
		panic(fmt.Errorf("unsupported function environment %T", fn.env))
	}
	frame := fn.frames.Get().(*fnFrame)
	frame.captured = false
	frame.env = *baseEnv
	frame.env.scope = &frame.scope
	frame.env.fnFrame = frame
	frame.scope.parent = baseEnv.scope
	clear(frame.scope.syms)
	fnEnv := &frame.env
	defer func() {
		if !frame.captured {
			frame.scope.parent = nil
			frame.env.scope = nil
			frame.env.fnFrame = nil
			fn.frames.Put(frame)
		}
	}()

	if fnNode.Local != nil {
		localNode := fnNode.Local.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(localNode.Name, fn)
	}

	methodNode := method.Sub.(*ast.FnMethodNode)

	fixedArity := methodNode.FixedArity
	methodVariadic := methodNode.IsVariadic
	body := methodNode.Body

	bindingValues := args[:fixedArity]

	arity := fixedArity
	var bindingRestValue interface{}
	if len(args) > len(bindingValues) {
		arity++
		bindingRestValue = lang.NewList(args[len(bindingValues):]...)
	}

Recur:

	params := methodNode.Params
	for i, paramValue := range bindingValues {
		param := params[i]
		paramNode := param.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(paramNode.Name, paramValue)
	}
	if bindingRestValue != nil {
		param := params[len(params)-1]
		paramNode := param.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(paramNode.Name, bindingRestValue)
	} else if methodVariadic {
		param := params[len(params)-1]
		paramNode := param.Sub.(*ast.BindingNode)
		fnEnv.BindLocal(paramNode.Name, nil)
	}

	recurEnv := fnEnv
	var rt interface{}
	methodRecurs := fn.singleMethod == method && fn.singleRecurs ||
		fn.singleMethod != method && fn.methodRecurs[method]
	if methodRecurs {
		rt = &frame.recur
		recurEnv.recurTarget = rt
		frame.recurErr.Target = rt
		recurEnv.recurErr = &frame.recurErr
	} else {
		recurEnv.recurTarget = nil
		recurEnv.recurErr = nil
	}
	evaluator := fn.methodEvals[method]
	if fn.singleMethod == method {
		evaluator = fn.singleEval
	}
	var res interface{}
	err = nil
	if evaluator != nil {
		res, err = evaluator(recurEnv)
	} else {
		res, err = recurEnv.EvalAST(body)
	}
	if err != nil {
		recurErr, isRecur := asRecurError(err)
		if isRecur && recurErr.Target == rt {
			if len(recurErr.Args) != arity {
				panic("wrong number of arguments to recur")
			}
			bindingRestValue = nil
			bindingValues = recurErr.Args[:fixedArity]
			if len(recurErr.Args) > fixedArity {
				bindingRestValue = recurErr.Args[fixedArity]
			}
			goto Recur
		}
		panic(errorWithStack(err, lang.StackFrame{})) // TODO: think through error stacks
	}
	return res
}

func (fn *Fn) findMethod(args []interface{}) (*ast.Node, error) {
	if fn.singleMethod != nil {
		methodNode := fn.singleMethod.Sub.(*ast.FnMethodNode)
		if len(args) == methodNode.FixedArity ||
			methodNode.IsVariadic && len(args) >= methodNode.FixedArity {
			return fn.singleMethod, nil
		}
		return nil, lang.NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments (%d)", len(args)))
	}
	if method := fn.methodsByArity[len(args)]; method != nil {
		return method, nil
	}
	if fn.variadicMethod == nil || len(args) < fn.variadicMethod.Sub.(*ast.FnMethodNode).FixedArity {
		return nil, lang.NewIllegalArgumentError(fmt.Sprintf("wrong number of arguments (%d)", len(args)))
	}
	return fn.variadicMethod, nil
}

func asRecurError(err error) (*lang.RecurError, bool) {
	if recurErr, ok := err.(*lang.RecurError); ok {
		return recurErr, true
	}
	var recurErr *lang.RecurError
	if errors.As(err, &recurErr) {
		return recurErr, true
	}
	return nil, false
}

func (fn *Fn) ApplyTo(args lang.ISeq) interface{} {
	return fn.Invoke(seqToSlice(args)...)
}

func errorWithStack(err error, stackFrame lang.StackFrame) error {
	if err == nil {
		return nil
	}
	valErr, ok := err.(*lang.EvalError)
	if !ok {
		return lang.NewEvalError(stackFrame, err)
	}
	return valErr.AddStack(stackFrame)
}
