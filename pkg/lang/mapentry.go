package lang

// MapEntry represents a key-value pair in a map.
type MapEntry struct {
	key, val interface{}
}

var (
	_ IMapEntry         = (*MapEntry)(nil)
	_ IPersistentVector = (*MapEntry)(nil)
	_ ISeqable          = (*MapEntry)(nil)
)

func NewMapEntry(key, val interface{}) *MapEntry {
	return &MapEntry{key: key, val: val}
}

func (me *MapEntry) xxx_sequential() {}

func (me *MapEntry) Key() interface{} {
	if me.key == nil {
		return nil
	}
	return me.key
}

func (me *MapEntry) GetKey() interface{} {
	return me.Key()
}

func (me *MapEntry) Val() interface{} {
	if me.val == nil {
		return nil
	}
	return me.val
}

func (me *MapEntry) GetValue() interface{} {
	return me.Val()
}

func (me *MapEntry) Count() int {
	return 2
}

func (me *MapEntry) Length() int {
	return me.Count()
}

func (me *MapEntry) Nth(i int) interface{} {
	switch i {
	case 0:
		return me.Key()
	case 1:
		return me.Val()
	default:
		panic(NewIndexOutOfBoundsError())
	}
}

func (me *MapEntry) NthDefault(i int, d interface{}) interface{} {
	if i >= 0 && i < 2 {
		return me.Nth(i)
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

func (me *MapEntry) Conj(v interface{}) Conjer {
	return me.asVector().Conj(v)
}

func (me *MapEntry) IsEmpty() bool {
	return false
}

func (me *MapEntry) Empty() IPersistentCollection {
	return nil
}

func (me *MapEntry) ValAt(k interface{}) interface{} {
	return me.asVector().ValAt(k)
}

func (me *MapEntry) ValAtDefault(k, def interface{}) interface{} {
	return me.asVector().ValAtDefault(k, def)
}
