package ast

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
)

// Dump writes a stable, human-readable representation of an analyzed AST.
// Runtime-only details such as environments, source forms, resolved Go
// methods, and object addresses are intentionally omitted.
func Dump(w io.Writer, root *Node) error {
	var b strings.Builder
	dumpNode(&b, root, 0)
	b.WriteByte('\n')
	_, err := io.WriteString(w, b.String())
	return err
}

// Format returns the same representation as Dump as a string.
func Format(root *Node) string {
	var b strings.Builder
	dumpNode(&b, root, 0)
	b.WriteByte('\n')
	return b.String()
}

func dumpNode(b *strings.Builder, node *Node, depth int) {
	if node == nil {
		b.WriteString("nil")
		return
	}

	b.WriteByte('(')
	b.WriteString(node.Op.String())
	if node.IsLiteral {
		b.WriteString(" :literal true")
	}
	if node.IsAssignable {
		b.WriteString(" :assignable true")
	}
	dumpStruct(b, reflect.ValueOf(node.Sub), depth+1)
	b.WriteByte(')')
}

func dumpStruct(b *strings.Builder, value reflect.Value, depth int) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" || field.Name == "Meta" || field.Name == "ResolvedMethod" {
			continue
		}
		if field.Name == "Value" && typ == reflect.TypeOf(ConstNode{}) {
			hostSymbol := value.FieldByName("HostSymbol")
			if !hostSymbol.IsNil() {
				continue
			}
		}
		fieldValue := value.Field(i)
		if omitDumpField(field.Name, fieldValue) {
			continue
		}
		b.WriteByte('\n')
		dumpIndent(b, depth)
		b.WriteByte(':')
		b.WriteString(dumpFieldName(field.Name))
		b.WriteByte(' ')
		dumpValue(b, fieldValue, depth)
	}
}

func dumpValue(b *strings.Builder, value reflect.Value, depth int) {
	if !value.IsValid() {
		b.WriteString("nil")
		return
	}
	if value.CanInterface() {
		switch value.Interface().(type) {
		case lang.Keyword, *lang.Symbol, *lang.Var, reflect.Type:
			dumpAtom(b, value.Interface())
			return
		}
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			b.WriteString("nil")
			return
		}
		dumpAtom(b, value.Interface())
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			b.WriteString("nil")
			return
		}
		if node, ok := value.Interface().(*Node); ok {
			dumpNode(b, node, depth)
			return
		}
		dumpAtom(b, value.Interface())
		return
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		b.WriteByte('[')
		for i := 0; i < value.Len(); i++ {
			if i > 0 {
				b.WriteByte('\n')
				dumpIndent(b, depth+1)
			}
			dumpValue(b, value.Index(i), depth+1)
		}
		b.WriteByte(']')
		return
	}
	if value.Kind() == reflect.Struct {
		b.WriteByte('{')
		dumpStruct(b, value, depth+1)
		b.WriteByte('}')
		return
	}
	dumpAtom(b, value.Interface())
}

func dumpAtom(b *strings.Builder, value interface{}) {
	switch value := value.(type) {
	case nil:
		b.WriteString("nil")
	case *lang.Var:
		b.WriteString(value.String())
	case reflect.Type:
		b.WriteString(value.String())
	case string:
		fmt.Fprintf(b, "%q", value)
	case bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		lang.Keyword, *lang.Symbol,
		lang.IPersistentMap, lang.IPersistentVector, lang.IPersistentSet, lang.ISeq:
		b.WriteString(lang.PrintString(value))
	default:
		typ := reflect.TypeOf(value)
		if typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Func ||
			typ.Kind() == reflect.Chan || typ.Kind() == reflect.Map ||
			typ.Kind() == reflect.Slice {
			fmt.Fprintf(b, "#object[%s]", typ)
			return
		}
		b.WriteString(lang.PrintString(value))
	}
}

func omitDumpField(name string, value reflect.Value) bool {
	if name == "Value" {
		return false
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
		return value.IsNil()
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint() == 0
	case reflect.String:
		return value.Len() == 0
	default:
		return false
	}
}

func dumpFieldName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('-')
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

func dumpIndent(b *strings.Builder, depth int) {
	b.WriteString(strings.Repeat("  ", depth))
}

func (op NodeOp) String() string {
	switch op {
	case OpUnknown:
		return "unknown"
	case OpConst:
		return "const"
	case OpDef:
		return "def"
	case OpSetBang:
		return "set!"
	case OpMaybeClass:
		return "maybe-class"
	case OpWithMeta:
		return "with-meta"
	case OpFn:
		return "fn"
	case OpFnMethod:
		return "fn-method"
	case OpMap:
		return "map"
	case OpVector:
		return "vector"
	case OpSet:
		return "set"
	case OpDo:
		return "do"
	case OpLet:
		return "let"
	case OpLetFn:
		return "letfn"
	case OpLoop:
		return "loop"
	case OpInvoke:
		return "invoke"
	case OpKeywordLookup:
		return "keyword-lookup"
	case OpAssoc:
		return "assoc"
	case OpReplaceLast:
		return "replace-last"
	case OpQuote:
		return "quote"
	case OpVar:
		return "var"
	case OpLocal:
		return "local"
	case OpBinding:
		return "binding"
	case OpHostCall:
		return "host-call"
	case OpHostInterop:
		return "host-interop"
	case OpHostField:
		return "host-field"
	case OpMaybeHostForm:
		return "maybe-host-form"
	case OpGoBuiltin:
		return "go-builtin"
	case OpGo:
		return "go"
	case OpIf:
		return "if"
	case OpCase:
		return "case"
	case OpCaseNode:
		return "case-node"
	case OpTheVar:
		return "the-var"
	case OpRecur:
		return "recur"
	case OpNew:
		return "new"
	case OpTry:
		return "try"
	case OpCatch:
		return "catch"
	case OpThrow:
		return "throw"
	default:
		return fmt.Sprintf("op-%d", op)
	}
}
