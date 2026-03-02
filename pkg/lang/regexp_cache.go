package lang

import (
	"regexp"
	"sync"
)

var regexpCache sync.Map // string -> *regexp.Regexp

// CachedCompileRegexp compiles pattern on first call and returns the cached
// *regexp.Regexp on subsequent calls. This is a drop-in replacement for
// regexp.MustCompile when the same patterns are compiled repeatedly.
func CachedCompileRegexp(pattern string) *regexp.Regexp {
	if cached, ok := regexpCache.Load(pattern); ok {
		return cached.(*regexp.Regexp)
	}
	re := regexp.MustCompile(pattern)
	actual, _ := regexpCache.LoadOrStore(pattern, re)
	return actual.(*regexp.Regexp)
}
