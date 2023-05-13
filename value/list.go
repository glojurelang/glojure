package value

// List is a list of values.
type List struct {
	meta IPersistentMap

	// the empty list is represented by a nil item and a nil next. all
	// other lists have a non-nil item and a non-nil next.
	item interface{}
	next *List
	size int
}

var (
	_ IObj            = (*List)(nil)
	_ ISeq            = (*List)(nil)
	_ IPersistentList = (*List)(nil)
	_ Counted         = (*List)(nil)
)

type EmptyList struct {
	meta IPersistentMap
}

var (
	_ IObj            = (*EmptyList)(nil)
	_ ISeq            = (*EmptyList)(nil)
	_ IPersistentList = (*EmptyList)(nil)
	_ Counted         = (*EmptyList)(nil)
	_ IReduce         = (*EmptyList)(nil)
	_ IReduceInit     = (*EmptyList)(nil)
)

func (e *EmptyList) xxx_sequential() {}

func (e *EmptyList) Conj(x interface{}) Conjer {
	return NewList(x)
}

func (e *EmptyList) Count() int {
	return 0
}

func (e *EmptyList) Peek() interface{} {
	return nil
}

func (e *EmptyList) Pop() IPersistentStack {
	panic("cannot pop empty list")
}

func (e *EmptyList) Seq() ISeq {
	return nil
}

func (e *EmptyList) First() interface{} {
	return nil
}

func (e *EmptyList) Next() ISeq {
	return nil
}

func (e *EmptyList) More() ISeq {
	return e
}

func (e *EmptyList) IsEmpty() bool {
	return true
}

func (e *EmptyList) Empty() IPersistentCollection {
	return e
}

func (e *EmptyList) Equal(other interface{}) bool {
	if e == other {
		return true
	}
	if _, ok := other.(*EmptyList); ok {
		return true
	}
	return false
}

func (e *EmptyList) Meta() IPersistentMap {
	return e.meta
}

func (e *EmptyList) WithMeta(meta IPersistentMap) interface{} {
	if Equal(e.meta, meta) {
		return e
	}

	cpy := *e
	cpy.meta = meta
	return &cpy
}

func (e *EmptyList) String() string {
	return "()"
}

func (e *EmptyList) Reduce(f IFn) interface{} {
	return f.Invoke()
}

func (e *EmptyList) ReduceInit(f IFn, init interface{}) interface{} {
	return init
}

var emptyList = &EmptyList{}

////////////////////////////////////////////////////////////////////////////////

func NewList(values ...interface{}) IPersistentList {
	if len(values) == 0 {
		return &EmptyList{}
	}

	var list *List
	size := 0
	for i := len(values) - 1; i >= 0; i-- {
		size++
		list = &List{
			item: values[i],
			next: list,
			size: size,
		}
	}
	return list
}

func ConsList(item interface{}, next *List) *List {
	size := 1
	if next != nil {
		size += next.size
	}
	return &List{
		item: item,
		next: next,
		size: size,
	}
}

func (l *List) xxx_sequential() {}

func (l *List) First() interface{} {
	return l.Item()
}

// Item returns the data from this list node. AKA car.
func (l *List) Item() interface{} {
	return l.item
}

func (l *List) Seq() ISeq {
	return l
}

// Next returns the next list node. AKA cdr, with the requirement that
// it must be a list.
func (l *List) Next() ISeq {
	if l.IsEmpty() || l.Count() == 1 {
		return nil
	}
	return l.next
}

func (l *List) More() ISeq {
	s := l.Next()
	if s == nil {
		return emptyList
	}
	return s
}

func (l *List) IsEmpty() bool {
	return false
}

func (l *List) Empty() IPersistentCollection {
	return emptyList.WithMeta(l.meta).(IPersistentCollection)
}

func (l *List) Count() int {
	return l.size
}

func (l *List) Conj(x interface{}) Conjer {
	return ConsList(x, l)
}

func (l *List) Nth(i int) (v interface{}, ok bool) {
	if i < 0 {
		return nil, false
	}
	for l != nil {
		if i == 0 {
			return l.item, true
		}
		i--
		l = l.next
	}
	return nil, false
}

func (l *List) Enumerate() (<-chan interface{}, func()) {
	return enumerateFunc(func() (v interface{}, ok bool) {
		if l.IsEmpty() {
			return nil, false
		}
		v = l.item
		l = l.next
		return v, true
	})
}

func enumerateFunc(next func() (v interface{}, ok bool)) (<-chan interface{}, func()) {
	ch := make(chan interface{})

	done := make(chan struct{})
	cancel := func() {
		close(done)
	}
	go func() {
		for {
			v, ok := next()
			if !ok {
				break
			}
			select {
			case ch <- v:
			case <-done:
				return
			}
		}
		close(ch)
	}()
	return ch, cancel
}

func (l *List) String() string {
	return PrintString(l)
}

// TODO: rename to Equiv
func (l *List) Equal(v interface{}) bool {
	// TODO: move to a helper for sequential equality
	if _, ok := v.(ISeqable); !ok {
		if _, ok := v.(*List); !ok {
			return false
		}
	}
	if counter, ok := v.(Counted); ok {
		if l.Count() != counter.Count() {
			return false
		}
	}
	seq := Seq(v)
	for cur := Seq(l); cur != nil; cur, seq = cur.Next(), seq.Next() {
		if seq == nil || !Equal(cur.First(), seq.First()) {
			return false
		}
	}
	return seq == nil
}

func (l *List) Meta() IPersistentMap {
	return l.meta
}

func (l *List) WithMeta(meta IPersistentMap) interface{} {
	if Equal(l.meta, meta) {
		return l
	}

	cpy := *l
	cpy.meta = meta
	return &cpy
}

func (l *List) Peek() interface{} {
	return l.Item()
}

func (l *List) Pop() IPersistentStack {
	if l.next == nil {
		return emptyList
	}
	return l.next
}
