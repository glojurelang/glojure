package lang

import "testing"

func TestArrayMapEquivalenceAcrossLayoutsAndDeltas(t *testing.T) {
	a, b := NewKeyword("a"), NewKeyword("b")
	shape := NewKeywordMapShape("a", "b")
	base := NewStaticKeywordMap(shape, int64(1), nil)
	updated := base.Assoc(a, int64(2)).(IPersistentMap)
	for _, test := range []struct {
		left, right any
		want        bool
	}{
		{base, NewStaticKeywordMap(shape, int64(1), nil), true},
		{base, NewStaticKeywordMap(NewKeywordMapShape("b", "a"), nil, int64(1)), true},
		{base, NewStaticKeywordMap(NewKeywordMapShape("a", "c"), int64(1), nil), false},
		{base, NewMap(b, nil, a, int64(1)), true},
		{base, NewPersistentHashMap(b, nil, a, int64(1)), true},
		{base, updated, false},
		{updated, NewMap(a, int64(2), b, nil), true},
		{base.(IObj).WithMeta(NewMap(a, true)), base, true},
		{NewMap(a, nil), NewMap(b, nil), false},
		{NewMap(a, base), NewMap(a, NewMap(a, int64(1), b, nil)), true},
	} {
		if got := Equals(test.left, test.right); got != test.want {
			t.Fatalf("equality of %v and %v = %v, want %v", test.left, test.right, got, test.want)
		}
		if got := Equals(test.right, test.left); got != test.want {
			t.Fatal("map equality was asymmetric")
		}
	}
}
