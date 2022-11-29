package value

// NewIterator returns a lazy sequence of x, f(x), f(f(x)), ....
func NewIterator(f func(interface{}) interface{}, x interface{}) ISeq {
	return &iterator{f: f, x: x}
}

type iterator struct {
	f func(interface{}) interface{}
	x interface{}
}

func (i *iterator) First() interface{} {
	return i.x
}

func (i *iterator) Rest() ISeq {
	return NewIterator(i.f, i.f(i.x))
}

func (i *iterator) IsEmpty() bool {
	return false
}

// NewRangeIterator returns a lazy sequence of start, start + step, start + 2*step, ...
func NewRangeIterator(start, end, step int64) ISeq {
	if end <= start {
		return emptyList
	}

	return &rangeIterator{start: start, end: end, step: step}
}

type rangeIterator struct {
	// TODO: support arbitrary numeric types!
	start, end, step int64
}

func (i *rangeIterator) First() interface{} {
	return i.start
}

func (i *rangeIterator) Rest() ISeq {
	next := i.start + i.step
	if next >= i.end {
		return emptyList
	}
	return &rangeIterator{start: next, end: i.end, step: i.step}
}

func (i *rangeIterator) IsEmpty() bool {
	return false
}

// NewValueIterator returns a lazy sequence of the values of x.
func NewVectorIterator(x *Vector, i int) ISeq {
	return &vectorIterator{v: x, i: i}
}

type vectorIterator struct {
	v *Vector
	i int
}

func (i *vectorIterator) First() interface{} {
	return i.v.ValueAt(i.i)
}

func (i *vectorIterator) Rest() ISeq {
	return &vectorIterator{v: i.v, i: i.i + 1}
}

func (i *vectorIterator) IsEmpty() bool {
	return i.i >= i.v.Count()
}
