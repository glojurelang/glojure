package lang

type (
	Iterate struct {
		meta         IPersistentMap
		hash, hasheq uint32

		f IFn

		prevSeed any
		// lazily realized
		seed any

		// cached
		next ISeq
	}
)

var (
	_ ASeq        = (*Iterate)(nil)
	_ IReduce     = (*Iterate)(nil)
	_ IReduceInit = (*Iterate)(nil)
	_ IPending    = (*Iterate)(nil)

	unrealizedSeed = &struct{}{}
)

func CreateIterate(f IFn, seed any) *Iterate {
	return newIterate(f, nil, seed)
}

func newIterate(f IFn, prevSeed, seed any) *Iterate {
	return &Iterate{
		f:        f,
		prevSeed: prevSeed,
		seed:     seed,
	}
}

func newIterateMeta(meta IPersistentMap, f IFn, prevSeed, seed any, next ISeq) *Iterate {
	return &Iterate{
		f:        f,
		prevSeed: prevSeed,
		seed:     seed,
		next:     next,
		meta:     meta,
	}
}

func (it *Iterate) IsRealized() bool {
	return it.seed != unrealizedSeed
}

func (it *Iterate) First() any {
	if it.seed == unrealizedSeed {
		it.seed = it.f.Invoke(it.prevSeed)
	}
	return it.seed
}

func (it *Iterate) Next() ISeq {
	if IsNil(it.next) {
		it.next = newIterate(it.f, it.First(), unrealizedSeed)
	}
	return it.next
}

func (it *Iterate) Meta() IPersistentMap {
	return it.meta
}

func (it *Iterate) WithMeta(meta IPersistentMap) any {
	if it.meta == meta {
		return it
	}

	return newIterateMeta(meta, it.f, it.prevSeed, it.seed, it.next)
}

func (it *Iterate) Reduce(rf IFn) any {
	first := it.First()
	ret := first
	v := it.f.Invoke(first)
	for {
		ret = rf.Invoke(ret, v)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
		v = it.f.Invoke(v)
	}
}

func (it *Iterate) ReduceInit(rf IFn, start any) any {
	ret := start
	v := it.First()
	for {
		ret = rf.Invoke(ret, v)
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
		v = it.f.Invoke(v)
	}
}

func (it *Iterate) Cons(o any) Conser {
	return aseqCons(it, o)
}

func (it *Iterate) Count() int {
	return aseqCount(it)
}

func (it *Iterate) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (it *Iterate) Equals(o any) bool {
	return aseqEquals(it, o)
}

func (it *Iterate) Equiv(o any) bool {
	return aseqEquiv(it, o)
}

func (it *Iterate) Hash() uint32 {
	return aseqHash(&it.hash, it)
}

func (it *Iterate) HashEq() uint32 {
	return aseqHashEq(&it.hasheq, it)
}

func (it *Iterate) More() ISeq {
	return aseqMore(it)
}

func (it *Iterate) Seq() ISeq {
	return it
}

func (it *Iterate) String() string {
	return aseqString(it)
}

func (it *Iterate) xxx_sequential() {}
