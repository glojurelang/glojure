//go:build glj_no_aot_stdlib

package glj

import (
	"go/ast"
	"go/parser"
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestBootstrapRegistersCoreTypes(t *testing.T) {
	types := map[string]reflect.Type{
		"clojure.lang.ChunkBuffer":                               reflect.TypeOf((*lang.ChunkBuffer)(nil)),
		"clojure.lang.IChunk":                                    reflect.TypeOf((*lang.IChunk)(nil)).Elem(),
		"github.com/glojurelang/glojure/pkg/lang.*TaggedLiteral": reflect.TypeOf((*lang.TaggedLiteral)(nil)),
	}
	for name, want := range types {
		got, ok := pkgmap.Get(name)
		if !ok {
			t.Errorf("bootstrap package map does not contain %s", name)
			continue
		}
		if got != want {
			t.Errorf("bootstrap package map %s = %v, want %v", name, got, want)
		}
	}
}

func TestBootstrapRegistersGoTypesDependencies(t *testing.T) {
	types := map[string]reflect.Type{
		"go/ast.*Ident":      reflect.TypeOf((*ast.Ident)(nil)),
		"go/ast.*ArrayType":  reflect.TypeOf((*ast.ArrayType)(nil)),
		"go/ast.*MapType":    reflect.TypeOf((*ast.MapType)(nil)),
		"go/ast.*ChanType":   reflect.TypeOf((*ast.ChanType)(nil)),
		"go/ast.*FuncType":   reflect.TypeOf((*ast.FuncType)(nil)),
		"go/ast.*Ellipsis":   reflect.TypeOf((*ast.Ellipsis)(nil)),
		"go/ast.*StructType": reflect.TypeOf((*ast.StructType)(nil)),
	}
	for name, want := range types {
		got, ok := pkgmap.Get(name)
		if !ok {
			t.Errorf("bootstrap package map does not contain %s", name)
			continue
		}
		if got != want {
			t.Errorf("bootstrap package map %s = %v, want %v", name, got, want)
		}
	}

	constants := map[string]ast.ChanDir{
		"go/ast.SEND": ast.SEND,
		"go/ast.RECV": ast.RECV,
	}
	for name, want := range constants {
		got, ok := pkgmap.Get(name)
		if !ok {
			t.Errorf("bootstrap package map does not contain %s", name)
			continue
		}
		if got != want {
			t.Errorf("bootstrap package map %s = %v, want %v", name, got, want)
		}
	}

	got, ok := pkgmap.Get("go/parser.ParseExpr")
	if !ok {
		t.Fatal("bootstrap package map does not contain go/parser.ParseExpr")
	}
	if reflect.ValueOf(got).Pointer() != reflect.ValueOf(parser.ParseExpr).Pointer() {
		t.Fatal("bootstrap package map contains the wrong go/parser.ParseExpr function")
	}
}
