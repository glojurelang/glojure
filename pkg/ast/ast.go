package ast

import "github.com/glojurelang/glojure/pkg/lang"

// Modeled after clojure's tools.analyzer
type (
	Node lang.Associative

	NodeOp int32

	Node2 struct {
		// Benchmarking shows that switching on an integer op is faster
		// than type switching on a polymorphic interface.
		Op NodeOp

		Form     interface{}
		RawForms []interface{}

		Env lang.IPersistentMap

		// Options:
		// 1. inline all fields for all types
		// 2. include pointers to type-specific structs for all types, but
		// only set one.
		// 3. include a single pointer to a type-specific struct.
		//
		// Start with 3, then benchmark 1 vs 3.
		// TODO
		Sub interface{}

		IsLiteral    bool
		IsAssignable bool
	}

	LocalNode struct {
		Name       *lang.Symbol
		Local      lang.Keyword
		ArgID      int
		IsVariadic bool
	}

	VarNode struct {
		Var  *lang.Var
		Meta lang.IPersistentMap
	}

	ConstNode struct {
		Type  lang.Keyword
		Value interface{}
		Meta  *Node2
	}

	MaybeHostFormNode struct {
		Class string
		Field *lang.Symbol
	}

	MaybeClassNode struct {
		Class interface{}
	}

	VectorNode struct {
		Items []*Node2
	}

	MapNode struct {
		Keys []*Node2
		Vals []*Node2
	}

	SetNode struct {
		Items []*Node2
	}

	DoNode struct {
		Statements []*Node2
		Ret        *Node2
		IsBody     bool
	}

	LetNode struct {
		Body     *Node2
		Bindings []*Node2
		LoopID   *lang.Symbol
	}

	BindingNode struct {
		Name       *lang.Symbol
		Init       *Node2
		Local      lang.Keyword
		ArgID      int
		IsVariadic bool
	}

	InvokeNode struct {
		Meta lang.IPersistentMap
		Fn   *Node2
		Args []*Node2
	}

	IfNode struct {
		Test *Node2
		Then *Node2
		Else *Node2
	}

	NewNode struct {
		Class *Node2
		Args  []*Node2
	}

	QuoteNode struct {
		Expr *Node2
	}

	SetBangNode struct {
		Target *Node2
		Val    *Node2
	}

	TryNode struct {
		Body    *Node2
		Catches []*Node2
		Finally *Node2
	}

	CatchNode struct {
		Class *Node2
		Local *Node2
		Body  *Node2
	}

	ThrowNode struct {
		Exception *Node2
	}

	DefNode struct {
		Name *lang.Symbol
		Var  *lang.Var
		Meta *Node2
		Init *Node2
		Doc  interface{}
	}

	HostCallNode struct {
		Target *Node2
		Method *lang.Symbol
		Args   []*Node2
	}

	HostFieldNode struct {
		Target *Node2
		Field  *lang.Symbol
	}

	HostInteropNode struct {
		Target *Node2
		MOrF   *lang.Symbol
	}

	LetFnNode struct {
		Bindings []*Node2
		Body     *Node2
	}

	RecurNode struct {
		Exprs  []*Node2
		LoopID *lang.Symbol
	}

	FnNode struct {
		IsVariadic    bool
		MaxFixedArity int
		Methods       []*Node2
		Once          bool
		Local         *Node2
	}

	FnMethodNode struct {
		Params     []*Node2
		FixedArity int
		Body       *Node2
		LoopID     *lang.Symbol
		IsVariadic bool
	}

	WithMetaNode struct {
		Expr *Node2
		Meta *Node2
	}

	CaseNode struct {
		Test    *Node2
		Nodes   []*Node2
		Default *Node2
	}

	CaseNodeNode struct {
		Tests []*Node2
		Then  *Node2
	}

	TheVarNode struct {
		Var *lang.Var
	}
)

const (
	OpUnknown NodeOp = iota
	OpConst
	OpDef
	OpSetBang
	OpMaybeClass
	OpWithMeta
	OpFn
	OpFnMethod
	OpMap
	OpVector
	OpSet
	OpDo
	OpLet
	OpLetFn
	OpLoop
	OpInvoke
	OpQuote
	OpVar
	OpLocal
	OpBinding
	OpHostCall
	OpHostInterop
	OpHostField
	OpMaybeHostForm
	OpIf
	OpCase
	OpCaseNode
	OpTheVar
	OpRecur
	OpNew
	OpTry
	OpCatch
	OpThrow
)

func MakeNode(op lang.Keyword, form interface{}) Node {
	return lang.NewMap(
		lang.KWOp, op,
		lang.KWForm, form,
	)
}

func MakeNode2(op NodeOp, form interface{}) *Node2 {
	return &Node2{
		Op:   op,
		Form: form,
	}
}

func Get(n Node, k interface{}) interface{} {
	if n == nil {
		return nil
	}
	return n.EntryAt(k).Val()
}

func Op(n Node) interface{} {
	return Get(n, lang.KWOp)
}

func Form(n Node) interface{} {
	return Get(n, lang.KWForm)
}

func RawForms(n Node) interface{} {
	return Get(n, lang.KWRawForms)
}

func Children(n Node) interface{} {
	return Get(n, lang.KWChildren)
}
