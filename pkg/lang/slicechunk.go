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
	return &SliceChunk{
		slc: slc,
	}
}

func (sc *SliceChunk) Count() int {
	return len(sc.slc)
}

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
