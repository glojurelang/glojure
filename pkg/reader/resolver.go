package reader

import value "github.com/glojurelang/glojure/pkg/lang"

type (
	SymbolResolver interface {
		CurrentNS() *value.Symbol
		ResolveStruct(*value.Symbol) *value.Symbol
		ResolveAlias(*value.Symbol) *value.Symbol
		ResolveVar(*value.Symbol) *value.Symbol
	}
)
