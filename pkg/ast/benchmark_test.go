package ast

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

type (
	A struct{}
	B struct{}
	C struct{}
	D struct{}
	E struct{}
	F struct{}

	TypCode int
	Generic struct {
		Typ TypCode
		V   interface{}
	}

	Ifc interface {
		Do()
	}
)

func (a *A) Do() {}
func (b *B) Do() {}
func (c *C) Do() {}
func (d *D) Do() {}
func (e *E) Do() {}
func (f *F) Do() {}

const (
	TypA TypCode = iota
	TypB
	TypC
	TypD
	TypE
	TypF
)

var (
	Items = []Ifc{
		&A{},
		&B{},
		&C{},
		&D{},
		&E{},
		&F{},
	}
	GenericItems = func() []*Generic {
		items := make([]*Generic, len(Items))
		for i, item := range Items {
			var typ TypCode
			switch item.(type) {
			case *A:
				typ = TypA
			case *B:
				typ = TypB
			case *C:
				typ = TypC
			case *D:
				typ = TypD
			case *E:
				typ = TypE
			case *F:
				typ = TypF
			}
			items[i] = &Generic{Typ: typ, V: item}
		}
		return items
	}()

	KWA = lang.NewKeyword("A")
	KWB = lang.NewKeyword("B")
	KWC = lang.NewKeyword("C")
	KWD = lang.NewKeyword("D")
	KWE = lang.NewKeyword("E")
	KWF = lang.NewKeyword("F")

	MapItems = func() []lang.IPersistentMap {
		items := make([]lang.IPersistentMap, len(Items))
		for i, item := range Items {
			var op lang.Keyword
			switch item.(type) {
			case *A:
				op = KWA
			case *B:
				op = KWB
			case *C:
				op = KWC
			case *D:
				op = KWD
			case *E:
				op = KWE
			case *F:
				op = KWF
			}
			items[i] = lang.NewMap(lang.KWOp, op)
		}
		return items
	}()
)

func BenchmarkIntSwitch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, item := range GenericItems {
			switch item.Typ {
			case TypA:
			case TypB:
			case TypC:
			case TypD:
			case TypE:
			case TypF:
			default:
				panic("unreachable")
			}
		}
	}
}

func BenchmarkTypeSwitch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, item := range Items {
			switch item.(type) {
			case *A:
			case *B:
			case *C:
			case *D:
			case *E:
			case *F:
			default:
				panic("unreachable")
			}
		}
	}
}

func BenchmarkMapSwitch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, item := range MapItems {
			switch lang.Get(item, lang.KWOp) {
			case KWA:
			case KWB:
			case KWC:
			case KWD:
			case KWE:
			case KWF:
			default:
				panic("unreachable")
			}
		}
	}
}

func BenchmarkIfc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, item := range Items {
			item.Do()
		}
	}
}
