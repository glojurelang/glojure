package lang

import (
	"fmt"
	"reflect"
)

func SliceSet(slc any, idx int, val any) {
	slcVal := reflect.ValueOf(slc)
	if val == nil {
		slcVal.Index(idx).Set(reflect.Zero(slcVal.Type().Elem()))
		return
	}
	valVal := reflect.ValueOf(val)
	// coerce valVal to the element type of slcVal
	if valVal.Type() != slcVal.Type().Elem() {
		if valVal.Type().ConvertibleTo(slcVal.Type().Elem()) {
			valVal = valVal.Convert(slcVal.Type().Elem())
		} else {
			panic(NewIllegalArgumentError(fmt.Sprintf("Cannot convert %T to %s", val, slcVal.Type().Elem().String())))
		}
	}
	slcVal.Index(idx).Set(valVal)
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

	// Handle string - convert each Unicode code point to a character value.
	if s, ok := x.(string); ok {
		runes := []rune(s)
		res := make([]any, len(runes))
		for i, ch := range runes {
			res[i] = NewChar(ch)
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

// SeqToTypedArray implements Clojure's one- and two-argument
// clojure.lang.RT/seqToTypedArray overloads using Go slices.
func SeqToTypedArray(args ...any) any {
	if len(args) != 1 && len(args) != 2 {
		panic(NewIllegalArgumentError(
			fmt.Sprintf("seqToTypedArray expects 1 or 2 arguments, got %d", len(args)),
		))
	}

	var typ reflect.Type
	var values []any
	if len(args) == 1 {
		values = seqToSlice(Seq(args[0]))
		if len(values) == 0 || values[0] == nil {
			typ = BuiltinTypes["any"]
		} else {
			typ = reflect.TypeOf(values[0])
		}
	} else {
		var ok bool
		typ, ok = args[0].(reflect.Type)
		if !ok {
			panic(NewIllegalArgumentError(
				fmt.Sprintf("array component type must be reflect.Type, got %T", args[0]),
			))
		}
		values = seqToSlice(Seq(args[1]))
	}

	result := reflect.MakeSlice(reflect.SliceOf(typ), len(values), len(values))
	for i, value := range values {
		coerced, err := coerceGoValue(typ, value)
		if err != nil {
			panic(NewIllegalArgumentError(
				fmt.Sprintf("cannot convert array element %d from %T to %s", i, value, typ),
			))
		}
		result.Index(i).Set(coerced)
	}
	return result.Interface()
}
