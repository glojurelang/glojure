package value

import (
	"strings"
)

// List is a list of values.
type List struct {
	Section

	// the empty list is represented by a nil item and a nil next. all
	// other lists have a non-nil item and a non-nil next.
	item interface{}
	next *List
	size int
}

var emptyList = &List{}

func NewList(values []interface{}, opts ...Option) *List {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	list := emptyList
	for i := len(values) - 1; i >= 0; i-- {
		list = &List{
			Section: o.section,
			item:    values[i],
			next:    list,
			size:    list.size + 1,
		}
	}
	return list
}

func ConsList(item interface{}, next *List) *List {
	if next == nil {
		next = emptyList
	}
	return &List{
		item: item,
		next: next,
		size: next.size + 1,
	}
}

func (l *List) First() interface{} {
	return l.Item()
}

// Item returns the data from this list node. AKA car.
func (l *List) Item() interface{} {
	if l.IsEmpty() {
		panic("cannot get item of empty list")
	}
	return l.item
}

// Next returns the next list node. AKA cdr, with the requirement that
// it must be a list.
func (l *List) Next() *List {
	if l.IsEmpty() {
		panic("cannot get next of empty list")
	}
	return l.next
}

func (l *List) Rest() ISeq {
	if l.IsEmpty() {
		return l
	}
	return l.Next()
}

func (l *List) IsEmpty() bool {
	return l.size == 0
}

func (l *List) Count() int {
	return l.size
}

func (l *List) Conj(items ...interface{}) Conjer {
	if len(items) == 0 {
		return l
	}

	for _, item := range items {
		l = ConsList(item, l)
	}
	return l
}

func (l *List) Nth(i int) (v interface{}, ok bool) {
	if i < 0 {
		return nil, false
	}
	for !l.IsEmpty() {
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
	b := strings.Builder{}
	b.WriteString("(")
	for cur := l; !cur.IsEmpty(); cur = cur.next {
		v := cur.item
		b.WriteString(ToString(v))
		if !cur.next.IsEmpty() {
			b.WriteString(" ")
		}
	}
	b.WriteString(")")
	return b.String()
}

func (l *List) Equal(v interface{}) bool {
	if l == v {
		return true
	}

	other, ok := v.(*List)
	if !ok {
		return false
	}

	for {
		if l.IsEmpty() != other.IsEmpty() {
			return false
		}
		if l.IsEmpty() {
			return true
		}
		if !Equal(l.item, other.item) {
			return false
		}
		l = l.next
		other = other.next
	}

	return true
}

func (l *List) GoValue() interface{} {
	var vals []interface{}
	for cur := l; !cur.IsEmpty(); cur = cur.next {
		val := cur.Item()
		if val == nil {
			vals = append(vals, nil)
			continue
		}

		if gv, ok := val.(GoValuer); ok {
			vals = append(vals, gv.GoValue())
			continue
		}

		vals = append(vals, val)
	}
	return vals
}
