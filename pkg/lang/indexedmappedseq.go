package lang

import (
	"strings"
	"sync"
)

// indexedMappedState retains realized values instead of a linked chain of
// lazy sequence nodes. This lets representation-aware consumers traverse the
// sequence directly while preserving the ordinary lazy-sequence guarantee
// that user code is invoked at most once for each retained element.
type indexedMappedState struct {
	mutex sync.Mutex

	fn        any
	source    ISeq
	nextIndex any
	values    []any
}

// indexedMappedSeq represents the two-argument form of map-indexed over an
// already-materialized, non-chunked sequence.
type indexedMappedSeq struct {
	meta         IPersistentMap
	hash, hasheq uint32

	state    *indexedMappedState
	position int

	nextOnce sync.Once
	next     ISeq
}

var (
	_ ASeq        = (*indexedMappedSeq)(nil)
	_ IPending    = (*indexedMappedSeq)(nil)
	_ IReduce     = (*indexedMappedSeq)(nil)
	_ IReduceInit = (*indexedMappedSeq)(nil)
)

// NewIndexedMappedSeq returns a lazy indexed mapping beginning at index zero.
func NewIndexedMappedSeq(fn any, source ISeq) ISeq {
	if source == nil {
		return nil
	}
	return &indexedMappedSeq{
		state: &indexedMappedState{
			fn:        fn,
			source:    source,
			nextIndex: int64(0),
		},
	}
}

func (state *indexedMappedState) valueAt(position int) (any, bool) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	for len(state.values) <= position {
		if state.source == nil {
			return nil, false
		}
		value := Apply2(
			state.fn,
			state.nextIndex,
			state.source.First(),
		)
		state.values = append(state.values, value)
		state.nextIndex = Numbers.Inc(state.nextIndex)
		state.source = state.source.Next()
	}
	return state.values[position], true
}

func (state *indexedMappedState) hasPosition(position int) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	return position < len(state.values) || state.source != nil
}

func (sequence *indexedMappedSeq) value() any {
	value, ok := sequence.state.valueAt(sequence.position)
	if !ok {
		panic(NewIllegalStateError("indexed mapped sequence exhausted"))
	}
	return value
}

func (*indexedMappedSeq) xxx_sequential() {}

func (sequence *indexedMappedSeq) Seq() ISeq {
	sequence.value()
	return sequence
}

func (sequence *indexedMappedSeq) First() any {
	return sequence.value()
}

func (sequence *indexedMappedSeq) Next() ISeq {
	sequence.value()
	sequence.nextOnce.Do(func() {
		position := sequence.position + 1
		if sequence.state.hasPosition(position) {
			sequence.next = &indexedMappedSeq{
				state:    sequence.state,
				position: position,
			}
		}
	})
	return sequence.next
}

func (sequence *indexedMappedSeq) More() ISeq {
	return aseqMore(sequence)
}

func (sequence *indexedMappedSeq) Count() int {
	return aseqCount(sequence)
}

func (sequence *indexedMappedSeq) Cons(value any) Conser {
	return aseqCons(sequence, value)
}

func (sequence *indexedMappedSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (sequence *indexedMappedSeq) Equals(other any) bool {
	return aseqEquals(sequence, other)
}

func (sequence *indexedMappedSeq) Equiv(other any) bool {
	return aseqEquiv(sequence, other)
}

func (sequence *indexedMappedSeq) Meta() IPersistentMap {
	return sequence.meta
}

func (sequence *indexedMappedSeq) WithMeta(meta IPersistentMap) any {
	if meta == sequence.meta {
		return sequence
	}
	return newLazySeqWithMeta(meta, sequence.Seq())
}

func (sequence *indexedMappedSeq) Hash() uint32 {
	return aseqHash(&sequence.hash, sequence)
}

func (sequence *indexedMappedSeq) HashEq() uint32 {
	return aseqHashEq(&sequence.hasheq, sequence)
}

func (sequence *indexedMappedSeq) String() string {
	return aseqString(sequence)
}

func (sequence *indexedMappedSeq) IsRealized() bool {
	sequence.state.mutex.Lock()
	defer sequence.state.mutex.Unlock()
	return sequence.position < len(sequence.state.values)
}

// AppendStrings is an optional representation-aware path for apply str. It
// stores each mapped result in the shared realization cache, preserving later
// traversal semantics without allocating a lazy sequence node per element.
func (sequence *indexedMappedSeq) AppendStrings(builder *strings.Builder) {
	for position := sequence.position; ; position++ {
		value, ok := sequence.state.valueAt(position)
		if !ok {
			return
		}
		if value != nil {
			builder.WriteString(ToString(value))
		}
	}
}

func (sequence *indexedMappedSeq) Reduce(reducer IFn) any {
	result := sequence.value()
	for position := sequence.position + 1; ; position++ {
		value, ok := sequence.state.valueAt(position)
		if !ok {
			return result
		}
		result = Apply2(reducer, result, value)
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
}

func (sequence *indexedMappedSeq) ReduceInit(reducer IFn, initial any) any {
	result := initial
	for position := sequence.position; ; position++ {
		value, ok := sequence.state.valueAt(position)
		if !ok {
			return result
		}
		result = Apply2(reducer, result, value)
		if IsReduced(result) {
			return result.(IDeref).Deref()
		}
	}
}
