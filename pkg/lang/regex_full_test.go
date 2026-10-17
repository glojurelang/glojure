package lang

import (
	"regexp"
	"sync"
	"testing"
)

func TestRegexpFullMatchExploresLongerAlternatives(t *testing.T) {
	pattern := regexp.MustCompile(`(a|ab)(b?)`)
	matcher := NewRegexpMatcher(pattern, "ab")
	if !matcher.Matches() || matcher.Group() != "ab" ||
		matcher.GroupInt(1) != "a" || matcher.GroupInt(2) != "b" {
		t.Fatal("full match lost an alternative or capture")
	}
	short := regexp.MustCompile(`a|ab`)
	if !NewRegexpMatcher(short, "ab").Matches() || short.FindString("ab") != "a" {
		t.Fatal("full matching either stopped early or changed find behavior")
	}
}

func TestRegexpFullMatchPreservesPOSIXFlags(t *testing.T) {
	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(`[^b]+`), regexp.MustCompilePOSIX(`[^b]+`),
	} {
		copy := pattern.Copy()
		copy.Longest()
		indices := copy.FindStringIndex("a\n")
		want := indices != nil && indices[0] == 0 && indices[1] == 2
		if got := NewRegexpMatcher(pattern, "a\n").Matches(); got != want {
			t.Fatalf("full matching changed regex flags: got %v, want %v", got, want)
		}
	}
}

func TestRegexpFullMatchCacheConcurrentReaders(t *testing.T) {
	pattern := regexp.MustCompile(`a|ab`)
	var workers sync.WaitGroup
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 10 {
				if !NewRegexpMatcher(pattern, "ab").Matches() {
					t.Error("concurrent full match failed")
				}
			}
		}()
	}
	workers.Wait()
}
