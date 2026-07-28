package lang

import (
	"errors"
	"math"
)

const longRangeChunkSize = 32

type (
	LongRange struct {
		meta   IPersistentMap
		hash   uint32
		hashEq uint32

		start, end, step int64
		count            int
	}

	LongChunk struct {
		start, step int64
		count       int
	}
)

var (
	_ ISeq               = (*LongRange)(nil)
	_ Sequential         = (*LongRange)(nil)
	_ IReduce            = (*LongRange)(nil)
	_ IReduceInit        = (*LongRange)(nil)
	_ Int64StepReducible = (*LongRange)(nil)
	_ ASeq               = (*LongRange)(nil)
	_ IDrop              = (*LongRange)(nil)
	_ IChunkedSeq        = (*LongRange)(nil)
	_ Counted            = (*LongRange)(nil)

	_ IChunk      = (*LongChunk)(nil)
	_ IReduceInit = (*LongChunk)(nil)
)

// NewLongRange returns a lazy sequence of start, start + step, start + 2*step, ...
func NewLongRange(start, end, step int64) (res ISeq) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(error); ok && errors.Is(err, NewArithmeticError("")) {
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
	count := Add(Add(Sub(end, start), step), o).(int64) / step
	if count > math.MaxInt {
		panic(NewArithmeticError("integer overflow"))
	}
	return int(count)
}

func (r *LongRange) xxx_sequential() {}

func (r *LongRange) Seq() ISeq {
	return r
}

func (r *LongRange) First() any {
	return r.start
}

func (r *LongRange) Next() ISeq {
	next := r.start + r.step
	if next >= r.end {
		return nil
	}
	return &LongRange{start: next, end: r.end, step: r.step, count: r.count - 1}
}

func (r *LongRange) More() ISeq {
	nxt := r.Next()
	if nxt == nil {
		return emptyList
	}
	return nxt
}

func (r *LongRange) ChunkedFirst() IChunk {
	return NewLongChunk(r.start, r.step, min(r.count, longRangeChunkSize))
}

func (r *LongRange) ChunkedNext() ISeq {
	more := r.ChunkedMore()
	if more == emptyList {
		return nil
	}
	return more
}

func (r *LongRange) ChunkedMore() ISeq {
	chunkCount := min(r.count, longRangeChunkSize)
	if chunkCount == r.count {
		return emptyList
	}
	return &LongRange{
		meta:  r.meta,
		start: r.start + int64(chunkCount)*r.step,
		end:   r.end,
		step:  r.step,
		count: r.count - chunkCount,
	}
}

func (r *LongRange) Cons(o any) Conser {
	return aseqCons(r, o)
}

func (r *LongRange) Count() int {
	return r.count
}

func (r *LongRange) xxx_counted() {}

func (r *LongRange) Empty() IPersistentCollection {
	return aseqEmpty()
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

func (r *LongRange) Reduce(f IFn) any {
	if reducer, ok := f.(Int64Reducer); ok {
		ret := r.start
		for i := r.start + r.step; i < r.end; i += r.step {
			ret = reducer.ReduceInt64(ret, i)
		}
		return ret
	}
	var ret any = r.start
	for i := r.start + r.step; i < r.end; i += r.step {
		ret = Apply2(f, ret, i)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (r *LongRange) ReduceInit(f IFn, init any) any {
	if reducer, ok := f.(Int64ReductionStepper); ok {
		if ret, ok := init.(int64); ok {
			return r.ReduceInt64Steps(reducer, ret)
		}
	}
	if reducer, ok := f.(Int64Reducer); ok {
		if ret, ok := init.(int64); ok {
			for i := r.start; i < r.end; i += r.step {
				ret = reducer.ReduceInt64(ret, i)
			}
			return ret
		}
	}
	var ret any = init
	for i := r.start; i < r.end; i += r.step {
		ret = Apply2(f, ret, i)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (r *LongRange) ReduceInt64Steps(
	reducer Int64ReductionStepper,
	initial int64,
) int64 {
	result := initial
	value := r.start
	for index := 0; index < r.count; index++ {
		var reduced bool
		result, reduced = reducer.ReduceInt64Step(result, value)
		if reduced {
			return result
		}
		if index+1 < r.count {
			value += r.step
		}
	}
	return result
}

func (r *LongRange) Drop(n int) Sequential {
	if n < 0 {
		return r
	}
	if n < r.count {
		return NewLongRange(r.start+int64(n)*r.step, r.end, r.step).(Sequential)
	} else {
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// LongChunk

func NewLongChunk(start, step int64, count int) *LongChunk {
	return &LongChunk{
		start: start,
		step:  step,
		count: count,
	}
}

func (c *LongChunk) First() any {
	return c.start
}

func (c *LongChunk) Nth(i int) any {
	return c.start + int64(i)*c.step
}

func (c *LongChunk) NthDefault(i int, notFound any) any {
	if i >= 0 && i < c.count {
		return c.start + int64(i)*c.step
	}
	return notFound
}

func (c *LongChunk) Count() int {
	return c.count
}

func (c *LongChunk) fieldOrMethod(name string) (interface{}, bool) {
	switch name {
	case "nth", "Nth":
		return FnFunc1(func(i any) any {
			return c.Nth(MustAsInt(i))
		}), true
	case "nthDefault", "NthDefault":
		return FnFunc2(func(i, notFound any) any {
			return c.NthDefault(MustAsInt(i), notFound)
		}), true
	case "count", "Count":
		return FnFunc0(func() any { return c.Count() }), true
	}
	return nil, false
}

func (c *LongChunk) xxx_counted() {}

func (c *LongChunk) DropFirst() IChunk {
	if c.count <= 0 {
		panic(NewIllegalStateError("dropFirst of empty chunk"))
	}
	return NewLongChunk(c.start+c.step, c.step, c.count-1)
}

func (c *LongChunk) ReduceInit(f IFn, init any) any {
	x := c.start
	ret := init
	for i := 0; i < c.count; i++ {
		ret = Apply2(f, ret, x)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
		x += c.step
	}
	return ret
}
