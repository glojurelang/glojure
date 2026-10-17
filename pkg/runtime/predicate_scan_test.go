package runtime

import (
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

type predicateSequenceSource struct{ values []any }

func (s *predicateSequenceSource) Seq() lang.ISeq        { return lang.NewList(s.values...) }
func (*predicateSequenceSource) Count() int              { panic("custom indexed count must not replace Seq") }
func (*predicateSequenceSource) Nth(int) any             { panic("custom indexed access must not replace Seq") }
func (*predicateSequenceSource) NthDefault(int, any) any { panic("custom indexed access") }

func TestPredicateScanTruthinessAndShortCircuit(t *testing.T) {
	values := []any{false, nil, int64(0), true}
	for _, source := range []any{lang.NewVector(values...), lang.NewList(values...),
		&predicateSequenceSource{values}, lang.NewLazySeq(func() any { return lang.NewList(values...) })} {
		var seen []any
		pred := lang.FnFunc1(func(value any) any { seen = append(seen, value); return value })
		value, found := scanPredicate(pred, source, true)
		if !found || value != int64(0) || !reflect.DeepEqual(seen, values[:3]) {
			t.Fatalf("truthy scan of %T = %v, %v, seen %v", source, value, found, seen)
		}
		seen = nil
		value, found = scanPredicate(pred, source, false)
		if !found || value != false || len(seen) != 1 {
			t.Fatalf("false scan of %T = %v, %v, seen %v", source, value, found, seen)
		}
	}
	for _, source := range []any{nil, lang.NewVector(), lang.NewList()} {
		for _, truthy := range []bool{false, true} {
			_, found := scanPredicate(lang.FnFunc1(func(any) any { panic("empty input called predicate") }), source, truthy)
			if found {
				t.Fatal("empty input produced a matching predicate value")
			}
		}
	}
}

func TestPredicateScanStopsBeforeLazyInputTail(t *testing.T) {
	source := lang.NewCons(42, lang.NewLazySeq(func() any { panic("tail forced") }))
	if value, found := scanPredicate(lang.FnFunc1(func(value any) any { return value }), source, true); !found || value != 42 {
		t.Fatal("predicate did not stop at its first truthy value")
	}
}
