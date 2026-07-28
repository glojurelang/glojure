package lang

import "testing"

var boxedCharSink any
var stringReduceSink any

func TestBoxCharPreservesCharValues(t *testing.T) {
	for _, value := range []rune{0, 'A', 255, 'λ'} {
		boxed := BoxChar(value)
		if got, ok := boxed.(Char); !ok || got != Char(value) {
			t.Fatalf("BoxChar(%d) = %#v, want Char(%d)", value, boxed, value)
		}
	}
}

func TestBoxCharCachesStringByteValues(t *testing.T) {
	allocations := testing.AllocsPerRun(1000, func() {
		boxedCharSink = BoxChar(255)
	})
	if allocations != 0 {
		t.Fatalf("cached byte Char allocated %.2f times", allocations)
	}
}

func TestStringSeqReduceUsesFixedArityWithoutAllocating(t *testing.T) {
	sequence := NewStringSeq("ACGT", 0)
	initial := any(struct{}{})
	reducer := FnFunc2(func(acc, value any) any {
		if _, ok := value.(Char); !ok {
			t.Fatalf("string reduction value has type %T, want Char", value)
		}
		return acc
	})
	if got := sequence.ReduceInit(reducer, initial); got != initial {
		t.Fatalf("string reduction = %#v, want initial value", got)
	}

	allocations := testing.AllocsPerRun(1000, func() {
		stringReduceSink = sequence.ReduceInit(reducer, initial)
	})
	if allocations != 0 {
		t.Fatalf("fixed-arity string reduction allocated %.2f times", allocations)
	}
}
