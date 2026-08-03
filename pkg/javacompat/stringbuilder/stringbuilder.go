// Package stringbuilder exposes java.lang.StringBuilder.
package stringbuilder

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type StringBuilder struct {
	value string
}

func New(args ...any) any {
	builder := &StringBuilder{}
	if len(args) > 1 {
		panic(fmt.Sprintf("StringBuilder/new: wrong number of args (%d)", len(args)))
	}
	if len(args) == 1 {
		builder.value = lang.ToString(args[0])
	}
	return builder
}

func (b *StringBuilder) Append(value any) *StringBuilder {
	b.value += lang.ToString(value)
	return b
}

func (b *StringBuilder) AppendCodePoint(value any) *StringBuilder {
	b.value += string(rune(lang.MustAsInt(value)))
	return b
}

func (b *StringBuilder) Write(value []byte) (int, error) {
	b.value += string(value)
	return len(value), nil
}

func (b *StringBuilder) Length() int { return len([]rune(b.value)) }

func (b *StringBuilder) CharAt(index any) lang.Char {
	return lang.NewChar([]rune(b.value)[lang.MustAsInt(index)])
}

func (b *StringBuilder) SetLength(length any) {
	n := lang.MustAsInt(length)
	runes := []rune(b.value)
	if n <= len(runes) {
		b.value = string(runes[:n])
		return
	}
	b.value += strings.Repeat("\x00", n-len(runes))
}

func (b *StringBuilder) ToString() string { return b.value }
func (b *StringBuilder) String() string   { return b.value }

func init() {
	pkgmap.SetHostClassPackage("StringBuilder", "java.lang")
	pkgmap.SetHostClass("StringBuilder",
		lang.NewClass(reflect.TypeOf((*StringBuilder)(nil)), "java.lang.StringBuilder"))
	lang.RegisterHostConstructor("java.lang.StringBuilder",
		lang.FnFunc(func(args ...any) any { return New(args...) }))
}
