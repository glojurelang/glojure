package glj

import "github.com/glojurelang/glojure/value"

// Var returns an IFn associated with the namespace and name.
func Var(ns, name interface{}) value.IFn {
	return value.InternVarName(asSym(ns), asSym(name))
}

func asSym(x interface{}) *value.Symbol {
	if str, ok := x.(string); ok {
		return value.NewSymbol(str)
	}
	return x.(*value.Symbol)
}
