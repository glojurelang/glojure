package reader

import "github.com/glojurelang/glojure/pkg/lang"

type (
	SymbolResolver interface {
		CurrentNS() *lang.Symbol
		ResolveStruct(*lang.Symbol) *lang.Symbol
		ResolveAlias(*lang.Symbol) *lang.Symbol
		ResolveVar(*lang.Symbol) *lang.Symbol
	}
)
