package lang

import (
	"regexp"
	"testing"
)

func TestRegexpSeqReturnsScalarAndGroupedMatches(t *testing.T) {
	scalar := NewRegexpSeq(regexp.MustCompile(`[a-z]+`), "12ab34cd")
	if got := scalar.First(); got != "ab" {
		t.Fatalf("first scalar match = %v, want ab", got)
	}
	if got := scalar.Next().First(); got != "cd" {
		t.Fatalf("second scalar match = %v, want cd", got)
	}
	if scalar.Next().Next() != nil {
		t.Fatal("scalar sequence has an unexpected third match")
	}

	grouped := NewRegexpSeq(
		regexp.MustCompile(`(a)?(b)`),
		"ab b",
	)
	first := grouped.First().(IPersistentVector)
	if got := first.Nth(0); got != "ab" {
		t.Fatalf("first full match = %v, want ab", got)
	}
	if got := first.Nth(1); got != "a" {
		t.Fatalf("first optional group = %v, want a", got)
	}
	second := grouped.Next().First().(IPersistentVector)
	if got := second.Nth(1); got != nil {
		t.Fatalf("unmatched optional group = %v, want nil", got)
	}
	if got := second.Nth(2); got != "b" {
		t.Fatalf("second required group = %v, want b", got)
	}
}

func TestRegexpSeqCountScansWithoutMarkingSequenceCounted(t *testing.T) {
	sequence := NewRegexpSeq(regexp.MustCompile(`a*`), "ba")
	if _, ok := sequence.(Counted); ok {
		t.Fatal("RegexpSeq exposes constant-time Counted semantics")
	}
	if got := Count(sequence); got != 2 {
		t.Fatalf("regexp sequence count = %d, want 2", got)
	}
	next := sequence.Next()
	if next == nil || next.First() != "a" {
		t.Fatalf("sequence after count = %v, want a", next)
	}
	if sequence.Next() != next {
		t.Fatal("RegexpSeq did not cache its successor")
	}
}

func TestRegexpSeqCountAvoidsSequenceNodeAllocations(t *testing.T) {
	expression := regexp.MustCompile(`a`)
	input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	allocations := testing.AllocsPerRun(100, func() {
		if got := Count(NewRegexpSeq(expression, input)); got != len(input) {
			panic(got)
		}
	})
	if allocations > float64(len(input)+3) {
		t.Fatalf(
			"regexp sequence count allocated %.0f objects for %d matches",
			allocations,
			len(input),
		)
	}
}

func TestRegexpSeqUsesFixedASCIIPlanWithoutChangingValues(t *testing.T) {
	expression := regexp.MustCompile(`[cgt]gggtaaa|tttaccc[acg]`)
	input := "xxcgggtaaayytttacccazz"
	sequence := NewRegexpSeq(expression, input).(*RegexpSeq)
	if sequence.fixed == nil {
		t.Fatal("fixed ASCII re-seq did not retain an execution plan")
	}
	if got := sequence.First(); got != "cgggtaaa" {
		t.Fatalf("first fixed match = %v, want cgggtaaa", got)
	}
	if got := sequence.Next().First(); got != "tttaccca" {
		t.Fatalf("second fixed match = %v, want tttaccca", got)
	}
	if got := Count(sequence); got != 2 {
		t.Fatalf("fixed regexp count = %d, want 2", got)
	}
}
