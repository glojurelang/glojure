package lang

import (
	"regexp"
	"sync"
)

// fastCounter is an internal count path for sequential values that can count
// without materializing their ordinary sequence representation. It does not
// imply Clojure's Counted marker or constant-time behavior.
type fastCounter interface {
	countFast() int
}

// RegexpSeq is the lazy sequence returned by re-seq. Each node keeps the
// already-observed match immutable and computes its successor at most once.
// Count can scan the remaining input directly without allocating sequence
// nodes, while counted? remains false.
type RegexpSeq struct {
	meta         IPersistentMap
	hash, hasheq uint32

	expression *regexp.Regexp
	fixed      *fixedASCIIRegexp
	input      string
	match      []int
	offset     int
	nextStart  int
	first      any

	nextOnce sync.Once
	next     ISeq
}

var (
	_ ASeq        = (*RegexpSeq)(nil)
	_ ISeq        = (*RegexpSeq)(nil)
	_ Sequential  = (*RegexpSeq)(nil)
	_ IReduce     = (*RegexpSeq)(nil)
	_ IReduceInit = (*RegexpSeq)(nil)
	_ fastCounter = (*RegexpSeq)(nil)
)

// NewRegexpSeq returns successive matches with the same scalar-or-vector group
// representation as Clojure's re-seq.
func NewRegexpSeq(expression *regexp.Regexp, input string) ISeq {
	var fixed *fixedASCIIRegexp
	if expression.NumSubexp() == 0 {
		fixed = fixedASCIIRegexpPlan(expression)
	}
	return newRegexpSeq(expression, fixed, input, 0)
}

func newRegexpSeq(
	expression *regexp.Regexp,
	fixed *fixedASCIIRegexp,
	input string,
	start int,
) ISeq {
	if start > len(input) {
		return nil
	}
	var match []int
	if fixed != nil {
		matchStart, matchEnd, ok := fixed.find(input, start)
		if !ok {
			return nil
		}
		match = []int{matchStart - start, matchEnd - start}
	} else {
		match = expression.FindStringSubmatchIndex(input[start:])
		if match == nil {
			return nil
		}
	}
	return &RegexpSeq{
		expression: expression,
		fixed:      fixed,
		input:      input,
		match:      match,
		offset:     start,
		nextStart:  regexpNextStart(start, match, len(input)),
		first:      regexpGroups(input, start, match),
	}
}

func regexpNextStart(offset int, match []int, inputLength int) int {
	end := offset + match[1]
	if end == inputLength {
		return inputLength + 1
	}
	if match[0] == match[1] {
		return end + 1
	}
	return end
}

func regexpGroups(input string, offset int, match []int) any {
	if len(match) == 2 {
		return input[offset+match[0] : offset+match[1]]
	}
	groups := make([]any, len(match)/2)
	for index := range groups {
		start, end := match[index*2], match[index*2+1]
		if start >= 0 && end >= 0 {
			groups[index] = input[offset+start : offset+end]
		}
	}
	return NewVector(groups...)
}

func (sequence *RegexpSeq) Meta() IPersistentMap {
	return sequence.meta
}

func (sequence *RegexpSeq) WithMeta(meta IPersistentMap) any {
	if meta == sequence.meta {
		return sequence
	}
	return newLazySeqWithMeta(meta, sequence)
}

func (*RegexpSeq) xxx_sequential() {}

func (sequence *RegexpSeq) First() any {
	return sequence.first
}

func (sequence *RegexpSeq) Next() ISeq {
	sequence.nextOnce.Do(func() {
		sequence.next = newRegexpSeq(
			sequence.expression,
			sequence.fixed,
			sequence.input,
			sequence.nextStart,
		)
	})
	return sequence.next
}

func (sequence *RegexpSeq) More() ISeq {
	return aseqMore(sequence)
}

func (sequence *RegexpSeq) Seq() ISeq {
	return sequence
}

func (sequence *RegexpSeq) Cons(value any) Conser {
	return aseqCons(sequence, value)
}

func (sequence *RegexpSeq) Count() int {
	return aseqCount(sequence)
}

func (sequence *RegexpSeq) countFast() int {
	if sequence.fixed != nil {
		return 1 + sequence.fixed.count(
			sequence.input,
			sequence.nextStart,
		)
	}
	count := 1
	for start := sequence.nextStart; start <= len(sequence.input); count++ {
		match := sequence.expression.FindStringIndex(sequence.input[start:])
		if match == nil {
			return count
		}
		start = regexpNextStart(start, match, len(sequence.input))
	}
	return count
}

func (sequence *RegexpSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (sequence *RegexpSeq) Equals(other any) bool {
	return aseqEquals(sequence, other)
}

func (sequence *RegexpSeq) Equiv(other any) bool {
	return aseqEquiv(sequence, other)
}

func (sequence *RegexpSeq) Hash() uint32 {
	return aseqHash(&sequence.hash, sequence)
}

func (sequence *RegexpSeq) HashEq() uint32 {
	return aseqHashEq(&sequence.hasheq, sequence)
}

func (sequence *RegexpSeq) String() string {
	return aseqString(sequence)
}

func (sequence *RegexpSeq) Reduce(reducer IFn) any {
	result := sequence.First()
	for next := sequence.Next(); next != nil; next = next.Next() {
		result = Apply2(reducer, result, next.First())
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
	return result
}

func (sequence *RegexpSeq) ReduceInit(reducer IFn, initial any) any {
	result := initial
	for current := ISeq(sequence); current != nil; current = current.Next() {
		result = Apply2(reducer, result, current.First())
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
	return result
}
