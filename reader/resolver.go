package reader

type (
	SymbolResolver interface {
		ResolveSymbol(name string) string
	}

	SymbolResolverFunc func(name string) string
)

func (f SymbolResolverFunc) ResolveSymbol(name string) string {
	return f(name)
}

var defaultSymbolResolver = SymbolResolverFunc(func(name string) string {
	return name
})
