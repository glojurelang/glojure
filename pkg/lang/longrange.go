package lang

import (
	"errors"
	"math"
)

type LongRange struct {
	meta   IPersistentMap
	hash   uint32
	hashEq uint32

	start, end, step int64
	count            int
}

var (
	_ ISeq        = (*LongRange)(nil)
	_ Sequential  = (*LongRange)(nil)
	_ IReduce     = (*LongRange)(nil)
	_ IReduceInit = (*LongRange)(nil)
	_ ASeq        = (*LongRange)(nil)
	_ IDrop       = (*LongRange)(nil)
	_ IChunkedSeq = (*LongRange)(nil)
)

// NewLongRange returns a lazy sequence of start, start + step, start + 2*step, ...
func NewLongRange(start, end, step int64) (res ISeq) {
	defer func() {
		if err := recover(); err != nil {
			if errors.Is(err, NewArithmeticError("")) {
				res = NewRange(start, end, step)
				return
			}
			panic(err)
		}
	}()

	count := 0
	if step > 0 {
		if end <= start {
			return emptyList
		}
		count = rangeCount(start, end, step)
	} else if step < 0 {
		if end >= start {
			return emptyList
		}
		count = rangeCount(start, end, step)
	} else {
		if end == start {
			return emptyList
		}
		return NewRepeat(start)
	}

	return &LongRange{
		start: start,
		end:   end,
		step:  step,
		count: count,
	}
}

func rangeCount(start, end, step int64) int {
	// (1) count = ceiling ( (end - start) / step )
	// (2) ceiling(a/b) = (a+b+o)/b where o=-1 for positive stepping and +1 for negative stepping
	// thus: count = end - start + step + o / step

	o := int64(1)
	if step > 0 {
		o = -1
	}
	count := Add(Add(Minus(end, start), step), o) / step
	if count > math.MaxInt {
		panic(NewArithmeticError("integer overflow"))
	}
	return int(count)
}

func (r *LongRange) xxx_sequential() {}

func (r *LongRange) Seq() ISeq {
	return r
}

func (r *LongRange) First() interface{} {
	return r.start
}

func (r *LongRange) Next() ISeq {
	next := r.start + r.step
	if next >= r.end {
		return nil
	}
	return &LongRange{start: next, end: r.end, step: r.step}
}

func (r *LongRange) More() ISeq {
	nxt := r.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (r *LongRange) ChunkedFirst() IChunk {
	return NewLongChunk(r.start, r.step, r.count)
}

func (r *LongRange) ChunkedNext() ISeq {
	return nil
}

func (r *LongRange) ChunkedMore() ISeq {
	return emptyList
}

func (r *LongRange) Cons(o any) Conser {
	return aseqCons(r, o)
}

func (r *LongRange) Count() int {
	return r.count
}

func (r *LongRange) Empty() IPersistentCollection {
	return asetEmpty()
}

func (r *LongRange) Equiv(o any) bool {
	return aseqEquiv(r, o)
}

func (r *LongRange) Equals(o any) bool {
	return aseqEquals(r, o)
}

func (r *LongRange) Hash() uint32 {
	return aseqHash(&r.hash, r)
}

func (r *LongRange) HashEq() uint32 {
	return aseqHashEq(&r.hashEq, r)
}

func (r *LongRange) String() string {
	return aseqString(r)
}

func (r *LongRange) Meta() IPersistentMap {
	return r.meta
}

func (r *LongRange) WithMeta(meta IPersistentMap) any {
	if r.meta == meta {
		return r
	}
	return &LongRange{
		meta:   meta,
		hash:   r.hash,
		hashEq: r.hashEq,
		start:  r.start,
		end:    r.end,
		step:   r.step,
		count:  r.count,
	}
}

////////////////////////////////////////////////////////////////////////////////

func (r *LongRange) Reduce(f IFn) interface{} {
	var ret interface{} = r.start
	for i := r.start + r.step; i < r.end; i += r.step {
		ret = f.Invoke(ret, i)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (r *LongRange) ReduceInit(f IFn, start interface{}) interface{} {
	var ret interface{} = start
	for i := r.start; i < r.end; i += r.step {
		ret = f.Invoke(ret, i)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (r *LongRange) Drop(n int) Sequential {
	if n < 0 {
		return r
	}
	if n < r.count {
		return NewLongRange(r.start+n*r.step, r.end, r.step)
	} else {
		return nil
	}
}
