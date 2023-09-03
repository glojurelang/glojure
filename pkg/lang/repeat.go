package lang

type Repeat struct {
	meta         IPersistentMap
	hash, hasheq uint32

	x     interface{}
	count int64
	next  ISeq
}

var (
	_ ASeq        = (*Repeat)(nil)
	_ ISeq        = (*Repeat)(nil)
	_ Sequential  = (*Repeat)(nil)
	_ IReduce     = (*Repeat)(nil)
	_ IReduceInit = (*Repeat)(nil)
)

func NewRepeat(x interface{}) *Repeat {
	return &Repeat{x: x, count: -1}
}

func NewRepeatN(count int64, x interface{}) ISeq {
	if count <= 0 {
		return emptyList
	}
	return &Repeat{x: x, count: count}
}

func (r *Repeat) Meta() IPersistentMap {
	return r.meta
}

func (r *Repeat) WithMeta(meta IPersistentMap) any {
	if meta == r.meta {
		return r
	}

	cpy := *r
	cpy.meta = meta
	return &cpy
}

func (r *Repeat) xxx_sequential() {}

func (r *Repeat) First() interface{} {
	return r.x
}

func (r *Repeat) More() ISeq {
	s := r.Next()
	if s == nil {
		return emptyList
	}
	return s
}

func (r *Repeat) Next() ISeq {
	if r.next != nil {
		return r.next
	}
	if r.count > 1 {
		r.next = NewRepeatN(r.count-1, r.x)
	} else if r.count == -1 {
		r.next = r
	}
	return r.next
}

func (r *Repeat) Seq() ISeq {
	return r
}

func (r *Repeat) Cons(val any) Conser {
	return aseqCons(r, val)
}

func (r *Repeat) Count() int {
	return aseqCount(r)
}

func (r *Repeat) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (r *Repeat) Equals(o any) bool {
	return aseqEquals(r, o)
}

func (r *Repeat) Equiv(o any) bool {
	return aseqEquiv(r, o)
}

func (r *Repeat) Hash() uint32 {
	return aseqHash(&r.hash, r)
}

func (r *Repeat) HashEq() uint32 {
	return aseqHashEq(&r.hasheq, r)
}

func (r *Repeat) String() string {
	return aseqString(r)
}

func (r *Repeat) Reduce(f IFn) interface{} {
	ret := r.x
	if r.count == -1 {
		for {
			ret = f.Invoke(ret, r.x)
			if IsReduced(ret) {
				return ret.(IDeref).Deref()
			}
		}
	} else {
		for i := int64(1); i < r.count; i++ {
			ret = f.Invoke(ret, r.x)
			if IsReduced(ret) {
				return ret.(IDeref).Deref()
			}
		}
		return ret
	}
}

func (r *Repeat) ReduceInit(f IFn, start interface{}) interface{} {
	ret := start
	if r.count == -1 {
		for {
			ret = f.Invoke(ret, r.x)
			if IsReduced(ret) {
				return ret.(IDeref).Deref()
			}
		}
	} else {
		for i := int64(0); i < r.count; i++ {
			ret = f.Invoke(ret, r.x)
			if IsReduced(ret) {
				return ret.(IDeref).Deref()
			}
		}
		return ret
	}
}
