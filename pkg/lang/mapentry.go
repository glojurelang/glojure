package lang

// MapEntry represents a key-value pair in a map.
type MapEntry struct {
	hasheq uint32

	key, val any
}

var (
	_ AMapEntry         = (*MapEntry)(nil)
	_ IPersistentVector = (*MapEntry)(nil)
	_ Seqable           = (*MapEntry)(nil)
)

func NewMapEntry(key, val any) *MapEntry {
	return &MapEntry{key: key, val: val}
}

func (me *MapEntry) xxx_sequential() {}

func (me *MapEntry) Key() any {
	return me.key
}

func (me *MapEntry) Val() any {
	return me.val
}

func (me *MapEntry) Count() int {
	return amapentryCount(me)
}

func (me *MapEntry) Nth(i int) any {
	return amapentryNth(me, i)
}

func (me *MapEntry) NthDefault(i int, d any) any {
	return amapentryNthDefault(me, i, d)
}

func (me *MapEntry) Pop() IPersistentStack {
	return amapentryPop(me)
}

func (me *MapEntry) Seq() ISeq {
	return amapentrySeq(me)
}

func (me *MapEntry) AssocN(i int, o any) IPersistentVector {
	return amapentryAssocN(me, i, o)
}

func (me *MapEntry) Cons(o any) Conser {
	return amapentryCons(me, o)
}

func (me *MapEntry) Empty() IPersistentCollection {
	return amapentryEmpty(me)
}

func (me *MapEntry) RSeq() ISeq {
	return apersistentVectorRSeq(me)
}

func (me *MapEntry) Assoc(k, v any) Associative {
	return apersistentVectorAssoc(me, k, v)
}

func (me *MapEntry) ContainsKey(k any) bool {
	return apersistentVectorContainsKey(me, k)
}

func (me *MapEntry) EntryAt(k any) IMapEntry {
	return apersistentVectorEntryAt(me, k)
}

func (me *MapEntry) String() string {
	return apersistentVectorString(me)
}

func (me *MapEntry) ApplyTo(args ISeq) any {
	return afnApplyTo(me, args)
}

func (me *MapEntry) Equiv(o any) bool {
	return apersistentVectorEquiv(me, o)
}

func (me *MapEntry) HashEq() uint32 {
	return apersistentVectorHashEq(&me.hasheq, me)
}

func (me *MapEntry) Invoke(args ...any) any {
	return apersistentVectorInvoke(me, args)
}

func (me *MapEntry) Length() int {
	return apersistentVectorLength(me)
}

func (me *MapEntry) Peek() any {
	return apersistentVectorPeek(me)
}

func (me *MapEntry) ValAt(key any) any {
	return apersistentVectorValAt(me, key)
}

func (me *MapEntry) ValAtDefault(key, notFound any) any {
	return apersistentVectorValAtDefault(me, key, notFound)
}
