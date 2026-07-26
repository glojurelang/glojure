package lang

import (
	"runtime"
	"strconv"
	"testing"
)

func TestSmallMapAssocUsesIndependentInlineStorage(t *testing.T) {
	keys := []Keyword{
		NewKeyword("a"),
		NewKeyword("b"),
		NewKeyword("c"),
		NewKeyword("d"),
	}
	original := NewMap(
		keys[0], int64(1),
		keys[1], int64(2),
		keys[2], int64(3),
		keys[3], int64(4),
	).(*Map)

	updated := original.Assoc(keys[1], int64(20)).(*Map)
	if got := original.ValAt(keys[1]); got != int64(2) {
		t.Fatalf("original value = %v, want 2", got)
	}
	if got := updated.ValAt(keys[1]); got != int64(20) {
		t.Fatalf("updated value = %v, want 20", got)
	}

	var result Associative
	key := any(keys[0])
	value := any(int64(10))
	if got := testing.AllocsPerRun(1_000, func() {
		result = original.Assoc(key, value)
	}); got != 1 {
		t.Fatalf("small-map assoc allocated %v objects per call, want 1", got)
	}
	runtime.KeepAlive(result)
}

func TestKeywordMapsUseClojureArrayMapThreshold(t *testing.T) {
	keyVals := make([]any, 0, arrayMapKeywordThreshold)
	for i := 0; i < arrayMapKeywordThreshold/2; i++ {
		keyVals = append(keyVals, NewKeyword("k"+strconv.Itoa(i)), int64(i))
	}

	m, ok := NewMap(keyVals...).(*Map)
	if !ok {
		t.Fatalf("keyword map at threshold has type %T, want *Map", NewMap(keyVals...))
	}
	if got := m.Count(); got != arrayMapKeywordThreshold/2 {
		t.Fatalf("Count() = %d, want %d", got, arrayMapKeywordThreshold/2)
	}

	overflow := m.Assoc(NewKeyword("overflow"), int64(1))
	if _, ok := overflow.(*PersistentHashMap); !ok {
		t.Fatalf("keyword map above threshold has type %T, want *PersistentHashMap", overflow)
	}
}

func TestNonKeywordMapUsesGeneralArrayMapThreshold(t *testing.T) {
	keyVals := make([]any, 0, arrayMapHashThreshold+2)
	for i := 0; i < arrayMapHashThreshold/2; i++ {
		keyVals = append(keyVals, i, i)
	}

	m, ok := NewMap(keyVals...).(*Map)
	if !ok {
		t.Fatalf("map at threshold has type %T, want *Map", NewMap(keyVals...))
	}

	overflow := m.Assoc(arrayMapHashThreshold/2, true)
	if _, ok := overflow.(*PersistentHashMap); !ok {
		t.Fatalf("map above threshold has type %T, want *PersistentHashMap", overflow)
	}
}
