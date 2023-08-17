package main

import (
	// Bootstrap the runtime

	"fmt"
	"go/format"
	"os"
	"reflect"
	"strings"

	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

func main() {
	coreNS := lang.FindNamespace(lang.NewSymbol("glojure.core"))

	coreGB := NewNSGoBuilder(coreNS)
	buf := []byte(coreGB.String())

	os.Stdout.Write(buf)
}

const prelude = `
package core

import (
	"os"

	"github.com/glojurelang/glojure/pkg/lang"
)

var (
	ns = lang.FindOrCreateNamespace(lang.NewSymbol(%q))
)

`

type (
	NSGoBuilder struct {
		ns *lang.Namespace
	}
)

func NewNSGoBuilder(ns *lang.Namespace) *NSGoBuilder {
	return &NSGoBuilder{
		ns: ns,
	}
}

func (gb *NSGoBuilder) String() string {
	builder := &strings.Builder{}
	builder.WriteString(fmt.Sprintf(prelude, gb.ns.Name().String()))

	mappings := gb.ns.Mappings()

	builder.WriteString("func init() {\n")
	builder.WriteString(fmt.Sprintf("\tns.ResetMeta(%s)\n", gb.writeValue(gb.ns.Meta())))
	builder.WriteString("\n")

	for s := lang.Seq(mappings); s != nil; s = s.Next() {
		gb.writeMapping(builder, s.First().(lang.IMapEntry))
	}
	builder.WriteString("}\n")

	buf, err := format.Source([]byte(builder.String()))
	if err != nil {
		panic(err)
	}
	return string(buf)
}

func (gb *NSGoBuilder) writeMapping(builder *strings.Builder, entry lang.IMapEntry) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "panic while writing mapping '%v': %v\n", entry.Key(), r)
			fmt.Fprintf(os.Stderr, "- var value: %+v\n", entry.Val().(*lang.Var).Deref())
			panic(r)
		}
	}()

	// skip glojure.core/in-ns
	if gb.ns.Name().String() == "glojure.core" {
		switch entry.Key().(*lang.Symbol).Name() {
		case "in-ns":
			return
		case "list":
			builder.WriteString(fmt.Sprintf("\t\tns.InternWithValue(lang.NewSymbol(%q), lang.NewList, true)\n", entry.Key().(*lang.Symbol).Name()))
			return
		}
	}

	builder.WriteString(fmt.Sprintf("\t{\n"))
	builder.WriteString(fmt.Sprintf("\t\tvr := ns.Intern(lang.NewSymbol(%q))\n", entry.Key().(*lang.Symbol).Name()))
	if _, ok := entry.Val().(*lang.Var).Deref().(*lang.UnboundVar); ok {
		builder.WriteString(fmt.Sprintf("\t\t_ = vr\n"))
	} else {
		builder.WriteString(fmt.Sprintf("\t\tvr.BindRoot(%s)\n", gb.writeValue(entry.Val().(*lang.Var).Deref())))
	}
	builder.WriteString(fmt.Sprintf("\t}\n"))
}

func (gb *NSGoBuilder) writeValue(v interface{}) (ret string) {
	if m, ok := v.(lang.IObj); ok {
		meta := m.Meta()
		if !lang.IsNil(meta) {
			defer func() {
				ret = fmt.Sprintf("lang.WithMeta(%s, %s)", ret, gb.writeValue(meta))
			}()
		}
	}

	switch v := v.(type) {
	case lang.IPersistentMap:
		return gb.writeMap(v)
	case *lang.EmptyList:
		return "lang.NewList()"
	case lang.Keyword:
		return fmt.Sprintf("lang.NewKeyword(%q)", v.String()[1:])
	case *lang.Symbol:
		return fmt.Sprintf("lang.NewSymbol(%q)", v.FullName())
	case string:
		return fmt.Sprintf("%q", v)
	case *runtime.Fn:
		return gb.writeFn(v)
	case nil:
		return "nil"
	case int64:
		return fmt.Sprintf("int64(%d)", v)
	case int:
		return fmt.Sprintf("int(%d)", v)
	case lang.Char:
		return fmt.Sprintf("lang.NewChar(%q)", rune(v))
	case bool:
		if v {
			return "true"
		} else {
			return "false"
		}
	case *os.File:
		return gb.writeFile(v)
	case *lang.Atom:
		return gb.writeAtom(v)
	case *lang.Set:
		return gb.writeSet(v)
	case *lang.Ref:
		return gb.writeRef(v)
	case *lang.Namespace:
		return fmt.Sprintf("lang.FindOrCreateNamespace(lang.NewSymbol(%q))", v.Name().String())
	case *lang.MultiFn:
		return gb.writeMultiFn(v)
	case *lang.LazySeq:
		return gb.writeLazySeq(v)
	case reflect.Type:
		return gb.writeType(v)
	default:
		panic(fmt.Errorf("unsupported type: %T", v))
	}
}

func (gb *NSGoBuilder) writeMap(m lang.IPersistentMap) string {
	builder := &strings.Builder{}
	builder.WriteString("lang.NewPersistentArrayMap(\n")
	for s := lang.Seq(m); s != nil; s = s.Next() {
		entry := s.First().(lang.IMapEntry)
		builder.WriteString(gb.writeValue(entry.Key()))
		builder.WriteString(", ")
		builder.WriteString(gb.writeValue(entry.Val()))
		builder.WriteString(",\n")
	}
	builder.WriteString(")")
	return builder.String()
}

func (gb *NSGoBuilder) writeAtom(a *lang.Atom) string {
	return fmt.Sprintf("lang.NewAtom(%s)", gb.writeValue(a.Deref()))
}

func (gb *NSGoBuilder) writeRef(r *lang.Ref) string {
	return fmt.Sprintf("lang.NewRef(%s)", gb.writeValue(r.Deref()))
}

func (gb *NSGoBuilder) writeSet(s *lang.Set) string {
	builder := &strings.Builder{}
	builder.WriteString("lang.NewPersistentHashSet(\n")
	for s := lang.Seq(s); s != nil; s = s.Next() {
		builder.WriteString(gb.writeValue(s.First()))
		builder.WriteString(",\n")
	}
	builder.WriteString(")")
	return builder.String()
}

func (gb *NSGoBuilder) writeFile(f *os.File) string {
	switch f {
	case os.Stdin:
		return "os.Stdin"
	case os.Stdout:
		return "os.Stdout"
	case os.Stderr:
		return "os.Stderr"
	default:
		panic(fmt.Errorf("unsupported file: %v", f))
	}
}

var todo = "lang.NewSymbol(\"todo\")"

func (gb *NSGoBuilder) writeFn(fn *runtime.Fn) string {
	return todo
}

func (gb *NSGoBuilder) writeMultiFn(mf *lang.MultiFn) string {
	return todo
}

func (gb *NSGoBuilder) writeLazySeq(ls *lang.LazySeq) string {
	return todo
}

func (gb *NSGoBuilder) writeType(t reflect.Type) string {
	return todo
}
