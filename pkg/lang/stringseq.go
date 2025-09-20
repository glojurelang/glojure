package lang

type (
	StringSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		str []rune
		i   int
	}
)

var (
	_ ASeq        = (*StringSeq)(nil)
	_ IndexedSeq  = (*StringSeq)(nil)
	_ IDrop       = (*StringSeq)(nil)
	_ IReduceInit = (*StringSeq)(nil)
)

func NewStringSeq(s string, i int) *StringSeq {
	if len(s) == 0 {
		return nil
	}
	runes := []rune(s)
	if i >= len(runes) {
		return nil
	}
	return &StringSeq{str: runes, i: i}
}

func newStringSeq(s []rune, i int) *StringSeq {
	if len(s) == 0 || i >= len(s) {
		return nil
	}
	return &StringSeq{str: s, i: i}
}

func (s *StringSeq) xxx_sequential() {}

func (s *StringSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *StringSeq) WithMeta(meta IPersistentMap) any {
	if meta == s.meta {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}

func (s *StringSeq) String() string {
	return aseqString(s)
}

func (s *StringSeq) Seq() ISeq {
	return s
}

func (s *StringSeq) Cons(o any) Conser {
	return aseqCons(s, o)
}

func (s *StringSeq) First() any {
	return NewChar(s.str[s.i])
}

func (s *StringSeq) Next() ISeq {
	if s.i+1 >= len(s.str) {
		return nil
	}
	res := newStringSeq(s.str, s.i+1)
	res.meta = s.meta
	return res
}

func (s *StringSeq) More() ISeq {
	return aseqMore(s)
}

func (s *StringSeq) Count() int {
	return len(s.str) - s.i
}

func (s *StringSeq) xxx_counted() {}

func (s *StringSeq) Empty() IPersistentCollection {
	return aseqEmpty()
}

func (s *StringSeq) Equals(o any) bool {
	return aseqEquals(s, o)
}

func (s *StringSeq) Equiv(o any) bool {
	return aseqEquiv(s, o)
}

func (s *StringSeq) Hash() uint32 {
	return aseqHash(&s.hash, s)
}

func (s *StringSeq) HashEq() uint32 {
	return aseqHashEq(&s.hasheq, s)
}

func (s *StringSeq) Index() int {
	return s.i
}

func (s *StringSeq) Drop(n int) Sequential {
	ii := s.i + n
	if ii >= len(s.str) {
		return nil
	}
	return newStringSeq(s.str, ii).WithMeta(s.meta).(Sequential)
}

func (s *StringSeq) ReduceInit(f IFn, init any) any {
	acc := init
	for i := s.i; i < len(s.str); i++ {
		acc = f.Invoke(acc, NewChar(s.str[i]))
		if IsReduced(acc) {
			return acc.(IDeref).Deref()
		}
	}
	return acc
}
