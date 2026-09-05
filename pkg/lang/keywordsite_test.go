package lang

import "testing"

func TestKeywordSiteMap(t *testing.T) {
	var site KeywordSite
	kw := NewKeyword("b")
	m := NewMap(NewKeyword("a"), int64(1), kw, int64(2))
	other := NewMap(NewKeyword("c"), int64(3))
	for i := 0; i < 3; i++ {
		if got := site.Get(kw, m, nil); got != int64(2) {
			t.Fatalf("Get(:b) = %v, want 2", got)
		}
		if got := site.Get(kw, other, "def"); got != "def" {
			t.Fatalf("Get(:b) on a map without it = %v, want def", got)
		}
	}
	if got := site.Get(kw, nil, "def"); got != "def" {
		t.Fatalf("Get(:b) on nil = %v, want def", got)
	}
}

func TestKeywordSiteRecord(t *testing.T) {
	var site KeywordSite
	recordType := InternRecordType("keywordsite.test", "Point", "x", "y")
	point := NewRecord(recordType, int64(1), int64(2))
	x, y, z := NewKeyword("x"), NewKeyword("y"), NewKeyword("z")
	for i := 0; i < 3; i++ {
		if got := site.Get(y, point, nil); got != int64(2) {
			t.Fatalf("Get(:y) = %v, want 2", got)
		}
	}
	if site.entries[0].Load().record != recordType {
		t.Fatalf("record type not cached")
	}
	var xSite, zSite KeywordSite
	if got := xSite.Get(x, point, nil); got != int64(1) {
		t.Fatalf("Get(:x) = %v, want 1", got)
	}
	if got := zSite.Get(z, point, "def"); got != "def" {
		t.Fatalf("Get(:z) = %v, want def", got)
	}
	ext := point.Assoc(z, int64(3)).(RecordValue)
	if got := zSite.Get(z, ext, "def"); got != int64(3) {
		t.Fatalf("Get(:z) on the extended record = %v, want 3", got)
	}
}
