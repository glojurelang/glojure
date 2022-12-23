package ast

import "github.com/glojurelang/glojure/value"

// Modeled after clojure's tools.analyzer
type (
	Node value.Associative
)

func MakeNode(op value.Keyword, form interface{}) Node {
	return value.NewMap(
		kw("op"), op,
		kw("form"), form,
	)
}

func Get(n Node, k interface{}) interface{} {
	if n == nil {
		return nil
	}
	return n.EntryAt(k).Val()
}

func Op(n Node) interface{} {
	return Get(n, kw("op"))
}

func Form(n Node) interface{} {
	return Get(n, kw("form"))
}

func RawForms(n Node) interface{} {
	return Get(n, kw("raw-forms"))
}

func Children(n Node) interface{} {
	return Get(n, kw("children"))
}

func kw(s string) value.Keyword {
	return value.NewKeyword(s)
}
