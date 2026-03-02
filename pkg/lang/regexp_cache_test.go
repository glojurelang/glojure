package lang

import (
	"regexp"
	"sync"
	"testing"
)

// TestCachedCompileRegexpReturnsCorrectRegexp verifies the cached regexp
// matches the same strings as regexp.MustCompile.
func TestCachedCompileRegexpReturnsCorrectRegexp(t *testing.T) {
	re := CachedCompileRegexp(`^[a-z]+$`)
	if !re.MatchString("hello") {
		t.Error("expected match for 'hello'")
	}
	if re.MatchString("Hello") {
		t.Error("expected no match for 'Hello'")
	}
}

// TestCachedCompileRegexpReturnsSamePointer verifies repeated calls with the
// same pattern return the same *regexp.Regexp pointer (cache hit).
func TestCachedCompileRegexpReturnsSamePointer(t *testing.T) {
	pattern := `^[0-9]+$`
	re1 := CachedCompileRegexp(pattern)
	re2 := CachedCompileRegexp(pattern)
	if re1 != re2 {
		t.Errorf("expected same pointer, got %p and %p", re1, re2)
	}
}

// TestCachedCompileRegexpDifferentPatterns verifies distinct patterns produce
// distinct *regexp.Regexp values.
func TestCachedCompileRegexpDifferentPatterns(t *testing.T) {
	re1 := CachedCompileRegexp(`^[a-z]$`)
	re2 := CachedCompileRegexp(`^[A-Z]$`)
	if re1 == re2 {
		t.Error("expected different pointers for different patterns")
	}
}

// TestCachedCompileRegexpConcurrent verifies the cache is safe under concurrent
// access from multiple goroutines.
func TestCachedCompileRegexpConcurrent(t *testing.T) {
	pattern := `^[a-zA-Z0-9]+$`
	const goroutines = 50
	results := make([]*regexp.Regexp, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			results[i] = CachedCompileRegexp(pattern)
		}()
	}
	wg.Wait()

	first := results[0]
	for i, re := range results {
		if re != first {
			t.Errorf("goroutine %d: got different pointer %p, want %p", i, re, first)
		}
	}
}
