package lang

import "testing"

type keywordEqualValue struct{ keyword Keyword }

func (v keywordEqualValue) Equals(other any) bool { return other == v.keyword }

func TestKeywordComparisonPreservesGenericEquality(t *testing.T) {
	key := NewKeyword("key")
	for _, value := range []any{nil, key, NewKeyword("other"), "key", int64(1),
		keywordEqualValue{keyword: key}} {
		if got, want := KeywordEquals(value, key), Equals(value, key); got != want {
			t.Fatalf("keyword comparison for %T = %v, want %v", value, got, want)
		}
	}
}

func TestKeywordDissocPreservesMapsAndMetadata(t *testing.T) {
	a, b, absent := NewKeyword("a"), NewKeyword("b"), NewKeyword("absent")
	shape := NewKeywordMapShape("a", "b")
	meta := NewMap(NewKeyword("tag"), true)
	for _, base := range []IPersistentMap{
		NewMap(a, nil, b, 2),
		NewStaticKeywordMap(shape, nil, 2),
		NewMap("string", 1, a, nil, b, 2),
		NewPersistentHashMap(a, nil, b, 2),
	} {
		m := base.(IObj).WithMeta(meta).(IPersistentMap)
		if got := DissocKeyword(m, absent); got != m {
			t.Fatalf("missing-key dissoc copied %T", m)
		}
		result := DissocKeyword(m, a).(IPersistentMap)
		if !Equals(result, Dissoc(m, a)) || result.(IMeta).Meta() != meta {
			t.Fatalf("keyword dissoc changed value or metadata for %T", m)
		}
		if !m.ContainsKey(a) || result.ContainsKey(a) || result.ValAt(b) != 2 {
			t.Fatalf("keyword dissoc mutated a snapshot or removed the wrong key in %T", m)
		}
	}
	if got := DissocKeyword(nil, a); got != nil {
		t.Fatalf("dissoc nil = %v", got)
	}
}
