package lang

import "sync/atomic"

// keywordSiteEntries is the number of map shapes a KeywordSite caches.
const keywordSiteEntries = 4

// KeywordSite is an inline cache for one keyword lookup site in compiled
// code. It remembers the keyword map shapes seen at the site together
// with the slot index of the keyword in each, so repeated lookups on
// maps of a known shape skip the key scan. A site that sees more shapes
// than it can hold scans like a plain lookup for the extra shapes.
type KeywordSite struct {
	entries [keywordSiteEntries]atomic.Pointer[keywordSiteEntry]
	full    atomic.Bool
}

type keywordSiteEntry struct {
	shape *KeywordMapShape
	index int
}

// Get returns the value of kw in coll, or def when coll has no entry
// for it. It behaves like kw.Invoke2(coll, def) for every collection.
func (s *KeywordSite) Get(kw Keyword, coll, def any) any {
	m, ok := coll.(*Map)
	if !ok {
		return kw.Invoke2(coll, def)
	}
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
		s.remember(shape, index)
	}
	return m.keywordValueAt(index)
}

func (s *KeywordSite) remember(shape *KeywordMapShape, index int) {
	entry := &keywordSiteEntry{shape: shape, index: index}
	for i := range s.entries {
		if s.entries[i].CompareAndSwap(nil, entry) {
			return
		}
		if s.entries[i].Load().shape == shape {
			return
		}
	}
	s.full.Store(true)
}
