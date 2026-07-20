package lang

const (
	minCachedInt64 = -128
	maxCachedInt64 = 4096
)

var cachedInt64Values = func() []any {
	values := make([]any, maxCachedInt64-minCachedInt64+1)
	for i := range values {
		values[i] = int64(i + minCachedInt64)
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
