package lang

type LongRange struct {
	start, end, step int64
}

var (
	_ ISeq        = (*LongRange)(nil)
	_ Sequential  = (*LongRange)(nil)
	_ IReduce     = (*LongRange)(nil)
	_ IReduceInit = (*LongRange)(nil)
)

// NewLongRange returns a lazy sequence of start, start + step, start + 2*step, ...
func NewLongRange(start, end, step int64) ISeq {
	if end <= start {
		return emptyList
	}

	return &LongRange{start: start, end: end, step: step}
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
