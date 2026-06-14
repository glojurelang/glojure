package glj

import "github.com/glojurelang/glojure/pkg/lang"

// Var returns an IFn associated with the namespace and name.
func Var(ns, name interface{}) lang.IFn {
	return lang.InternVarName(asSym(ns), asSym(name))
}

func asSym(x interface{}) *lang.Symbol {
	if str, ok := x.(string); ok {
		return lang.NewSymbol(str)
	}
	return x.(*lang.Symbol)
}
