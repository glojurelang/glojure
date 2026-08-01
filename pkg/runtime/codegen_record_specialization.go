//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/compiler"
	"github.com/glojurelang/glojure/pkg/lang"
)

func (g *Generator) generateAOTSpecializedRecordType(
	record *aotRecordType,
) {
	typeName := record.typeName
	stateName := record.stateName
	fmt.Fprintf(&g.aotDeclarations,
		"type %s struct {\n"+
			"\tattrs *lang.RecordAttrs\n"+
			"\tfields []any\n"+
			"}\n",
		stateName,
	)
	fmt.Fprintf(&g.aotDeclarations,
		"type %s struct {\n\tlang.RecordMarker\n\tstate *%s\n",
		typeName,
		stateName,
	)
	for index, field := range record.shape.Fields {
		fmt.Fprintf(&g.aotDeclarations, "\tf%d %s\n",
			index, recordAOTTypeExpr(g, field))
	}
	fmt.Fprintln(&g.aotDeclarations, "}")

	g.generateAOTSpecializedRecordFastConstructor(record)
	g.generateAOTSpecializedRecordGenericConstructor(record)
	g.generateAOTSpecializedRecordMapFactory(record)
	g.generateAOTSpecializedRecordValueMethods(record)
}

func (g *Generator) generateAOTSpecializedRecordFastConstructor(
	record *aotRecordType,
) {
	params := make([]string, len(record.shape.Fields))
	fields := make([]string, len(record.shape.Fields))
	for index, field := range record.shape.Fields {
		params[index] = fmt.Sprintf(
			"p%d %s",
			index,
			recordAOTTypeExpr(g, field),
		)
		fields[index] = fmt.Sprintf("f%d: p%d", index, index)
	}
	fmt.Fprintf(&g.aotDeclarations,
		"func %s(%s) *%s {\n"+
			"\treturn &%s{%s}\n"+
			"}\n",
		record.fastCtor,
		strings.Join(params, ", "),
		record.typeName,
		record.typeName,
		strings.Join(fields, ", "),
	)
}

func (g *Generator) generateAOTSpecializedRecordGenericConstructor(
	record *aotRecordType,
) {
	params := make([]string, len(record.fieldNames))
	args := make([]string, len(record.fieldNames))
	for index := range params {
		params[index] = fmt.Sprintf("p%d any", index)
		args[index] = fmt.Sprintf("p%d", index)
	}
	fmt.Fprintf(&g.aotDeclarations,
		"func %s(%s) *%s {\n"+
			"\treturn &%s{state: &%s{fields: []any{%s}}}\n"+
			"}\n",
		record.constructor,
		strings.Join(params, ", "),
		record.typeName,
		record.typeName,
		record.stateName,
		strings.Join(args, ", "),
	)
}

func (g *Generator) generateAOTSpecializedRecordMapFactory(
	record *aotRecordType,
) {
	fmt.Fprintf(&g.aotDeclarations,
		"func %s(value any) *%s {\n"+
			"\tsource := lang.NewRecordFromMap(%s, value)\n"+
			"\tresult := %s(",
		record.mapFactory,
		record.typeName,
		record.descriptorGo,
		record.constructor,
	)
	for index := range record.fieldNames {
		if index > 0 {
			fmt.Fprint(&g.aotDeclarations, ", ")
		}
		fmt.Fprintf(&g.aotDeclarations, "source.RecordField(%d)", index)
	}
	fmt.Fprintf(&g.aotDeclarations,
		")\n"+
			"\tresult.state.attrs = lang.NewRecordAttrs("+
			"source.RecordMeta(), source.RecordExtMap())\n"+
			"\treturn result\n"+
			"}\n",
	)
}

func (g *Generator) generateAOTSpecializedRecordValueMethods(
	record *aotRecordType,
) {
	typeName := record.typeName
	stateName := record.stateName
	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) aotRecordFast() bool {\n"+
			"\treturn r != nil && (r.state == nil || r.state.fields == nil)\n"+
			"}\n"+
			"func (r *%s) aotRecordAttrs() *lang.RecordAttrs {\n"+
			"\tif r.state == nil { return nil }\n"+
			"\treturn r.state.attrs\n"+
			"}\n"+
			"func (r *%s) aotRecordAttrsPtr() **lang.RecordAttrs {\n"+
			"\tif r.state == nil { r.state = &%s{} }\n"+
			"\treturn &r.state.attrs\n"+
			"}\n"+
			"func (r *%s) RecordType() *lang.RecordType { return %s }\n"+
			"func (r *%s) RecordExtMap() lang.IPersistentMap {\n"+
			"\treturn lang.RecordAttrsExt(r.aotRecordAttrs())\n"+
			"}\n"+
			"func (r *%s) RecordMeta() lang.IPersistentMap {\n"+
			"\treturn lang.RecordAttrsMeta(r.aotRecordAttrs())\n"+
			"}\n"+
			"func (r *%s) Meta() lang.IPersistentMap { return r.RecordMeta() }\n",
		typeName,
		typeName,
		typeName, stateName,
		typeName, record.descriptorGo,
		typeName,
		typeName,
		typeName,
	)
	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) RecordField(index int) any {\n"+
			"\tif r.state != nil && r.state.fields != nil {\n"+
			"\t\treturn r.state.fields[index]\n"+
			"\t}\n"+
			"\tswitch index {\n",
		typeName,
	)
	for index, field := range record.shape.Fields {
		if field.Kind == compiler.IRRecordSpecializedRecord {
			fmt.Fprintf(&g.aotDeclarations,
				"\tcase %d:\n"+
					"\t\tif r.f%d == nil { return nil }\n"+
					"\t\treturn r.f%d\n",
				index, index, index)
		} else {
			fmt.Fprintf(&g.aotDeclarations,
				"\tcase %d: return r.f%d\n", index, index)
		}
	}
	fmt.Fprintln(&g.aotDeclarations,
		"\tdefault:\n"+
			"\t\tpanic(lang.NewIllegalArgumentError("+
			"\"record field index out of bounds\"))\n"+
			"\t}\n"+
			"}")

	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) RecordWithField("+
			"index int, value any) lang.RecordValue {\n"+
			"\tif lang.Identical(r.RecordField(index), value) { return r }\n"+
			"\tfields := make([]any, %d)\n"+
			"\tfor i := range fields { fields[i] = r.RecordField(i) }\n"+
			"\tfields[index] = value\n"+
			"\tresult := &%s{state: &%s{\n"+
			"\t\tattrs: lang.RecordAttrsWithoutHash(r.aotRecordAttrs()),\n"+
			"\t\tfields: fields,\n"+
			"\t}}\n"+
			"\treturn result\n"+
			"}\n",
		typeName,
		len(record.fieldNames),
		typeName,
		stateName,
	)
	fmt.Fprintf(&g.aotDeclarations,
		"func (r *%s) RecordWithExtMap("+
			"ext lang.IPersistentMap) lang.RecordValue {\n"+
			"\tif r.RecordExtMap() == ext { return r }\n"+
			"\tresult := *r\n"+
			"\tstate := %s{}\n"+
			"\tif r.state != nil { state = *r.state }\n"+
			"\tstate.attrs = lang.RecordAttrsWithExt("+
			"r.aotRecordAttrs(), ext)\n"+
			"\tresult.state = &state\n"+
			"\treturn &result\n"+
			"}\n"+
			"func (r *%s) RecordWithMeta("+
			"meta lang.IPersistentMap) lang.RecordValue {\n"+
			"\tif r.RecordMeta() == meta { return r }\n"+
			"\tresult := *r\n"+
			"\tstate := %s{}\n"+
			"\tif r.state != nil { state = *r.state }\n"+
			"\tstate.attrs = lang.RecordAttrsWithMeta("+
			"r.aotRecordAttrs(), meta)\n"+
			"\tresult.state = &state\n"+
			"\treturn &result\n"+
			"}\n",
		typeName, stateName,
		typeName, stateName,
	)
}

func (g *Generator) prepareAOTRecordSpecializations(vars []namedVar) {
	if !g.directLink {
		return
	}
	constructors := make(map[*lang.Var]*lang.RecordType)
	methods := make(map[*lang.Var]*ast.FnMethodNode)
	var records []*lang.RecordType
	seenRecords := make(map[*lang.RecordType]bool)
	for _, named := range vars {
		vr := named.vr
		if !vr.IsBound() || vr.IsMacro() || vr.IsDynamic() ||
			RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef)) {
			continue
		}
		switch value := codegenVarValue(vr).(type) {
		case *lang.RecordConstructor:
			if !value.FromMap() {
				constructors[vr] = value.RecordType()
				if !seenRecords[value.RecordType()] {
					seenRecords[value.RecordType()] = true
					records = append(records, value.RecordType())
				}
			}
		case *Fn:
			fn := value.ASTNode().Sub.(*ast.FnNode)
			if !fn.IsVariadic && len(fn.Methods) == 1 {
				method := fn.Methods[0].Sub.(*ast.FnMethodNode)
				if !method.IsVariadic && method.FixedArity <= 20 {
					methods[vr] = method
				}
			}
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].FullName() < records[j].FullName()
	})
	var functionVars []*lang.Var
	for vr := range methods {
		functionVars = append(functionVars, vr)
	}
	sort.Slice(functionVars, func(i, j int) bool {
		return functionVars[i].String() < functionVars[j].String()
	})
	producerParams := recordProducerCallSiteParams(functionVars, methods)

	signatures := make(map[*lang.Var]compiler.IRRecordFunctionSignature)
	for _, record := range records {
		for _, vr := range functionVars {
			params := producerParams[vr]
			if params == nil {
				continue
			}
			shape, plan := compiler.InferRecursiveRecordProducerWithParams(
				methods[vr],
				vr,
				record,
				constructors,
				params,
			)
			if shape == nil || plan == nil {
				continue
			}
			if existing := g.aotRecordShapes[record]; existing != nil &&
				!sameAOTRecordShape(existing, shape) {
				continue
			}
			g.aotRecordShapes[record] = shape
			g.aotRecordPlans[vr] = plan
			signatures[vr] = plan.Signature
		}
	}

	for {
		changed := false
		for _, vr := range functionVars {
			if g.aotRecordPlans[vr] != nil {
				continue
			}
			method := methods[vr]
			for _, params := range recordAOTParamCandidates(
				method.FixedArity,
				records,
				g.aotRecordShapes,
			) {
				result := compiler.IRRecordSpecializedType{
					Kind: compiler.IRRecordSpecializedInt64,
				}
				candidate := compiler.IRRecordFunctionSignature{
					Params: params,
					Result: result,
				}
				available := cloneAOTRecordSignatures(signatures)
				available[vr] = candidate
				plan := compiler.AnalyzeRecordSpecializedFunction(
					method,
					params,
					result,
					constructors,
					g.aotRecordShapes,
					available,
				)
				if plan == nil {
					continue
				}
				g.aotRecordPlans[vr] = plan
				signatures[vr] = plan.Signature
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
}

func recordProducerCallSiteParams(
	functionVars []*lang.Var,
	methods map[*lang.Var]*ast.FnMethodNode,
) map[*lang.Var][]compiler.IRRecordSpecializedType {
	type observed struct {
		params []compiler.IRRecordSpecializedType
	}
	byTarget := make(map[*lang.Var]map[string]observed)
	for _, caller := range functionVars {
		value, ok := codegenVarValue(caller).(*Fn)
		if !ok {
			continue
		}
		for _, site := range compiler.BuildTypedIR(
			value.ASTNode(),
		).DirectCallSites() {
			if methods[site.Var] == nil {
				continue
			}
			params, key, ok := recordParamsFromIR(site.ArgumentTypes)
			if !ok {
				continue
			}
			if byTarget[site.Var] == nil {
				byTarget[site.Var] = make(map[string]observed)
			}
			byTarget[site.Var][key] = observed{params: params}
		}
	}
	result := make(map[*lang.Var][]compiler.IRRecordSpecializedType)
	for target, candidates := range byTarget {
		// A single helper cannot represent conflicting parameter signatures.
		// Retain ordinary dispatch until multi-versioning is available.
		if len(candidates) != 1 {
			continue
		}
		for _, candidate := range candidates {
			result[target] = candidate.params
		}
	}
	return result
}

func recordParamsFromIR(
	types []compiler.IRType,
) ([]compiler.IRRecordSpecializedType, string, bool) {
	params := make([]compiler.IRRecordSpecializedType, len(types))
	var key strings.Builder
	for index, typ := range types {
		switch typ.Kind {
		case compiler.IRBool:
			if typ.GoType != reflect.TypeOf(false) {
				return nil, "", false
			}
			params[index].Kind = compiler.IRRecordSpecializedBool
			key.WriteByte('b')
		case compiler.IRInt:
			if typ.GoType != reflect.TypeOf(int64(0)) {
				return nil, "", false
			}
			params[index].Kind = compiler.IRRecordSpecializedInt64
			key.WriteByte('i')
		default:
			return nil, "", false
		}
	}
	return params, key.String(), true
}

func sameAOTRecordShape(
	left, right *compiler.IRRecordShape,
) bool {
	if left == nil || right == nil || left.Record != right.Record ||
		len(left.Fields) != len(right.Fields) {
		return false
	}
	for index := range left.Fields {
		if !left.Fields[index].Equal(right.Fields[index]) {
			return false
		}
	}
	return true
}

func recordAOTParamCandidates(
	arity int,
	records []*lang.RecordType,
	shapes map[*lang.RecordType]*compiler.IRRecordShape,
) [][]compiler.IRRecordSpecializedType {
	if arity < 1 || arity > 20 {
		return nil
	}
	intType := compiler.IRRecordSpecializedType{
		Kind: compiler.IRRecordSpecializedInt64,
	}
	var typedRecords []compiler.IRRecordSpecializedType
	for _, record := range records {
		if shapes[record] != nil {
			typedRecords = append(typedRecords, compiler.IRRecordSpecializedType{
				Kind:   compiler.IRRecordSpecializedRecord,
				Record: record,
			})
		}
	}
	const candidateBudget = 1 << 12
	candidateCount := 1
	for range arity {
		candidateCount *= 1 + len(typedRecords)
		if candidateCount > candidateBudget {
			return nil
		}
	}
	var result [][]compiler.IRRecordSpecializedType
	var visit func(int, []compiler.IRRecordSpecializedType, bool)
	visit = func(
		index int,
		params []compiler.IRRecordSpecializedType,
		hasRecord bool,
	) {
		if index == arity {
			if hasRecord {
				result = append(
					result,
					append([]compiler.IRRecordSpecializedType(nil), params...),
				)
			}
			return
		}
		visit(index+1, append(params, intType), hasRecord)
		for _, record := range typedRecords {
			visit(index+1, append(params, record), true)
		}
	}
	visit(0, nil, false)
	return result
}

func cloneAOTRecordSignatures(
	source map[*lang.Var]compiler.IRRecordFunctionSignature,
) map[*lang.Var]compiler.IRRecordFunctionSignature {
	result := make(
		map[*lang.Var]compiler.IRRecordFunctionSignature,
		len(source)+1,
	)
	for vr, signature := range source {
		result[vr] = signature
	}
	return result
}

func recordAOTTypeExpr(
	g *Generator,
	typ compiler.IRRecordSpecializedType,
) string {
	switch typ.Kind {
	case compiler.IRRecordSpecializedBool:
		return "bool"
	case compiler.IRRecordSpecializedInt64:
		return "int64"
	case compiler.IRRecordSpecializedRecord:
		return "*" + g.allocAOTRecordType(typ.Record).typeName
	default:
		panic(fmt.Sprintf("unsupported record AOT type: %v", typ.Kind))
	}
}

type recordAOTEmitter struct {
	g        *Generator
	target   *aotSpecializationTarget
	analysis *compiler.IRRecordFunctionPlan
	helper   string
}

func (g *Generator) generateRecordSpecializedFixedFn(
	fn *Fn,
	fnVar string,
	method *ast.FnMethodNode,
	paramNames []string,
) bool {
	target := g.specializationTarget
	if target == nil || target.fn != fn || target.recordAnalysis == nil {
		return false
	}
	analysis := target.recordAnalysis
	helper := g.allocateTempVar()
	typedParams := make([]string, len(paramNames))
	locals := make(map[*lang.Symbol]aotTypedLocal, len(paramNames))
	for index, param := range method.Params {
		name := g.allocateTempVar()
		typ := analysis.Signature.Params[index]
		typedParams[index] = name + " " + recordAOTTypeExpr(g, typ)
		locals[param.Sub.(*ast.BindingNode).Name] = aotTypedLocal{
			name: name,
		}
	}
	resultType := recordAOTTypeExpr(g, analysis.Signature.Result)
	g.writef("var %s func(%s) %s\n",
		helper, strings.Join(typedParams, ", "), resultType)
	g.writef("%s = func(%s) %s {\n",
		helper, strings.Join(typedParams, ", "), resultType)
	emitter := recordAOTEmitter{
		g:        g,
		target:   target,
		analysis: analysis,
		helper:   helper,
	}
	result := emitter.emitExpr(method.Body, locals)
	g.writef("return %s\n", result)
	g.writef("}\n")
	g.writef("%s = %s\n", target.recordFnVar, helper)

	signature := ""
	if len(paramNames) > 0 {
		signature = strings.Join(paramNames, ", ") + " any"
	}
	g.writef("%s = lang.FnFunc%d(func(%s) any {\n",
		fnVar, len(paramNames), signature)
	guards := make([]string, len(paramNames))
	args := make([]string, len(paramNames))
	for index, param := range paramNames {
		typ := analysis.Signature.Params[index]
		args[index] = g.allocateTempVar()
		guards[index] = g.allocateTempVar()
		g.writef("%s, %s := %s.(%s)\n",
			args[index], guards[index], param, recordAOTTypeExpr(g, typ))
		if typ.Kind == compiler.IRRecordSpecializedRecord {
			g.writef("%s = %s && %s.aotRecordFast()\n",
				guards[index], guards[index], args[index])
		}
	}
	if len(guards) > 0 {
		g.writef("if %s {\n", strings.Join(guards, " && "))
	}
	g.writef("return %s(%s)\n", helper, strings.Join(args, ", "))
	if len(guards) > 0 {
		g.writef("}\n")
	}
	g.generateFnMethodFixed(method, paramNames)
	g.writef("})\n")
	return true
}

func (e *recordAOTEmitter) emitExpr(
	node *ast.Node,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	node = unwrapAOTDo(node)
	switch node.Op {
	case ast.OpConst:
		switch value := node.Sub.(*ast.ConstNode).Value.(type) {
		case nil:
			return "nil"
		case bool:
			return strconv.FormatBool(value)
		case int64:
			return "int64(" + strconv.FormatInt(value, 10) + ")"
		}

	case ast.OpLocal:
		return locals[node.Sub.(*ast.LocalNode).Name].name

	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		result := e.g.allocateTempVar()
		resultType := recordAOTTypeExpr(e.g, e.analysis.Types[node])
		e.g.writef("var %s %s\n", result, resultType)
		e.g.writef("{\n")
		nested := cloneAOTLocals(locals)
		for _, binding := range let.Bindings {
			binding := binding.Sub.(*ast.BindingNode)
			value := e.emitExpr(binding.Init, nested)
			name := e.g.allocateTempVar()
			e.g.writef("%s := %s\n", name, value)
			nested[binding.Name] = aotTypedLocal{name: name}
		}
		value := e.emitExpr(let.Body, nested)
		e.g.writef("%s = %s\n", result, value)
		e.g.writef("}\n")
		return result

	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		result := e.g.allocateTempVar()
		resultType := recordAOTTypeExpr(e.g, e.analysis.Types[node])
		e.g.writef("var %s %s\n", result, resultType)
		test := e.emitExpr(conditional.Test, locals)
		e.g.writef("if %s {\n", test)
		e.g.writef("%s = %s\n", result, e.emitExpr(conditional.Then, locals))
		e.g.writef("} else {\n")
		e.g.writef("%s = %s\n", result, e.emitExpr(conditional.Else, locals))
		e.g.writef("}\n")
		return result

	case ast.OpHostCall:
		return e.emitHostCall(node.Sub.(*ast.HostCallNode), locals)

	case ast.OpInvoke:
		return e.emitInvoke(node, node.Sub.(*ast.InvokeNode), locals)

	case ast.OpKeywordLookup:
		lookup := node.Sub.(*ast.KeywordLookupNode)
		target := e.emitExpr(lookup.Target, locals)
		index := e.analysis.FieldReads[node]
		recordType := e.analysis.Types[lookup.Target].Record
		record := e.g.allocAOTRecordType(recordType)
		fieldType := record.shape.Fields[index]
		switch fieldType.Kind {
		case compiler.IRRecordSpecializedBool:
			return target + ".f" + strconv.Itoa(index)
		case compiler.IRRecordSpecializedInt64:
			return target + ".f" + strconv.Itoa(index)
		case compiler.IRRecordSpecializedRecord:
			return target + ".f" + strconv.Itoa(index)
		}
	}
	panic(fmt.Sprintf("unsupported record AOT expression: %v", node.Op))
}

func (e *recordAOTEmitter) emitHostCall(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	args := make([]string, len(call.Args))
	for index, argument := range call.Args {
		args[index] = e.emitExpr(argument, locals)
	}
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		switch name {
		case "inc":
			return "lang.CheckedAddInt64(" + args[0] + ", 1)"
		case "unchecked_inc":
			return "(" + args[0] + " + 1)"
		case "dec":
			return "lang.CheckedSubInt64(" + args[0] + ", 1)"
		case "uncheckeddec", "unchecked_dec":
			return "(" + args[0] + " - 1)"
		case "minus":
			return "lang.CheckedNegateInt64(" + args[0] + ")"
		case "unchecked_minus":
			return "(-" + args[0] + ")"
		case "iszero":
			return "(" + args[0] + " == 0)"
		case "ispos":
			return "(" + args[0] + " > 0)"
		case "isneg":
			return "(" + args[0] + " < 0)"
		}
	}
	switch name {
	case "add":
		return "lang.CheckedAddInt64(" + args[0] + ", " + args[1] + ")"
	case "uncheckedadd":
		return "(" + args[0] + " + " + args[1] + ")"
	case "minus":
		return "lang.CheckedSubInt64(" + args[0] + ", " + args[1] + ")"
	case "unchecked_minus":
		return "(" + args[0] + " - " + args[1] + ")"
	case "multiply":
		return "lang.CheckedMultiplyInt64(" + args[0] + ", " + args[1] + ")"
	case "unchecked_multiply":
		return "(" + args[0] + " * " + args[1] + ")"
	case "quotient":
		return "(" + args[0] + " / " + args[1] + ")"
	case "remainder":
		return "(" + args[0] + " % " + args[1] + ")"
	case "lt":
		return "(" + args[0] + " < " + args[1] + ")"
	case "lte":
		return "(" + args[0] + " <= " + args[1] + ")"
	case "gt":
		return "(" + args[0] + " > " + args[1] + ")"
	case "gte":
		return "(" + args[0] + " >= " + args[1] + ")"
	case "equiv":
		return "(" + args[0] + " == " + args[1] + ")"
	}
	panic("unsupported record AOT host call")
}

func (e *recordAOTEmitter) emitInvoke(
	node *ast.Node,
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]aotTypedLocal,
) string {
	args := make([]string, len(invoke.Args))
	for index, argument := range invoke.Args {
		args[index] = e.emitExpr(argument, locals)
	}
	if recordType := e.analysis.Constructors[node]; recordType != nil {
		record := e.g.allocAOTRecordType(recordType)
		return record.fastCtor + "(" + strings.Join(args, ", ") + ")"
	}
	if vr := e.analysis.Calls[node]; vr != nil {
		helper := e.helper
		if vr != e.target.vr {
			helper = e.g.aotCallTargets[vr].recordFnVar
		}
		return helper + "(" + strings.Join(args, ", ") + ")"
	}
	if invoke.Fn.Op == ast.OpVar {
		vr := invoke.Fn.Sub.(*ast.VarNode).Var
		if vr.Namespace() == lang.NSCore && vr.Symbol().String() == "-" &&
			len(args) == 1 {
			return "lang.CheckedNegateInt64(" + args[0] + ")"
		}
		if vr.Namespace() == lang.NSCore && vr.Symbol().String() == "=" {
			return "(" + args[0] + " == " + args[1] + ")"
		}
	}
	if invoke.Fn.Op == ast.OpConst && len(args) == 2 {
		return "(" + args[0] + " == " + args[1] + ")"
	}
	panic("unsupported record AOT invocation")
}
