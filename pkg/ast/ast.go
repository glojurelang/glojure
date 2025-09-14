package ast

import "github.com/glojurelang/glojure/pkg/lang"

type (
	NodeOp int32

	Node struct {
		// Benchmarking shows that switching on an integer op is faster
		// than type switching on a polymorphic interface or calling a
		// polymorphic method.
		Op NodeOp

		Form     interface{}
		RawForms []interface{}

		Env lang.IPersistentMap

		// Sub is a pointer to an Op-specific struct.
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
		Meta  *Node
	}

	GoBuiltinNode struct {
		Sym   *lang.Symbol
		Value interface{}
	}

	GoNode struct {
		Invoke *Node
	}

	MaybeHostFormNode struct {
		Class string
		Field *lang.Symbol
	}

	MaybeClassNode struct {
		Class interface{}
	}

	VectorNode struct {
		Items []*Node
	}

	MapNode struct {
		Keys []*Node
		Vals []*Node
	}

	SetNode struct {
		Items []*Node
	}

	DoNode struct {
		Statements []*Node
		Ret        *Node
		IsBody     bool
	}

	LetNode struct {
		Body     *Node
		Bindings []*Node
		LoopID   *lang.Symbol
	}

	BindingNode struct {
		Name       *lang.Symbol
		Init       *Node
		Local      lang.Keyword
		ArgID      int
		IsVariadic bool
	}

	InvokeNode struct {
		Meta lang.IPersistentMap
		Fn   *Node
		Args []*Node
	}

	IfNode struct {
		Test *Node
		Then *Node
		Else *Node
	}

	NewNode struct {
		Class *Node
		Args  []*Node
	}

	QuoteNode struct {
		Expr *Node
	}

	SetBangNode struct {
		Target *Node
		Val    *Node
	}

	TryNode struct {
		Body    *Node
		Catches []*Node
		Finally *Node
	}

	CatchNode struct {
		Class *Node
		Local *Node
		Body  *Node
	}

	ThrowNode struct {
		Exception *Node
	}

	DefNode struct {
		Name *lang.Symbol
		Var  *lang.Var
		Meta *Node
		Init *Node
		Doc  interface{}
	}

	HostCallNode struct {
		Target *Node
		Method *lang.Symbol
		Args   []*Node
	}

	HostFieldNode struct {
		Target *Node
		Field  *lang.Symbol
	}

	HostInteropNode struct {
		Target *Node
		MOrF   *lang.Symbol
	}

	LetFnNode struct {
		Bindings []*Node
		Body     *Node
	}

	RecurNode struct {
		Exprs  []*Node
		LoopID *lang.Symbol
	}

	FnNode struct {
		IsVariadic    bool
		MaxFixedArity int
		Methods       []*Node
		Once          bool
		Local         *Node
	}

	FnMethodNode struct {
		Params     []*Node
		FixedArity int
		Body       *Node
		LoopID     *lang.Symbol
		IsVariadic bool
	}

	WithMetaNode struct {
		Expr *Node
		Meta *Node
	}

	CaseNode struct {
		Test       *Node            // The expression to test
		Shift      int64            // Bit shift for hash compaction
		Mask       int64            // Bit mask for hash compaction
		TestType   interface{}      // Keyword: :int, :hash-identity, or :hash-equiv
		SwitchType interface{}      // Keyword: :compact or :sparse
		Default    *Node            // Default expression
		Entries    []CaseEntry      // Case entries
		SkipCheck  map[int64]bool   // Set of keys with collisions
	}

	CaseEntry struct {
		Key          int64       // Map key (int value or shifted/masked hash)
		TestConstant *Node       // Original test constant (nil for collisions)
		ResultExpr   *Node       // Result expression or condp for collisions
		HasCollision bool        // Whether this is a collision case
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
	OpGoBuiltin
	OpGo
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

func MakeNode(op NodeOp, form interface{}) *Node {
	return &Node{
		Op:   op,
		Form: form,
	}
}
