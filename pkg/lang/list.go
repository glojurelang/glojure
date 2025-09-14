package lang

import (
	"reflect"

	"github.com/glojurelang/glojure/internal/murmur3"
)

// List is a list of values.
type List struct {
	meta   IPersistentMap
	hash   uint32
	hashEq uint32

	// the empty list is represented by a nil item and a nil next. all
	// other lists have a non-nil item and a non-nil next.
	item any
	next *List
	size int
}

var (
	_ ASeq            = (*List)(nil)
	_ IObj            = (*List)(nil)
	_ ISeq            = (*List)(nil)
	_ IPersistentList = (*List)(nil)
	_ Counted         = (*List)(nil)
	_ IReduce         = (*List)(nil)
	_ IReduceInit     = (*List)(nil)
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
	_ IHashEq         = (*EmptyList)(nil)
)

func (e *EmptyList) xxx_sequential() {}

func (e *EmptyList) Cons(x any) Conser {
	return NewList(x)
}

func (e *EmptyList) Count() int {
	return 0
}

func (e *EmptyList) xxx_counted() {}

func (e *EmptyList) Peek() any {
	return nil
}

func (e *EmptyList) Pop() IPersistentStack {
	panic("cannot pop empty list")
}

func (e *EmptyList) Seq() ISeq {
	return nil
}

func (e *EmptyList) First() any {
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

func (e *EmptyList) Equals(other any) bool {
	if e == other {
		return true
	}
	if _, ok := other.(Sequential); ok {
		return Seq(other) == nil
	}
	t := reflect.TypeOf(other)
	if t != nil && t.Kind() == reflect.Slice {
		return Seq(other) == nil
	}
	return false
}

func (e *EmptyList) Equiv(other any) bool {
	return e.Equals(other)
}

func (e *EmptyList) Meta() IPersistentMap {
	return e.meta
}

func (e *EmptyList) Hash() uint32 {
	return 1
}

var (
	emptyHashOrdered = murmur3.HashOrdered(nil, HashEq)
)

func (e *EmptyList) HashEq() uint32 {
	return emptyHashOrdered
}

func (e *EmptyList) WithMeta(meta IPersistentMap) any {
	if e.meta == meta {
		return e
	}

	cpy := *e
	cpy.meta = meta
	return &cpy
}

func (e *EmptyList) String() string {
	return "()"
}

func (e *EmptyList) Reduce(f IFn) any {
	return f.Invoke()
}

func (e *EmptyList) ReduceInit(f IFn, init any) any {
	return init
}

var emptyList = &EmptyList{}

////////////////////////////////////////////////////////////////////////////////

func NewList(values ...any) IPersistentList {
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

func ConsList(item any, next *List) *List {
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

func (l *List) First() any {
	return l.Item()
}

// Item returns the data from this list node. AKA car.
func (l *List) Item() any {
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

func (l *List) xxx_counted() {}

func (l *List) Cons(x any) Conser {
	return ConsList(x, l)
}

func (l *List) String() string {
	return PrintString(l)
}

func (l *List) Reduce(f IFn) any {
	ret := l.First()
	for s := l.Next(); s != nil; s = s.Next() {
		ret = f.Invoke(ret, s.First())
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
	}
	return ret
}

func (l *List) ReduceInit(f IFn, init any) any {
	ret := f.Invoke(init, l.First())
	for s := l.Next(); s != nil; s = s.Next() {
		if IsReduced(ret) {
			return ret.(IDeref).Deref()
		}
		ret = f.Invoke(ret, s.First())
	}
	if IsReduced(ret) {
		return ret.(IDeref).Deref()
	}
	return ret
}

func (l *List) Meta() IPersistentMap {
	return l.meta
}

func (l *List) WithMeta(meta IPersistentMap) any {
	if l.meta == meta {
		return l
	}

	cpy := *l
	cpy.meta = meta
	return &cpy
}

func (l *List) Peek() any {
	return l.Item()
}

func (l *List) Pop() IPersistentStack {
	if l.next == nil {
		return emptyList
	}
	return l.next
}

func (l *List) Equals(other any) bool {
	return aseqEquals(l, other)
}

func (l *List) Equiv(other any) bool {
	return aseqEquiv(l, other)
}

func (l *List) Hash() uint32 {
	return aseqHash(&l.hash, l)
}

func (l *List) HashEq() uint32 {
	return aseqHashEq(&l.hashEq, l)
}
