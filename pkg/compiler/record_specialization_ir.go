package compiler

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

// IRRecordSpecializedKind names concrete representations supported by a
// record-specialized function region.
type IRRecordSpecializedKind uint8

const (
	IRRecordSpecializedInvalid IRRecordSpecializedKind = iota
	IRRecordSpecializedNil
	IRRecordSpecializedBool
	IRRecordSpecializedInt64
	IRRecordSpecializedRecord
)

type IRRecordSpecializedType struct {
	Kind   IRRecordSpecializedKind
	Record *lang.RecordType
}

func (t IRRecordSpecializedType) Equal(other IRRecordSpecializedType) bool {
	return t.Kind == other.Kind && t.Record == other.Record
}

type IRRecordShape struct {
	Record *lang.RecordType
	Fields []IRRecordSpecializedType
}

type IRRecordFunctionSignature struct {
	Params []IRRecordSpecializedType
	Result IRRecordSpecializedType
}

// IRRecordFunctionPlan is backend-neutral evidence that a complete function
// body can run with unboxed integers and concrete record pointers. Nodes keep
// their ordinary AST identity so unsupported backends retain the fallback.
type IRRecordFunctionPlan struct {
	Signature    IRRecordFunctionSignature
	Types        map[*ast.Node]IRRecordSpecializedType
	Constructors map[*ast.Node]*lang.RecordType
	FieldReads   map[*ast.Node]int
	Calls        map[*ast.Node]*lang.Var
}

// InferRecursiveRecordProducer finds functions which build one record shape
// from int64 parameters and recursive calls to themselves. Multiple
// constructor branches must agree field-by-field; nil and the same record
// type combine into a nullable record pointer.
func InferRecursiveRecordProducer(
	method *ast.FnMethodNode,
	self *lang.Var,
	record *lang.RecordType,
	constructors map[*lang.Var]*lang.RecordType,
) (*IRRecordShape, *IRRecordFunctionPlan) {
	if method == nil || method.IsVariadic || method.FixedArity < 1 ||
		method.FixedArity > 4 || self == nil || record == nil {
		return nil, nil
	}
	intType := IRRecordSpecializedType{Kind: IRRecordSpecializedInt64}
	params := make([]IRRecordSpecializedType, method.FixedArity)
	locals := make(map[*lang.Symbol]IRRecordSpecializedType, len(params))
	for index, param := range method.Params {
		params[index] = intType
		locals[param.Sub.(*ast.BindingNode).Name] = intType
	}
	signatures := map[*lang.Var]IRRecordFunctionSignature{
		self: {
			Params: params,
			Result: IRRecordSpecializedType{
				Kind:   IRRecordSpecializedRecord,
				Record: record,
			},
		},
	}
	analyzer := newIRRecordFunctionAnalyzer(
		constructors,
		nil,
		signatures,
	)
	result := analyzer.expr(method.Body, locals)
	if !analyzer.valid ||
		result.Kind != IRRecordSpecializedRecord ||
		result.Record != record ||
		len(analyzer.constructorFields) == 0 {
		return nil, nil
	}
	fields := make([]IRRecordSpecializedType, record.FieldCount())
	for index := range fields {
		var merged IRRecordSpecializedType
		for _, constructorFields := range analyzer.constructorFields {
			if len(constructorFields) != len(fields) {
				return nil, nil
			}
			var ok bool
			merged, ok = mergeIRRecordFieldTypes(
				merged,
				constructorFields[index],
				record,
			)
			if !ok {
				return nil, nil
			}
		}
		if merged.Kind == IRRecordSpecializedNil {
			merged = IRRecordSpecializedType{
				Kind:   IRRecordSpecializedRecord,
				Record: record,
			}
		}
		fields[index] = merged
	}
	shape := &IRRecordShape{Record: record, Fields: fields}
	plan := AnalyzeRecordSpecializedFunction(
		method,
		params,
		signatures[self].Result,
		constructors,
		map[*lang.RecordType]*IRRecordShape{record: shape},
		signatures,
	)
	return shape, plan
}

func AnalyzeRecordSpecializedFunction(
	method *ast.FnMethodNode,
	params []IRRecordSpecializedType,
	result IRRecordSpecializedType,
	constructors map[*lang.Var]*lang.RecordType,
	shapes map[*lang.RecordType]*IRRecordShape,
	signatures map[*lang.Var]IRRecordFunctionSignature,
) *IRRecordFunctionPlan {
	if method == nil || method.IsVariadic ||
		len(params) != method.FixedArity ||
		result.Kind == IRRecordSpecializedInvalid {
		return nil
	}
	locals := make(map[*lang.Symbol]IRRecordSpecializedType, len(params))
	for index, param := range method.Params {
		locals[param.Sub.(*ast.BindingNode).Name] = params[index]
	}
	analyzer := newIRRecordFunctionAnalyzer(constructors, shapes, signatures)
	actual := analyzer.expr(method.Body, locals)
	if !analyzer.valid || !actual.Equal(result) {
		return nil
	}
	return &IRRecordFunctionPlan{
		Signature: IRRecordFunctionSignature{
			Params: append([]IRRecordSpecializedType(nil), params...),
			Result: result,
		},
		Types:        analyzer.types,
		Constructors: analyzer.constructorsUsed,
		FieldReads:   analyzer.fieldReads,
		Calls:        analyzer.calls,
	}
}

type irRecordFunctionAnalyzer struct {
	constructors      map[*lang.Var]*lang.RecordType
	shapes            map[*lang.RecordType]*IRRecordShape
	signatures        map[*lang.Var]IRRecordFunctionSignature
	types             map[*ast.Node]IRRecordSpecializedType
	constructorFields [][]IRRecordSpecializedType
	constructorsUsed  map[*ast.Node]*lang.RecordType
	fieldReads        map[*ast.Node]int
	calls             map[*ast.Node]*lang.Var
	valid             bool
}

func newIRRecordFunctionAnalyzer(
	constructors map[*lang.Var]*lang.RecordType,
	shapes map[*lang.RecordType]*IRRecordShape,
	signatures map[*lang.Var]IRRecordFunctionSignature,
) *irRecordFunctionAnalyzer {
	return &irRecordFunctionAnalyzer{
		constructors:     constructors,
		shapes:           shapes,
		signatures:       signatures,
		types:            make(map[*ast.Node]IRRecordSpecializedType),
		constructorsUsed: make(map[*ast.Node]*lang.RecordType),
		fieldReads:       make(map[*ast.Node]int),
		calls:            make(map[*ast.Node]*lang.Var),
		valid:            true,
	}
}

func (a *irRecordFunctionAnalyzer) expr(
	node *ast.Node,
	locals map[*lang.Symbol]IRRecordSpecializedType,
) IRRecordSpecializedType {
	node = irUnwrapDo(node)
	if node == nil || !a.valid {
		return IRRecordSpecializedType{}
	}
	var result IRRecordSpecializedType
	switch node.Op {
	case ast.OpConst:
		switch node.Sub.(*ast.ConstNode).Value.(type) {
		case nil:
			result.Kind = IRRecordSpecializedNil
		case bool:
			result.Kind = IRRecordSpecializedBool
		case int64:
			result.Kind = IRRecordSpecializedInt64
		}

	case ast.OpLocal:
		result = locals[node.Sub.(*ast.LocalNode).Name]

	case ast.OpLet:
		let := node.Sub.(*ast.LetNode)
		nested := cloneIRRecordTypes(locals)
		for _, binding := range let.Bindings {
			binding := binding.Sub.(*ast.BindingNode)
			valueType := a.expr(binding.Init, nested)
			if valueType.Kind == IRRecordSpecializedInvalid {
				a.valid = false
				break
			}
			nested[binding.Name] = valueType
		}
		result = a.expr(let.Body, nested)

	case ast.OpIf:
		conditional := node.Sub.(*ast.IfNode)
		if a.expr(conditional.Test, locals).Kind !=
			IRRecordSpecializedBool {
			a.valid = false
			break
		}
		left := a.expr(conditional.Then, locals)
		right := a.expr(conditional.Else, locals)
		var ok bool
		result, ok = mergeIRRecordResultTypes(left, right)
		if !ok {
			a.valid = false
		}

	case ast.OpHostCall:
		result = a.hostCall(node.Sub.(*ast.HostCallNode), locals)

	case ast.OpInvoke:
		result = a.invoke(node, node.Sub.(*ast.InvokeNode), locals)

	case ast.OpKeywordLookup:
		lookup := node.Sub.(*ast.KeywordLookupNode)
		target := a.expr(lookup.Target, locals)
		if target.Kind != IRRecordSpecializedRecord {
			a.valid = false
			break
		}
		shape := a.shapes[target.Record]
		index, ok := target.Record.FieldIndex(lookup.Keyword)
		if shape == nil || !ok || index >= len(shape.Fields) {
			a.valid = false
			break
		}
		if lookup.Default != nil {
			fallback := a.expr(lookup.Default, locals)
			if fallback.Kind != IRRecordSpecializedNil {
				a.valid = false
				break
			}
		}
		result = shape.Fields[index]
		a.fieldReads[node] = index

	default:
		a.valid = false
	}
	if result.Kind == IRRecordSpecializedInvalid {
		a.valid = false
	}
	a.types[node] = result
	return result
}

func (a *irRecordFunctionAnalyzer) hostCall(
	call *ast.HostCallNode,
	locals map[*lang.Symbol]IRRecordSpecializedType,
) IRRecordSpecializedType {
	if !irRecordNumbersCall(call) {
		return IRRecordSpecializedType{}
	}
	args := make([]IRRecordSpecializedType, len(call.Args))
	for index, argument := range call.Args {
		args[index] = a.expr(argument, locals)
	}
	name := strings.ToLower(call.Method.Name())
	if len(args) == 1 {
		switch name {
		case "inc", "unchecked_inc", "dec", "uncheckeddec", "unchecked_dec",
			"minus", "unchecked_minus":
			if args[0].Kind == IRRecordSpecializedInt64 {
				return args[0]
			}
		case "iszero", "ispos", "isneg":
			if args[0].Kind == IRRecordSpecializedInt64 {
				return IRRecordSpecializedType{
					Kind: IRRecordSpecializedBool,
				}
			}
		}
		return IRRecordSpecializedType{}
	}
	if len(args) != 2 ||
		args[0].Kind != IRRecordSpecializedInt64 ||
		args[1].Kind != IRRecordSpecializedInt64 {
		return IRRecordSpecializedType{}
	}
	switch name {
	case "add", "uncheckedadd", "minus", "unchecked_minus",
		"multiply", "unchecked_multiply", "quotient", "remainder":
		return IRRecordSpecializedType{Kind: IRRecordSpecializedInt64}
	case "lt", "lte", "gt", "gte", "equiv":
		return IRRecordSpecializedType{Kind: IRRecordSpecializedBool}
	}
	return IRRecordSpecializedType{}
}

func (a *irRecordFunctionAnalyzer) invoke(
	node *ast.Node,
	invoke *ast.InvokeNode,
	locals map[*lang.Symbol]IRRecordSpecializedType,
) IRRecordSpecializedType {
	args := make([]IRRecordSpecializedType, len(invoke.Args))
	for index, argument := range invoke.Args {
		args[index] = a.expr(argument, locals)
	}
	if invoke.Fn.Op == ast.OpVar {
		vr := invoke.Fn.Sub.(*ast.VarNode).Var
		if record := a.constructors[vr]; record != nil {
			if len(args) != record.FieldCount() {
				return IRRecordSpecializedType{}
			}
			a.constructorFields = append(a.constructorFields, args)
			a.constructorsUsed[node] = record
			return IRRecordSpecializedType{
				Kind:   IRRecordSpecializedRecord,
				Record: record,
			}
		}
		if signature, ok := a.signatures[vr]; ok &&
			irRecordArgsMatch(args, signature.Params) {
			a.calls[node] = vr
			return signature.Result
		}
		if vr.Namespace() == lang.NSCore &&
			vr.Symbol().String() == "-" &&
			len(args) == 1 &&
			args[0].Kind == IRRecordSpecializedInt64 {
			return args[0]
		}
		if vr.Namespace() == lang.NSCore &&
			vr.Symbol().String() == "=" &&
			len(args) == 2 && args[0].Equal(args[1]) &&
			(args[0].Kind == IRRecordSpecializedInt64 ||
				args[0].Kind == IRRecordSpecializedBool) {
			return IRRecordSpecializedType{
				Kind: IRRecordSpecializedBool,
			}
		}
		return IRRecordSpecializedType{}
	}
	if invoke.Fn.Op == ast.OpConst &&
		irRecordIdenticalCall(invoke.Fn.Sub.(*ast.ConstNode)) &&
		len(args) == 2 {
		if args[0].Equal(args[1]) ||
			args[0].Kind == IRRecordSpecializedNil &&
				args[1].Kind == IRRecordSpecializedRecord ||
			args[1].Kind == IRRecordSpecializedNil &&
				args[0].Kind == IRRecordSpecializedRecord {
			return IRRecordSpecializedType{
				Kind: IRRecordSpecializedBool,
			}
		}
	}
	return IRRecordSpecializedType{}
}

func irRecordNumbersCall(call *ast.HostCallNode) bool {
	if call == nil || call.Target == nil || call.Target.Op != ast.OpConst {
		return false
	}
	target := call.Target.Sub.(*ast.ConstNode)
	return target.Value == lang.Numbers ||
		target.HostSymbol != nil &&
			target.HostSymbol.String() ==
				"github.com:glojurelang:glojure:pkg:lang.Numbers"
}

func irRecordIdenticalCall(fn *ast.ConstNode) bool {
	return fn != nil && fn.HostSymbol != nil &&
		fn.HostSymbol.String() ==
			"github.com:glojurelang:glojure:pkg:lang.Identical"
}

func irRecordArgsMatch(
	actual, expected []IRRecordSpecializedType,
) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if !actual[index].Equal(expected[index]) {
			return false
		}
	}
	return true
}

func mergeIRRecordFieldTypes(
	left, right IRRecordSpecializedType,
	record *lang.RecordType,
) (IRRecordSpecializedType, bool) {
	if left.Kind == IRRecordSpecializedInvalid {
		return right, right.Kind != IRRecordSpecializedInvalid
	}
	if left.Equal(right) {
		return left, true
	}
	if left.Kind == IRRecordSpecializedNil &&
		right.Kind == IRRecordSpecializedRecord &&
		right.Record == record {
		return right, true
	}
	if right.Kind == IRRecordSpecializedNil &&
		left.Kind == IRRecordSpecializedRecord &&
		left.Record == record {
		return left, true
	}
	return IRRecordSpecializedType{}, false
}

func mergeIRRecordResultTypes(
	left, right IRRecordSpecializedType,
) (IRRecordSpecializedType, bool) {
	if left.Equal(right) {
		return left, left.Kind != IRRecordSpecializedInvalid
	}
	if left.Kind == IRRecordSpecializedNil &&
		right.Kind == IRRecordSpecializedRecord {
		return right, true
	}
	if right.Kind == IRRecordSpecializedNil &&
		left.Kind == IRRecordSpecializedRecord {
		return left, true
	}
	return IRRecordSpecializedType{}, false
}

func cloneIRRecordTypes(
	locals map[*lang.Symbol]IRRecordSpecializedType,
) map[*lang.Symbol]IRRecordSpecializedType {
	result := make(map[*lang.Symbol]IRRecordSpecializedType, len(locals)+1)
	for symbol, typ := range locals {
		result[symbol] = typ
	}
	return result
}
