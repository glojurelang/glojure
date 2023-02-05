package reader

import "github.com/glojurelang/glojure/value"

type (
	SymbolResolver interface {
		CurrentNS() *value.Symbol
		ResolveStruct(*value.Symbol) *value.Symbol
		ResolveAlias(*value.Symbol) *value.Symbol
		ResolveVar(*value.Symbol) *value.Symbol
	}
)
