package lang

const maxCachedByteChar = 255

var cachedByteChars = func() []any {
	values := make([]any, maxCachedByteChar+1)
	for i := range values {
		values[i] = Char(i)
	}
	return values
}()

// BoxChar returns a shared interface value for characters represented by a
// string byte. Go otherwise allocates when a Char escapes through an
// interface, while every Glojure string indexing and sequence operation
// exposes byte values as Char values.
func BoxChar(value rune) any {
	if value >= 0 && value <= maxCachedByteChar {
		return cachedByteChars[value]
	}
	return Char(value)
}
