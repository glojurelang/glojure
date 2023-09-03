package lang

type (
	boundsCheck interface {
		ExceededBounds(val any) bool
	}

	Range struct {
		meta         IPersistentMap
		hash, hasheq uint32

		start, end, step any
		boundsCheck      boundsCheck

		chunk     IChunk // lazy
		chunkNext ISeq   // lazy
		next      ISeq   // cached
	}

	positiveStep struct {
		end any
	}

	negativeStep struct {
		end any
	}
)

var (
	_ ASeq        = (*Range)(nil)
	_ Sequential  = (*Range)(nil)
	_ IReduce     = (*Range)(nil)
	_ IReduceInit = (*Range)(nil)
	_ IChunkedSeq = (*Range)(nil)
)

func (ps positiveStep) ExceededBounds(val any) bool {
	return Numbers.Gte(val, ps.end)
}

func (ns negativeStep) ExceededBounds(val any) bool {
	return Numbers.Lte(val, ns.end)
}

// NewRange returns a lazy sequence of start, start + step, start + 2*step, ...
func NewRange(start, end, step any) ISeq {
	if (Numbers.IsPos(step) && Numbers.Gt(start, end)) ||
		(Numbers.IsNeg(step) && Numbers.Gt(end, start)) ||
		Numbers.Equiv(start, end) {
		return emptyList
	}
	if Numbers.IsZero(step) {
		return NewRepeat(start)
	}

	var bc boundsCheck
	if Numbers.IsPos(step) {
		bc = positiveStep{end: end}
	} else {
		bc = negativeStep{end: end}
	}

	return &Range{
		start:       start,
		end:         end,
		step:        step,
		boundsCheck: bc,
	}
}

func (r *Range) xxx_sequential() {}

func (r *Range) Meta() IPersistentMap {
	return r.meta
}

func (r *Range) WithMeta(meta IPersistentMap) any {
	if meta == r.meta {
		return r
	}
	rng := *r
	rng.meta = meta
	return &rng
}

func (r *Range) Seq() ISeq {
	return r
}

func (r *Range) Cons(val any) Conser {
	return aseqCons(r, val)
}

func (r *Range) First() any {
	return r.start
}

func (r *Range) More() ISeq {
	return aseqMore(r)
}

func (r *Range) Count() int {
	return aseqCount(r)
}

func (r *Range) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (r *Range) Equals(o any) bool {
	return aseqEquals(r, o)
}

func (r *Range) Equiv(o any) bool {
	return aseqEquiv(r, o)
}

func (r *Range) Hash() uint32 {
	return aseqHash(&r.hash, r)
}

func (r *Range) HashEq() uint32 {
	return aseqHashEq(&r.hasheq, r)
}

func (r *Range) String() string {
	return aseqString(r)
}

func (r *Range) ForceChunk() {
	if r.chunk != nil {
		return
	}

	const chunkSize = 32

	arr := [chunkSize]any{}
	n := 0
	val := r.start
	for n < chunkSize {
		arr[n] = val
		n++
		val = Numbers.AddP(val, r.step)
		if r.boundsCheck.ExceededBounds(val) {
			r.chunk = NewSliceChunk(arr[:n])
			return
		}
	}
	if r.boundsCheck.ExceededBounds(val) {
		r.chunk = NewSliceChunk(arr[:n])
		return
	}

	r.chunk = NewSliceChunk(arr[:chunkSize])
	r.chunkNext = NewRange(val, r.end, r.step)
}

func (r *Range) Next() ISeq {
	if r.next != nil {
		return r.next
	}

	r.ForceChunk()
	if r.chunk.Count() > 1 {
		smallerChunk := r.chunk.DropFirst()
		r.next = &Range{
			start:       smallerChunk.Nth(0),
			end:         r.end,
			step:        r.step,
			boundsCheck: r.boundsCheck,
			chunk:       smallerChunk,
			chunkNext:   r.chunkNext,
		}
		return r.next
	}
	return r.ChunkedNext()
}

func (r *Range) ChunkedFirst() IChunk {
	r.ForceChunk()
	return r.chunk
}

func (r *Range) ChunkedNext() ISeq {
	return r.ChunkedMore().Seq()
}

func (r *Range) ChunkedMore() ISeq {
	r.ForceChunk()
	if r.chunkNext == nil {
		return emptyList
	}
	return r.chunkNext
}

func (r *Range) Reduce(f IFn) any {
	acc := r.start
	i := Numbers.AddP(r.start, r.step)
	for !r.boundsCheck.ExceededBounds(i) {
		acc = f.Invoke(acc, i)
		if IsReduced(acc) {
			return acc.(*Reduced).Deref()
		}
		i = Numbers.AddP(i, r.step)
	}
	return acc
}

func (r *Range) ReduceInit(f IFn, init any) any {
	acc := init
	i := r.start
	for !r.boundsCheck.ExceededBounds(i) {
		acc = f.Invoke(acc, i)
		if IsReduced(acc) {
			return acc.(*Reduced).Deref()
		}
		i = Numbers.AddP(i, r.step)
	}
	return acc
}
