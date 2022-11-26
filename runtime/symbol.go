package runtime

import "github.com/glojurelang/glojure/value"

// TODO: this stuff isn't really used. clean it up.

type Location struct {
}

type Symbol struct {
	Name string
	Help string
	// where the symbol is defined
	// if nil, it is a builtin
	DefLocation *Location
	Value       value.Value
}

type Package struct {
	Name    string
	Symbols []*Symbol
}
