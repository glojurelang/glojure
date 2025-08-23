package lang

import "errors"

// TransformerIterator provides a view over a Transduced collection.
type TransformerIterator struct {
	// source
	sourceIter any
	xf         IFn
	multi      bool

	// iteration state
	buffer    []any
	next      any
	completed bool
}

var (
	_ Iterator = (*TransformerIterator)(nil)

	transformerIteratorNone = &struct{}{}
)

// NewTransformerIteratorSeq creates a new transformer iterator.
func NewTransformerIterator(xform IFn, iter any, multi bool) *TransformerIterator {
	ti := &TransformerIterator{
		sourceIter: iter,
		multi:      multi,
		next:       transformerIteratorNone,
	}
	ti.xf = xform.Invoke(NewFnFunc(func(args ...any) any {
		switch len(args) {
		case 0:
			return nil
		case 1:
			return args[0]
		case 2:
			ti.buffer = append(ti.buffer, args[1])
			return args[0]
		default:
			panic("invalid arity")
		}
	})).(IFn)

	return ti
}

func (ti *TransformerIterator) HasNext() bool {
	return ti.step()
}

func (ti *TransformerIterator) Next() any {
	if ti.HasNext() {
		ret := ti.next
		ti.next = transformerIteratorNone
		return ret
	}
	panic(errors.New("no next element"))
}

func (ti *TransformerIterator) Remove() {
	panic(errors.New("remove not supported"))
}

func (ti *TransformerIterator) step() bool {
	return false
}
