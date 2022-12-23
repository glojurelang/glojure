package value

// MapEntry represents a key-value pair in a map.
type MapEntry struct {
	key, val interface{}
}

var (
	_ IMapEntry         = (*MapEntry)(nil)
	_ IPersistentVector = (*MapEntry)(nil)
	_ ISeqable          = (*MapEntry)(nil)
)

func (me *MapEntry) Key() interface{} {
	if me.key == nil {
		return nil
	}
	return me.key
}

func (me *MapEntry) Val() interface{} {
	if me.val == nil {
		return nil
	}
	return me.val
}

func (me *MapEntry) Count() int {
	return 2
}

func (me *MapEntry) Length() int {
	return me.Count()
}

func (me *MapEntry) Nth(i int) (interface{}, bool) {
	switch i {
	case 0:
		return me.Key(), true
	case 1:
		return me.Val(), true
	default:
		return nil, false
	}
}

func (me *MapEntry) NthDefault(i int, d interface{}) interface{} {
	if v, ok := me.Nth(i); ok {
		return v
	}
	return d
}

func (me *MapEntry) Peek() interface{} {
	return me.Val()
}

func (me *MapEntry) Pop() IPersistentStack {
	return NewVector(me.key)
}

func (me *MapEntry) RSeq() ISeq {
	return me.asVector().RSeq()
}

func (me *MapEntry) Assoc(k, v interface{}) Associative {
	return me.asVector().Assoc(k, v)
}

func (me *MapEntry) AssocN(i int, o interface{}) IPersistentVector {
	return me.asVector().AssocN(i, o)
}

func (me *MapEntry) Cons(o interface{}) IPersistentVector {
	return me.asVector().Cons(o)
}

func (me *MapEntry) ContainsKey(k interface{}) bool {
	return me.asVector().ContainsKey(k)
}

func (me *MapEntry) EntryAt(k interface{}) IMapEntry {
	return me.asVector().EntryAt(k)
}

func (me *MapEntry) Equal(o interface{}) bool {
	return me.asVector().Equal(o)
}

func (me *MapEntry) Seq() ISeq {
	return me.asVector().Seq()
}

func (me *MapEntry) asVector() *Vector {
	return NewVector(me.key, me.val)
}

func (me *MapEntry) String() string {
	return me.asVector().String()
}
