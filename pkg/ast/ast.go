package ast

import value "github.com/glojurelang/glojure/pkg/lang"

// Modeled after clojure's tools.analyzer
type (
	Node value.Associative
)

func MakeNode(op value.Keyword, form interface{}) Node {
	return value.NewMap(
		value.KWOp, op,
		value.KWForm, form,
	)
}

func Get(n Node, k interface{}) interface{} {
	if n == nil {
		return nil
	}
	return n.EntryAt(k).Val()
}

func Op(n Node) interface{} {
	return Get(n, value.KWOp)
}

func Form(n Node) interface{} {
	return Get(n, value.KWForm)
}

func RawForms(n Node) interface{} {
	return Get(n, value.KWRawForms)
}

func Children(n Node) interface{} {
	return Get(n, value.KWChildren)
}
