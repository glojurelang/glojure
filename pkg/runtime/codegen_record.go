//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (g *Generator) prepareAOTRecordTypes(vars []namedVar) {
	unique := make(map[*lang.RecordType]struct{})
	for _, named := range vars {
		if !named.vr.IsBound() {
			continue
		}
		switch value := codegenVarValue(named.vr).(type) {
		case *lang.RecordType:
			unique[value] = struct{}{}
		case *lang.RecordConstructor:
			unique[value.RecordType()] = struct{}{}
		}
	}
	types := make([]*lang.RecordType, 0, len(unique))
	for recordType := range unique {
		types = append(types, recordType)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].FullName() < types[j].FullName()
	})
	for _, recordType := range types {
		g.allocAOTRecordType(recordType)
	}
}

func (g *Generator) allocAOTRecordType(
	descriptor *lang.RecordType,
) *aotRecordType {
	if record := g.aotRecordTypes[descriptor]; record != nil {
		return record
	}
	index := len(g.aotRecordTypes)
	record := &aotRecordType{
		index:        index,
		descriptor:   descriptor,
		descriptorGo: fmt.Sprintf("aotRecordType%d", index),
		typeName: fmt.Sprintf(
			"aotRecord%d%s",
			index,
			mungeID(descriptor.Name()),
		),
		constructor: fmt.Sprintf("aotRecordNew%d", index),
		mapFactory:  fmt.Sprintf("aotRecordFromMap%d", index),
		fieldNames:  descriptor.FieldNames(),
		shape:       g.aotRecordShapes[descriptor],
		stateName:   fmt.Sprintf("aotRecordState%d", index),
		fastCtor:    fmt.Sprintf("aotRecordFastNew%d", index),
	}
	g.aotRecordTypes[descriptor] = record
	g.generateAOTRecordType(record)
	return record
}

func (g *Generator) sortedAOTRecordTypes() []*aotRecordType {
	records := make([]*aotRecordType, 0, len(g.aotRecordTypes))
	for _, record := range g.aotRecordTypes {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].index < records[j].index
	})
	return records
}

func (g *Generator) generateAOTRecordType(record *aotRecordType) {
	fieldNames := make([]string, len(record.fieldNames))
	for i, field := range record.fieldNames {
		fieldNames[i] = fmt.Sprintf("%q", field)
	}
	fmt.Fprintf(
		&g.aotDeclarations,
		"var %s = lang.InternRecordType(%q, %q, %s)\n",
		record.descriptorGo,
		record.descriptor.Namespace(),
		record.descriptor.Name(),
		strings.Join(fieldNames, ", "),
	)
	if record.shape != nil {
		g.generateAOTSpecializedRecordType(record)
		g.generateAOTRecordCollectionMethods(record)
		return
	}
	fmt.Fprintf(&g.aotDeclarations, "type %s struct {\n", record.typeName)
	fmt.Fprintln(&g.aotDeclarations, "\tlang.RecordMarker")
	fmt.Fprintln(&g.aotDeclarations, "\tattrs *lang.RecordAttrs")
	for i := range record.fieldNames {
		fmt.Fprintf(&g.aotDeclarations, "\tf%d any\n", i)
	}
	fmt.Fprintln(&g.aotDeclarations, "}")

	params := make([]string, len(record.fieldNames))
	fields := make([]string, len(record.fieldNames))
	for i := range record.fieldNames {
		params[i] = fmt.Sprintf("p%d any", i)
		fields[i] = fmt.Sprintf("f%d: p%d", i, i)
	}
	fmt.Fprintf(
		&g.aotDeclarations,
		"func %s(%s) *%s {\n\treturn &%s{%s}\n}\n",
		record.constructor,
		strings.Join(params, ", "),
		record.typeName,
		record.typeName,
		strings.Join(fields, ", "),
	)

	fmt.Fprintf(
		&g.aotDeclarations,
		"func %s(value any) *%s {\n"+
			"\tsource := lang.NewRecordFromMap(%s, value)\n"+
			"\treturn &%s{\n"+
			"\t\tattrs: lang.NewRecordAttrs("+
			"source.RecordMeta(), source.RecordExtMap()),\n",
		record.mapFactory,
		record.typeName,
		record.descriptorGo,
		record.typeName,
	)
	for i := range record.fieldNames {
		fmt.Fprintf(&g.aotDeclarations,
			"\t\tf%d: source.RecordField(%d),\n", i, i)
	}
	fmt.Fprintln(&g.aotDeclarations, "\t}\n}")

	g.generateAOTRecordValueMethods(record)
	g.generateAOTRecordCollectionMethods(record)
}

func (g *Generator) generateAOTRecordValueMethods(record *aotRecordType) {
	typeName := record.typeName
	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) RecordType() *lang.RecordType { return %s }\n"+
			"func (r *%s) RecordExtMap() lang.IPersistentMap {\n"+
			"\treturn lang.RecordAttrsExt(r.attrs)\n}\n"+
			"func (r *%s) RecordMeta() lang.IPersistentMap {\n"+
			"\treturn lang.RecordAttrsMeta(r.attrs)\n}\n"+
			"func (r *%s) Meta() lang.IPersistentMap { return r.RecordMeta() }\n",
		typeName, record.descriptorGo,
		typeName,
		typeName,
		typeName,
	)
	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) RecordField(index int) any {\n\tswitch index {\n",
		typeName)
	for i := range record.fieldNames {
		fmt.Fprintf(&g.aotDeclarations, "\tcase %d: return r.f%d\n", i, i)
	}
	fmt.Fprintln(&g.aotDeclarations,
		"\tdefault: panic(lang.NewIllegalArgumentError(\"record field index out of bounds\"))\n\t}\n}")

	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) RecordWithField(index int, value any) lang.RecordValue {\n"+
			"\tif lang.Identical(r.RecordField(index), value) { return r }\n"+
			"\tresult := *r\n"+
			"\tswitch index {\n",
		typeName)
	for i := range record.fieldNames {
		fmt.Fprintf(&g.aotDeclarations,
			"\tcase %d: result.f%d = value\n", i, i)
	}
	fmt.Fprintln(&g.aotDeclarations,
		"\tdefault: panic(lang.NewIllegalArgumentError(\"record field index out of bounds\"))\n"+
			"\t}\n\tresult.attrs = lang.RecordAttrsWithoutHash(r.attrs)\n"+
			"\treturn &result\n}")

	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) RecordWithExtMap(ext lang.IPersistentMap) lang.RecordValue {\n"+
			"\tif r.RecordExtMap() == ext { return r }\n"+
			"\tresult := *r\n"+
			"\tresult.attrs = lang.RecordAttrsWithExt(r.attrs, ext)\n"+
			"\treturn &result\n}\n"+
			"func (r *%s) RecordWithMeta(meta lang.IPersistentMap) lang.RecordValue {\n"+
			"\tif r.RecordMeta() == meta { return r }\n"+
			"\tresult := *r\n"+
			"\tresult.attrs = lang.RecordAttrsWithMeta(r.attrs, meta)\n"+
			"\treturn &result\n}\n",
		typeName,
		typeName,
	)
}

func (g *Generator) generateAOTRecordCollectionMethods(
	record *aotRecordType,
) {
	typeName := record.typeName
	hashPointer := "&r.attrs"
	if record.shape != nil {
		hashPointer = "r.aotRecordAttrsPtr()"
	}
	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) ValAt(key any) any { return lang.RecordValAt(r, key) }\n"+
			"func (r *%s) ValAtDefault(key, fallback any) any {\n"+
			"\treturn lang.RecordValAtDefault(r, key, fallback)\n}\n"+
			"func (r *%s) ContainsKey(key any) bool {\n"+
			"\treturn lang.RecordContainsKey(r, key)\n}\n"+
			"func (r *%s) EntryAt(key any) lang.IMapEntry {\n"+
			"\treturn lang.RecordEntryAt(r, key)\n}\n"+
			"func (r *%s) Assoc(key, value any) lang.Associative {\n"+
			"\treturn lang.RecordAssoc(r, key, value)\n}\n"+
			"func (r *%s) AssocEx(key, value any) lang.IPersistentMap {\n"+
			"\treturn lang.RecordAssocEx(r, key, value)\n}\n"+
			"func (r *%s) Without(key any) lang.IPersistentMap {\n"+
			"\treturn lang.RecordWithout(r, key)\n}\n"+
			"func (r *%s) Count() int { return lang.RecordCount(r) }\n"+
			"func (r *%s) Seq() lang.ISeq { return lang.RecordSeq(r) }\n"+
			"func (r *%s) Empty() lang.IPersistentCollection {\n"+
			"\treturn lang.RecordEmpty(r)\n}\n"+
			"func (r *%s) Cons(value any) lang.Conser {\n"+
			"\treturn lang.RecordCons(r, value)\n}\n"+
			"func (r *%s) Equiv(other any) bool {\n"+
			"\treturn lang.RecordEquiv(r, other)\n}\n",
		typeName, typeName, typeName, typeName,
		typeName, typeName, typeName, typeName,
		typeName, typeName, typeName, typeName,
	)
	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) WithMeta(meta lang.IPersistentMap) any {\n"+
			"\treturn r.RecordWithMeta(meta)\n}\n"+
			"func (r *%s) Invoke(args ...any) any {\n"+
			"\treturn lang.RecordInvoke(r, args...)\n}\n"+
			"func (r *%s) ApplyTo(args lang.ISeq) any {\n"+
			"\treturn lang.RecordApplyTo(r, args)\n}\n"+
			"func (r *%s) Hash() uint32 {\n"+
			"\treturn lang.RecordAttrsHash(%s, r)\n}\n"+
			"func (r *%s) HashEq() uint32 {\n"+
			"\treturn lang.RecordAttrsHashEq(%s, r)\n}\n"+
			"func (r *%s) String() string { return lang.RecordString(r) }\n",
		typeName, typeName, typeName, typeName, hashPointer,
		typeName, hashPointer, typeName,
	)
}

func (g *Generator) generateRecordConstructorValue(
	constructor *lang.RecordConstructor,
) string {
	record := g.allocAOTRecordType(constructor.RecordType())
	if constructor.FromMap() {
		return fmt.Sprintf(
			"lang.FnFunc1(func(p0 any) any { return %s(p0) })",
			record.mapFactory,
		)
	}
	arity := len(record.fieldNames)
	if arity > 20 {
		return fmt.Sprintf(
			"lang.NewRecordConstructor(%s, false)",
			record.descriptorGo,
		)
	}
	params := make([]string, arity)
	args := make([]string, arity)
	for i := range params {
		params[i] = fmt.Sprintf("p%d any", i)
		args[i] = fmt.Sprintf("p%d", i)
	}
	return fmt.Sprintf(
		"lang.FnFunc%d(func(%s) any { return %s(%s) })",
		arity,
		strings.Join(params, ", "),
		record.constructor,
		strings.Join(args, ", "),
	)
}

func (g *Generator) aotRecordInvokeTarget(
	invoke *ast.InvokeNode,
) *aotRecordCallTarget {
	if !g.directLink || invoke.Fn.Op != ast.OpVar {
		return nil
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if !vr.IsBound() || vr.IsMacro() || vr.IsDynamic() ||
		RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef)) {
		return nil
	}
	constructor, ok := codegenVarValue(vr).(*lang.RecordConstructor)
	if !ok {
		return nil
	}
	arity := constructor.RecordType().FieldCount()
	if constructor.FromMap() {
		arity = 1
	}
	if len(invoke.Args) != arity {
		return nil
	}
	return &aotRecordCallTarget{
		record:  g.allocAOTRecordType(constructor.RecordType()),
		fromMap: constructor.FromMap(),
	}
}
