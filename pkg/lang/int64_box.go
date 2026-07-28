package lang

const (
	minCachedInt64 = -128
	maxCachedInt64 = 4096
	minCachedInt   = -128
	maxCachedInt   = 127
)

var cachedInt64Values = func() []any {
	values := make([]any, maxCachedInt64-minCachedInt64+1)
	for i := range values {
		values[i] = int64(i + minCachedInt64)
	}
	return values
}()

var cachedIntValues = func() []any {
	values := make([]any, maxCachedInt-minCachedInt+1)
	for i := range values {
		values[i] = int(i + minCachedInt)
	}
	return values
}()

// BoxInt64 returns a shared interface value for common integers. Go otherwise
// allocates when an int64 escapes through an interface, which is frequent in
// the dynamic numeric path used by both interpreted and generated code.
func BoxInt64(value int64) any {
	if value >= minCachedInt64 && value <= maxCachedInt64 {
		return cachedInt64Values[value-minCachedInt64]
	}
	return value
}

// BoxInt is the native-int counterpart to BoxInt64. Host functions commonly
// return int, and generated Clojure functions must expose those results
// through any without allocating a fresh box for common values.
func BoxInt(value int) any {
	if value >= minCachedInt && value <= maxCachedInt {
		return cachedIntValues[value-minCachedInt]
	}
	return value
}
