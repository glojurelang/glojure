package lang

import "errors"

type (
	SliceChunk struct {
		slc []interface{}
	}
)

var (
	_ IChunk = (*SliceChunk)(nil)
)

func NewSliceChunk(slc []interface{}) *SliceChunk {
	return &SliceChunk{slc: slc}
}

func (sc *SliceChunk) Count() int {
	return len(sc.slc)
}

func (sc *SliceChunk) xxx_counted() {}

func (sc *SliceChunk) DropFirst() IChunk {
	if len(sc.slc) == 0 {
		panic(errors.New("DropFirst of empty chunk"))
	}
	return NewSliceChunk(sc.slc[1:])
}

func (sc *SliceChunk) Nth(i int) interface{} {
	return sc.slc[i]
}

func (sc *SliceChunk) NthDefault(i int, def interface{}) interface{} {
	if i >= 0 && i < len(sc.slc) {
		return sc.Nth(i)
	}
	return def
}

func (sc *SliceChunk) fieldOrMethod(name string) (interface{}, bool) {
	switch name {
	case "nth", "Nth":
		return FnFunc1(func(i any) any {
			return sc.Nth(MustAsInt(i))
		}), true
	case "nthDefault", "NthDefault":
		return FnFunc2(func(i, notFound any) any {
			return sc.NthDefault(MustAsInt(i), notFound)
		}), true
	case "count", "Count":
		return FnFunc0(func() any { return sc.Count() }), true
	}
	return nil, false
}

func (sc *SliceChunk) ReduceInit(fn IFn, init interface{}) interface{} {
	ret := fn.Invoke(init, sc.slc[0])
	if IsReduced(ret) {
		return ret
	}
	for i := 1; i < len(sc.slc); i++ {
		ret = fn.Invoke(ret, sc.slc[i])
		if IsReduced(ret) {
			return ret
		}
	}
	return ret
}
