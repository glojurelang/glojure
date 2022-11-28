package value

import "strings"

// List is a list of values.
type List struct {
	Section

	// the empty list is represented by a nil item and a nil next. all
	// other lists have a non-nil item and a non-nil next.
	item Value
	next *List
}

var emptyList = &List{}

func NewList(values []Value, opts ...Option) *List {
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
		}
	}
	return list
}

func ConsList(item Value, next *List) *List {
	if next == nil {
		next = emptyList
	}
	return &List{
		item: item,
		next: next,
	}
}

// Item returns the data from this list node. AKA car.
func (l *List) Item() Value {
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

func (l *List) IsEmpty() bool {
	return l.item == nil && l.next == nil
}

func (l *List) Count() int {
	count := 0
	for !l.IsEmpty() {
		count++
		l = l.next
	}
	return count
}

func (l *List) Conj(items ...Value) Conjer {
	if len(items) == 0 {
		return l
	}

	for _, item := range items {
		l = ConsList(item, l)
	}
	return l
}

func (l *List) Nth(i int) (v Value, ok bool) {
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

func (l *List) Enumerate() (<-chan Value, func()) {
	return enumerateFunc(func() (v Value, ok bool) {
		if l.IsEmpty() {
			return nil, false
		}
		v = l.item
		l = l.next
		return v, true
	})
}

func enumerateFunc(next func() (v Value, ok bool)) (<-chan Value, func()) {
	ch := make(chan Value)

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

	// special case for quoted values
	if l.Count() == 2 {
		// TODO: only do this if it used quote shorthand when read.
		if sym, ok := l.item.(*Symbol); ok {
			switch sym.Value {
			case "splice-unquote":
				b.WriteString("~@")
			default:
				goto NoQuote
			}
			b.WriteString(ToString(l.next.item))
			return b.String()
		}
	}
NoQuote:

	b.WriteString("(")
	for cur := l; !cur.IsEmpty(); cur = cur.next {
		v := cur.item
		if v == nil {
			b.WriteString("()")
		} else {
			b.WriteString(ToString(v))
		}
		if !cur.next.IsEmpty() {
			b.WriteString(" ")
		}
	}
	b.WriteString(")")
	return b.String()
}

func (l *List) Equal(v interface{}) bool {
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
