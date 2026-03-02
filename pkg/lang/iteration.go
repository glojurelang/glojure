package lang

import (
	"fmt"
	"reflect"
	"unicode/utf8"
)

// Nther is an interface for compound values whose elements can be
// accessed by index.
type Nther interface {
	Nth(int) (v interface{}, ok bool)
}

// MustNth returns the nth element of the vector. It panics if the
// index is out of range.
func MustNth(x interface{}, i int) interface{} {
	v, ok := Nth(x, i)
	if !ok {
		panic("index out of range")
	}
	return v
}

func Nth(x interface{}, n int) (interface{}, bool) {
	switch x := x.(type) {
	// Deprecate this
	case Nther:
		return x.Nth(n)
	case Indexed:
		val := x.NthDefault(n, notFound)
		if val == notFound {
			return nil, false
		}
		return val, true
	case ISeq:
		x = Seq(x)
		for i := 0; i <= n; i++ {
			if x == nil {
				return nil, false
			}
			if i == n {
				return x.First(), true
			}
			x = x.Next()
		}
	case string:
		if n < 0 {
			return nil, false
		}
		// Walk bytes, using single-byte increment for ASCII and
		// utf8.DecodeRuneInString only for multi-byte characters.
		// This is effectively O(1) per rune for pure-ASCII input.
		bytePos := 0
		for i := 0; i < n; i++ {
			if bytePos >= len(x) {
				return nil, false
			}
			if x[bytePos] < 0x80 {
				bytePos++
			} else {
				_, size := utf8.DecodeRuneInString(x[bytePos:])
				bytePos += size
			}
		}
		if bytePos >= len(x) {
			return nil, false
		}
		r, _ := utf8.DecodeRuneInString(x[bytePos:])
		return NewChar(r), true
	}

	if seq := Seq(x); seq != nil {
		if seq == x {
			panic(fmt.Errorf("unexpected Seq result equal to input"))
		}
		return Nth(seq, n)
	}

	reflectVal := reflect.ValueOf(x)
	switch reflectVal.Kind() {
	case reflect.Array, reflect.Slice:
		if n < 0 || n >= reflectVal.Len() {
			return nil, false
		}
		return reflectVal.Index(n).Interface(), true
	}

	return nil, false
}
