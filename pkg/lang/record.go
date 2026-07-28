package lang

import (
	"fmt"
	"sync"
)

// RecordMarker lets generated record implementations satisfy the private
// marker methods on IRecord and Counted by embedding an exported Go type.
type RecordMarker struct{}

func (RecordMarker) xxx_irecord() {}
func (RecordMarker) xxx_counted() {}

// RecordType is the backend-neutral identity and field layout of a defrecord.
// Runtime records and AOT-generated Go records share the same descriptor.
type RecordType struct {
	namespace  string
	name       string
	fieldNames []string
	fieldKeys  []Keyword
	fieldIndex map[Keyword]int
}

var recordTypes sync.Map

// InternRecordType returns the canonical descriptor for a qualified record
// name. Reusing a name with a different field layout is an error.
func InternRecordType(namespace, name string, fieldNames ...string) *RecordType {
	qualifiedName := name
	if namespace != "" {
		qualifiedName = namespace + "." + name
	}
	candidate := &RecordType{
		namespace:  namespace,
		name:       name,
		fieldNames: append([]string(nil), fieldNames...),
		fieldKeys:  make([]Keyword, len(fieldNames)),
		fieldIndex: make(map[Keyword]int, len(fieldNames)),
	}
	for i, field := range fieldNames {
		key := NewKeyword(field)
		if _, exists := candidate.fieldIndex[key]; exists {
			panic(NewIllegalArgumentError(
				"duplicate defrecord field: " + field,
			))
		}
		candidate.fieldKeys[i] = key
		candidate.fieldIndex[key] = i
	}
	actual, loaded := recordTypes.LoadOrStore(qualifiedName, candidate)
	if !loaded {
		return candidate
	}
	existing := actual.(*RecordType)
	if len(existing.fieldNames) != len(fieldNames) {
		panic(NewIllegalArgumentError(
			"record type redefined with a different field layout: " +
				qualifiedName,
		))
	}
	for i := range fieldNames {
		if existing.fieldNames[i] != fieldNames[i] {
			panic(NewIllegalArgumentError(
				"record type redefined with a different field layout: " +
					qualifiedName,
			))
		}
	}
	return existing
}

func (t *RecordType) Namespace() string { return t.namespace }
func (t *RecordType) Name() string      { return t.name }

func (t *RecordType) FullName() string {
	if t.namespace == "" {
		return t.name
	}
	return t.namespace + "." + t.name
}

func (t *RecordType) String() string { return t.FullName() }

func (t *RecordType) FieldNames() []string {
	return append([]string(nil), t.fieldNames...)
}

func (t *RecordType) FieldKeys() []Keyword {
	return append([]Keyword(nil), t.fieldKeys...)
}

func (t *RecordType) FieldCount() int { return len(t.fieldNames) }

func (t *RecordType) FieldIndex(key Keyword) (int, bool) {
	index, ok := t.fieldIndex[key]
	return index, ok
}

// RecordValue is the small interface shared by interpreted and generated
// records. The ordinary Clojure collection methods are implemented in terms
// of these operations.
type RecordValue interface {
	IRecord
	IPersistentMap
	IObj
	IFn
	IHashEq
	Hasher

	RecordType() *RecordType
	RecordField(index int) any
	RecordExtMap() IPersistentMap
	RecordMeta() IPersistentMap
	RecordWithField(index int, value any) RecordValue
	RecordWithExtMap(ext IPersistentMap) RecordValue
	RecordWithMeta(meta IPersistentMap) RecordValue
}

// RecordAttrs holds record state that most values never use. Keeping
// metadata, extension entries, and hash caches behind one pointer avoids
// paying for four mostly-empty fields on every interpreted or generated
// record.
type RecordAttrs struct {
	meta   IPersistentMap
	ext    IPersistentMap
	hash   uint32
	hasheq uint32
}

func NewRecordAttrs(meta, ext IPersistentMap) *RecordAttrs {
	if meta == nil && ext == nil {
		return nil
	}
	return &RecordAttrs{meta: meta, ext: ext}
}

func RecordAttrsMeta(attrs *RecordAttrs) IPersistentMap {
	if attrs == nil {
		return nil
	}
	return attrs.meta
}

func RecordAttrsExt(attrs *RecordAttrs) IPersistentMap {
	if attrs == nil {
		return nil
	}
	return attrs.ext
}

func RecordAttrsWithoutHash(attrs *RecordAttrs) *RecordAttrs {
	if attrs == nil {
		return nil
	}
	if attrs.meta == nil && attrs.ext == nil {
		return nil
	}
	return &RecordAttrs{meta: attrs.meta, ext: attrs.ext}
}

func RecordAttrsWithMeta(
	attrs *RecordAttrs,
	meta IPersistentMap,
) *RecordAttrs {
	if RecordAttrsMeta(attrs) == meta {
		return attrs
	}
	result := &RecordAttrs{meta: meta}
	if attrs != nil {
		*result = *attrs
		result.meta = meta
	}
	if result.meta == nil && result.ext == nil &&
		result.hash == 0 && result.hasheq == 0 {
		return nil
	}
	return result
}

func RecordAttrsWithExt(
	attrs *RecordAttrs,
	ext IPersistentMap,
) *RecordAttrs {
	if RecordAttrsExt(attrs) == ext {
		return attrs
	}
	result := &RecordAttrs{ext: ext}
	if attrs != nil {
		result.meta = attrs.meta
	}
	return result
}

func RecordAttrsHash(attrs **RecordAttrs, record RecordValue) uint32 {
	if *attrs == nil {
		*attrs = &RecordAttrs{}
	}
	return RecordHash(&(*attrs).hash, record)
}

func RecordAttrsHashEq(attrs **RecordAttrs, record RecordValue) uint32 {
	if *attrs == nil {
		*attrs = &RecordAttrs{}
	}
	return RecordHashEq(&(*attrs).hasheq, record)
}

// Record is the descriptor-backed implementation used by interpreted code.
// AOT emits a concrete Go struct with the same RecordValue contract.
type Record struct {
	RecordMarker
	recordType *RecordType
	fields     []any
	attrs      *RecordAttrs
}

var (
	_ RecordValue = (*Record)(nil)
	_ Counted     = (*Record)(nil)
)

func NewRecord(recordType *RecordType, values ...any) *Record {
	if recordType == nil {
		panic(NewIllegalArgumentError("record type is nil"))
	}
	if len(values) != recordType.FieldCount() {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"wrong number of arguments for %s: expected %d, got %d",
			recordType.FullName(),
			recordType.FieldCount(),
			len(values),
		)))
	}
	return &Record{
		recordType: recordType,
		fields:     append([]any(nil), values...),
	}
}

func (r *Record) RecordType() *RecordType        { return r.recordType }
func (r *Record) RecordField(index int) any      { return r.fields[index] }
func (r *Record) RecordExtMap() IPersistentMap   { return RecordAttrsExt(r.attrs) }
func (r *Record) RecordMeta() IPersistentMap     { return RecordAttrsMeta(r.attrs) }
func (r *Record) Meta() IPersistentMap           { return RecordAttrsMeta(r.attrs) }
func (r *Record) Count() int                     { return RecordCount(r) }
func (r *Record) ValAt(key any) any              { return RecordValAt(r, key) }
func (r *Record) ContainsKey(key any) bool       { return RecordContainsKey(r, key) }
func (r *Record) EntryAt(key any) IMapEntry      { return RecordEntryAt(r, key) }
func (r *Record) Seq() ISeq                      { return RecordSeq(r) }
func (r *Record) Equiv(other any) bool           { return RecordEquiv(r, other) }
func (r *Record) Hash() uint32                   { return RecordAttrsHash(&r.attrs, r) }
func (r *Record) HashEq() uint32                 { return RecordAttrsHashEq(&r.attrs, r) }
func (r *Record) String() string                 { return RecordString(r) }
func (r *Record) Empty() IPersistentCollection   { return RecordEmpty(r) }
func (r *Record) Without(key any) IPersistentMap { return RecordWithout(r, key) }

func (r *Record) ValAtDefault(key, fallback any) any {
	return RecordValAtDefault(r, key, fallback)
}

func (r *Record) Assoc(key, value any) Associative {
	return RecordAssoc(r, key, value)
}

func (r *Record) AssocEx(key, value any) IPersistentMap {
	return RecordAssocEx(r, key, value)
}

func (r *Record) Cons(value any) Conser {
	return RecordCons(r, value)
}

func (r *Record) WithMeta(meta IPersistentMap) any {
	return r.RecordWithMeta(meta)
}

func (r *Record) RecordWithField(index int, value any) RecordValue {
	if Identical(r.fields[index], value) {
		return r
	}
	result := *r
	result.fields = append([]any(nil), r.fields...)
	result.fields[index] = value
	result.attrs = RecordAttrsWithoutHash(r.attrs)
	return &result
}

func (r *Record) RecordWithExtMap(ext IPersistentMap) RecordValue {
	if r.RecordExtMap() == ext {
		return r
	}
	result := *r
	result.attrs = RecordAttrsWithExt(r.attrs, ext)
	return &result
}

func (r *Record) RecordWithMeta(meta IPersistentMap) RecordValue {
	if r.RecordMeta() == meta {
		return r
	}
	result := *r
	result.attrs = RecordAttrsWithMeta(r.attrs, meta)
	return &result
}

func (r *Record) Invoke(args ...any) any {
	return RecordInvoke(r, args...)
}

func (r *Record) ApplyTo(args ISeq) any {
	values := make([]any, 0, 2)
	for seq := args; seq != nil; seq = seq.Next() {
		values = append(values, seq.First())
	}
	return RecordInvoke(r, values...)
}

// RecordConstructor is the first-class root value of ->Type and map->Type.
// Keeping its descriptor explicit lets AOT direct-link any record constructor
// without recognizing source-level names.
type RecordConstructor struct {
	recordType *RecordType
	fromMap    bool
}

var _ IFn = (*RecordConstructor)(nil)

func NewRecordConstructor(
	recordType *RecordType,
	fromMap bool,
) *RecordConstructor {
	if recordType == nil {
		panic(NewIllegalArgumentError("record constructor type is nil"))
	}
	return &RecordConstructor{recordType: recordType, fromMap: fromMap}
}

func (c *RecordConstructor) RecordType() *RecordType { return c.recordType }
func (c *RecordConstructor) FromMap() bool           { return c.fromMap }

func (c *RecordConstructor) Invoke(args ...any) any {
	if !c.fromMap {
		return NewRecord(c.recordType, args...)
	}
	if len(args) != 1 {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"wrong number of arguments for map->%s: expected 1, got %d",
			c.recordType.Name(),
			len(args),
		)))
	}
	return NewRecordFromMap(c.recordType, args[0])
}

func (c *RecordConstructor) ApplyTo(args ISeq) any {
	var values []any
	for seq := args; seq != nil; seq = seq.Next() {
		values = append(values, seq.First())
	}
	return c.Invoke(values...)
}

func NewRecordFromMap(recordType *RecordType, value any) RecordValue {
	var source IPersistentMap
	switch value := value.(type) {
	case nil:
		source = NewMap()
	case IPersistentMap:
		source = value
	default:
		source = NewMap()
		for seq := Seq(value); seq != nil; seq = seq.Next() {
			entry, ok := seq.First().(IMapEntry)
			if !ok {
				panic(NewIllegalArgumentError(
					"map record constructor expects map entries",
				))
			}
			source = source.Assoc(entry.Key(), entry.Val()).(IPersistentMap)
		}
	}
	fields := make([]any, recordType.FieldCount())
	ext := source
	for i, key := range recordType.fieldKeys {
		fields[i] = source.ValAt(key)
		ext = ext.Without(key)
	}
	if ext.Count() == 0 {
		ext = nil
	}
	record := NewRecord(recordType, fields...)
	record.attrs = NewRecordAttrs(nil, ext)
	return record
}

func RecordValAt(record RecordValue, key any) any {
	return RecordValAtDefault(record, key, nil)
}

func RecordValAtDefault(record RecordValue, key, fallback any) any {
	if keyword, ok := key.(Keyword); ok {
		if index, found := record.RecordType().FieldIndex(keyword); found {
			return record.RecordField(index)
		}
	}
	if ext := record.RecordExtMap(); ext != nil {
		return ext.ValAtDefault(key, fallback)
	}
	return fallback
}

func RecordContainsKey(record RecordValue, key any) bool {
	if keyword, ok := key.(Keyword); ok {
		if _, found := record.RecordType().FieldIndex(keyword); found {
			return true
		}
	}
	return record.RecordExtMap() != nil &&
		record.RecordExtMap().ContainsKey(key)
}

func RecordEntryAt(record RecordValue, key any) IMapEntry {
	if !RecordContainsKey(record, key) {
		return nil
	}
	return NewMapEntry(key, RecordValAt(record, key))
}

func RecordAssoc(record RecordValue, key, value any) Associative {
	if keyword, ok := key.(Keyword); ok {
		if index, found := record.RecordType().FieldIndex(keyword); found {
			return record.RecordWithField(index, value)
		}
	}
	ext := record.RecordExtMap()
	if ext == nil {
		ext = NewMap()
	}
	return record.RecordWithExtMap(
		ext.Assoc(key, value).(IPersistentMap),
	)
}

func RecordAssocEx(record RecordValue, key, value any) IPersistentMap {
	if RecordContainsKey(record, key) {
		panic(NewIllegalArgumentError("key already present"))
	}
	return RecordAssoc(record, key, value).(IPersistentMap)
}

func RecordWithout(record RecordValue, key any) IPersistentMap {
	if keyword, ok := key.(Keyword); ok {
		if _, found := record.RecordType().FieldIndex(keyword); found {
			return RecordPersistentMap(record).Without(key)
		}
	}
	ext := record.RecordExtMap()
	if ext == nil || !ext.ContainsKey(key) {
		return record
	}
	next := ext.Without(key)
	if next.Count() == 0 {
		next = nil
	}
	return record.RecordWithExtMap(next)
}

func RecordCount(record RecordValue) int {
	count := record.RecordType().FieldCount()
	if ext := record.RecordExtMap(); ext != nil {
		count += ext.Count()
	}
	return count
}

func RecordPersistentMap(record RecordValue) IPersistentMap {
	keyValues := make([]any, 0, RecordCount(record)*2)
	for i, key := range record.RecordType().fieldKeys {
		keyValues = append(keyValues, key, record.RecordField(i))
	}
	if ext := record.RecordExtMap(); ext != nil {
		for seq := ext.Seq(); seq != nil; seq = seq.Next() {
			entry := seq.First().(IMapEntry)
			keyValues = append(keyValues, entry.Key(), entry.Val())
		}
	}
	result := NewMapUniqueKeys(keyValues...)
	if meta := record.RecordMeta(); meta != nil {
		result = result.(IObj).WithMeta(meta).(IPersistentMap)
	}
	return result
}

func RecordSeq(record RecordValue) ISeq {
	return RecordPersistentMap(record).Seq()
}

func RecordEmpty(record RecordValue) IPersistentCollection {
	panic(NewUnsupportedOperationError(
		"Can't create empty: " + record.RecordType().FullName(),
	))
}

func RecordCons(record RecordValue, value any) Conser {
	switch value := value.(type) {
	case IMapEntry:
		return record.Assoc(value.Key(), value.Val()).(Conser)
	case IPersistentVector:
		if value.Count() != 2 {
			panic(NewIllegalArgumentError(
				"vector arg to map conj must be a pair",
			))
		}
		return record.Assoc(
			MustNth(value, 0),
			MustNth(value, 1),
		).(Conser)
	}
	var result Conser = record
	for seq := Seq(value); seq != nil; seq = seq.Next() {
		result = result.Cons(seq.First())
	}
	return result
}

func RecordEquiv(record RecordValue, other any) bool {
	if record == other {
		return true
	}
	otherRecord, ok := other.(RecordValue)
	if !ok || otherRecord.RecordType() != record.RecordType() {
		return false
	}
	if !Equals(record.RecordExtMap(), otherRecord.RecordExtMap()) {
		return false
	}
	for i := 0; i < record.RecordType().FieldCount(); i++ {
		if !Equals(record.RecordField(i), otherRecord.RecordField(i)) {
			return false
		}
	}
	return true
}

func RecordHash(cache *uint32, record RecordValue) uint32 {
	if *cache == 0 {
		*cache = RecordPersistentMap(record).(Hasher).Hash()
	}
	return *cache
}

func RecordHashEq(cache *uint32, record RecordValue) uint32 {
	if *cache == 0 {
		mapHash := RecordPersistentMap(record).(IHashEq).HashEq()
		*cache = Hash(record.RecordType().FullName()) ^ mapHash
	}
	return *cache
}

func RecordString(record RecordValue) string {
	return "#" + record.RecordType().FullName() +
		PrintString(RecordPersistentMap(record))
}

func RecordInvoke(record RecordValue, args ...any) any {
	switch len(args) {
	case 1:
		return RecordValAt(record, args[0])
	case 2:
		return RecordValAtDefault(record, args[0], args[1])
	default:
		panic(NewIllegalArgumentError(
			"record expects either 1 or 2 arguments",
		))
	}
}

func RecordApplyTo(record RecordValue, args ISeq) any {
	values := make([]any, 0, 2)
	for seq := args; seq != nil; seq = seq.Next() {
		values = append(values, seq.First())
	}
	return RecordInvoke(record, values...)
}
