package compiler

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

var foldableNumberMethods = map[string]bool{
	"add":                true,
	"addp":               true,
	"and":                true,
	"andnot":             true,
	"clearbit":           true,
	"compare":            true,
	"dec":                true,
	"decp":               true,
	"divide":             true,
	"equiv":              true,
	"flipbit":            true,
	"gt":                 true,
	"gte":                true,
	"inc":                true,
	"incp":               true,
	"isneg":              true,
	"ispos":              true,
	"iszero":             true,
	"lt":                 true,
	"lte":                true,
	"max":                true,
	"min":                true,
	"minus":              true,
	"minusp":             true,
	"multiply":           true,
	"multiplyp":          true,
	"not":                true,
	"or":                 true,
	"quotient":           true,
	"remainder":          true,
	"rationalize":        true,
	"setbit":             true,
	"shiftleft":          true,
	"shiftright":         true,
	"testbit":            true,
	"unchecked_inc":      true,
	"unchecked_minus":    true,
	"unchecked_multiply": true,
	"unchecked_negate":   true,
	"uncheckedadd":       true,
	"uncheckeddec":       true,
	"unsignedshiftright": true,
	"xor":                true,
}

// foldLiteralNumberCall evaluates a resolved, pure Numbers call only when all
// arguments are literals. Calls that trap are left intact so compilation does
// not change when the program observes the error.
func foldLiteralNumberCall(call *ast.HostCallNode) (value interface{}, ok bool) {
	if call.Target.Op != ast.OpConst ||
		call.Target.Sub.(*ast.ConstNode).Value != lang.Numbers ||
		call.ResolvedMethod == nil ||
		!foldableNumberMethods[strings.ToLower(call.Method.Name())] {
		return nil, false
	}

	args := make([]interface{}, len(call.Args))
	for i, arg := range call.Args {
		if arg.Op != ast.OpConst {
			return nil, false
		}
		args[i] = arg.Sub.(*ast.ConstNode).Value
	}

	defer func() {
		if recover() != nil {
			value, ok = nil, false
		}
	}()
	return lang.Apply(call.ResolvedMethod, args), true
}
