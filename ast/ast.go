package ast

import "github.com/glojurelang/glojure/value"

// Modeled after clojure's tools.analyzer
type (
	Node value.Associative

	// Node struct {
	// 	// The node op.
	// 	Op interface{}

	// 	// The glojure form from which the node originated.
	// 	Form interface{}

	// 	// If this node's Form has been macroexpanded, a sequence of all the
	// 	// intermediate forms from the original form to the macroexpanded
	// 	// form.
	// 	RawForms []interface{}

	// 	Literal bool

	// 	Children map[string]interface{}
	// }

	OpBinding struct{}
	OpCatch   struct{}

	OpConst struct {
		// one of one of :nil, :bool, :keyword, :symbol, :string, :number,
		// :type, :record, :map, :vector, :set, :seq, :char, :regex,
		// :class, :var, or :unknown
		Type string
		Val  interface{}
	}

	OpDef         struct{}
	OpDo          struct{}
	OpFn          struct{}
	OpFnMethod    struct{}
	OpHostCall    struct{}
	OpHostField   struct{}
	OpHostInterop struct{}
	OpIf          struct{}
	OpInvoke      struct{}
	OpLetFn       struct{}
	OpLocal       struct{}
	OpLoop        struct{}

	OpMap struct {
		Keys []*Node
		Vals []*Node
	}

	// OpMaybeClass
	// OpMaybeHostForm
	OpNew   struct{}
	OpQuote struct{}
	OpRecur struct{}

	OpSet struct {
		Items []*Node
	}

	OpSetBang struct{}
	OpThrow   struct{}
	OpTry     struct{}
	OpVar     struct{}

	OpVector struct {
		Items []*Node
	}

	OpWithMeta struct{}
)

func MakeNode(op value.Keyword, form interface{}) Node {
	return value.NewMap(
		kw("op"), op,
		kw("form"), form,
	)
}

func Op(n Node) interface{} {
	return n.EntryAt(kw("op"))
}

func Form(n Node) interface{} {
	return n.EntryAt(kw("form"))
}

func RawForms(n Node) interface{} {
	return n.EntryAt(kw("raw-forms"))
}

func Children(n Node) interface{} {
	return n.EntryAt(kw("children"))
}

func kw(s string) value.Keyword {
	return value.NewKeyword(s)
}
