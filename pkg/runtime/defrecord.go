package runtime

import (
	"fmt"

	"github.com/glojurelang/glojure/pkg/lang"
)

func installDefrecord(core *lang.Namespace) {
	defrecord := lang.InternVar(
		core,
		lang.NewSymbol("defrecord"),
		lang.NewFnFunc(defrecordMacro),
		true,
	)
	defrecord.SetMacro()
	lang.InternVar(
		core,
		lang.NewSymbol("record?"),
		lang.NewFnFunc1(func(value any) any {
			_, ok := value.(lang.IRecord)
			return ok
		}),
		true,
	)
}

// defrecordMacro establishes a first-class descriptor and constructor roots.
// The descriptor remains visible in the analyzed forms so AOT can lower the
// same definition to a concrete Go type rather than rediscovering map shapes.
func defrecordMacro(args ...any) any {
	if len(args) < 4 {
		panic(lang.NewIllegalArgumentError(
			"defrecord expects a name and field vector",
		))
	}
	if len(args) != 4 {
		panic(lang.NewUnsupportedOperationError(
			"defrecord protocol implementations are not yet supported",
		))
	}
	name, ok := args[2].(*lang.Symbol)
	if !ok || name.Namespace() != "" {
		panic(lang.NewIllegalArgumentError(
			"defrecord name must be an unqualified symbol",
		))
	}
	fields, ok := args[3].(lang.IPersistentVector)
	if !ok {
		panic(lang.NewIllegalArgumentError(
			"defrecord fields must be a vector",
		))
	}
	fieldNames := make([]string, fields.Count())
	reserved := map[string]bool{
		"__meta":   true,
		"__extmap": true,
		"__hash":   true,
		"__hasheq": true,
	}
	seen := make(map[string]bool, fields.Count())
	for i := range fieldNames {
		field, ok := lang.MustNth(fields, i).(*lang.Symbol)
		if !ok || field.Namespace() != "" {
			panic(lang.NewIllegalArgumentError(fmt.Sprintf(
				"defrecord field %d must be an unqualified symbol",
				i,
			)))
		}
		fieldName := field.Name()
		if reserved[fieldName] {
			panic(lang.NewIllegalArgumentError(
				"reserved defrecord field: " + fieldName,
			))
		}
		if seen[fieldName] {
			panic(lang.NewIllegalArgumentError(
				"duplicate defrecord field: " + fieldName,
			))
		}
		seen[fieldName] = true
		fieldNames[i] = fieldName
	}

	namespace := lang.VarCurrentNS.Deref().(*lang.Namespace)
	recordType := lang.InternRecordType(
		namespace.Name().String(),
		name.Name(),
		fieldNames...,
	)
	positional := lang.NewRecordConstructor(recordType, false)
	fromMap := lang.NewRecordConstructor(recordType, true)

	def := lang.NewSymbol("def")
	return lang.NewList(
		lang.NewSymbol("do"),
		lang.NewList(def, lang.NewSymbol(name.Name()), recordType),
		lang.NewList(
			def,
			lang.NewSymbol("->"+name.Name()),
			positional,
		),
		lang.NewList(
			def,
			lang.NewSymbol("map->"+name.Name()),
			fromMap,
		),
		recordType,
	)
}
