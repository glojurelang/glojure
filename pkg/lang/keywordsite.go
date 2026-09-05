package lang

import "sync/atomic"

// keywordSiteEntries is the number of shapes a KeywordSite caches.
const keywordSiteEntries = 4

// KeywordSite is an inline cache for one keyword lookup site in compiled
// code. It remembers the keyword map shapes and record types seen at
// the site together with the slot index of the keyword in each, so
// repeated lookups on values of a known shape skip the key scan. A site
// that sees more shapes than it can hold scans like a plain lookup for
// the extra shapes.
type KeywordSite struct {
	entries [keywordSiteEntries]atomic.Pointer[keywordSiteEntry]
	full    atomic.Bool
}

// keywordSiteEntry caches the slot index of the site's keyword in one
// keyword map shape or one record type; exactly one of the two is set.
type keywordSiteEntry struct {
	shape  *KeywordMapShape
	record *RecordType
	index  int
}

// Get returns the value of kw in coll, or def when coll has no entry
// for it. It behaves like kw.Invoke2(coll, def) for every collection.
// A map of the first cached shape with no pending updates is the
// common case at a site that keeps seeing one shape, so it reads the
// slot before the general lookup. A map without a keyword shape never
// matches because record entries carry recordEntryShape instead of a
// nil shape.
func (s *KeywordSite) Get(kw Keyword, coll, def any) any {
	if m, ok := coll.(*Map); ok && m.keywordDelta == nil {
		if e := s.entries[0].Load(); e != nil && e.shape == m.keywordShape {
			return m.keyVals[e.index]
		}
	}
	return s.get(kw, coll, def)
}

// recordEntryShape marks the entries of record types so the inline
// check in Get cannot mistake one for a map that has no keyword shape.
var recordEntryShape = &KeywordMapShape{}

func (s *KeywordSite) get(kw Keyword, coll, def any) any {
	switch c := coll.(type) {
	case *Map:
		return s.getMap(kw, c, def)
	case RecordValue:
		return s.getRecord(kw, c, def)
	}
	return kw.Invoke2(coll, def)
}

func (s *KeywordSite) getMap(kw Keyword, m *Map, def any) any {
	shape := m.keywordShape
	if shape == nil {
		return m.valAtKeyword(kw, def)
	}
	for i := range s.entries {
		entry := s.entries[i].Load()
		if entry == nil {
			break
		}
		if entry.shape == shape {
			return m.keywordValueAt(entry.index)
		}
	}
	index := shape.indexOf(kw)
	if index < 0 {
		return def
	}
	if !s.full.Load() {
		s.remember(&keywordSiteEntry{shape: shape, index: index})
	}
	return m.keywordValueAt(index)
}

// getRecord reads a record field by its cached index. Records defined
// in another compiled package have a Go type the generated lookup
// helpers cannot name, so this is their fast path.
func (s *KeywordSite) getRecord(kw Keyword, r RecordValue, def any) any {
	recordType := r.RecordType()
	for i := range s.entries {
		entry := s.entries[i].Load()
		if entry == nil {
			break
		}
		if entry.record == recordType {
			return r.RecordField(entry.index)
		}
	}
	index, ok := recordType.FieldIndex(kw)
	if !ok {
		return RecordValAtDefault(r, kw, def)
	}
	if !s.full.Load() {
		s.remember(&keywordSiteEntry{
			shape:  recordEntryShape,
			record: recordType,
			index:  index,
		})
	}
	return r.RecordField(index)
}

func (s *KeywordSite) remember(entry *keywordSiteEntry) {
	for i := range s.entries {
		if s.entries[i].CompareAndSwap(nil, entry) {
			return
		}
		current := s.entries[i].Load()
		if current.shape == entry.shape && current.record == entry.record {
			return
		}
	}
	s.full.Store(true)
}
