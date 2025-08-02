package lang

import (
	"fmt"
	"reflect"
)

func SliceSet(slc any, idx int, val any) {
	slcVal := reflect.ValueOf(slc)
	slcVal.Index(idx).Set(reflect.ValueOf(val))
}

func ToSlice(x any) []any {
	// Handle nil - Clojure returns empty array for nil
	if IsNil(x) {
		return []any{}
	}

	// Handle []any - return as-is
	if slice, ok := x.([]any); ok {
		return slice
	}

	// Handle IPersistentVector
	if vec, ok := x.(IPersistentVector); ok {
		count := vec.Count()
		res := make([]any, count)
		for i := 0; i < count; i++ {
			res[i] = vec.Nth(i)
		}
		return res
	}

	// Handle IPersistentMap - convert to array of MapEntry objects
	if m, ok := x.(IPersistentMap); ok {
		seq := m.Seq()
		res := make([]any, 0, m.Count())
		for seq != nil {
			res = append(res, seq.First()) // Each element is a MapEntry
			seq = seq.Next()
		}
		return res
	}

	// Handle Set - convert to array of values
	if s, ok := x.(*Set); ok {
		seq := s.Seq()
		res := make([]any, 0, s.Count())
		for seq != nil {
			res = append(res, seq.First())
			seq = seq.Next()
		}
		return res
	}

	// Handle string - convert to character array
	if s, ok := x.(string); ok {
		runes := []rune(s) // Important: use runes for proper Unicode handling
		res := make([]any, len(runes))
		for i, ch := range runes {
			res[i] = NewChar(ch) // Convert each rune to Char
		}
		return res
	}

	// Handle ISeq
	if s, ok := x.(ISeq); ok {
		res := make([]interface{}, 0, Count(x))
		for s := Seq(s); s != nil; s = s.Next() {
			res = append(res, s.First())
		}
		return res
	}

	// Handle reflection-based slice/array
	xVal := reflect.ValueOf(x)
	if xVal.Kind() == reflect.Slice || xVal.Kind() == reflect.Array {
		res := make([]interface{}, xVal.Len())
		for i := 0; i < xVal.Len(); i++ {
			res[i] = xVal.Index(i).Interface()
		}
		return res
	}

	// Error with Clojure-style message
	panic(NewIllegalArgumentError(fmt.Sprintf("Unable to convert: %T to Object[]", x)))
}
