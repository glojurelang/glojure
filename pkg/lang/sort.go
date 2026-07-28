package lang

import (
	"fmt"
	"strings"
)

// SortSlice performs an in-place stable sort on the given array using the provided comparator.
// This matches java.util.Arrays.sort semantics:
// - Stable sort (equal elements maintain their relative order)
// - In-place modification of the array
// - Comparator returns -1 for less than, 0 for equal, 1 for greater than
func SortSlice(slice []any, comp any) {
	// comp is a Clojure function that acts as a comparator
	compFn, ok := comp.(IFn)
	if !ok {
		panic(NewIllegalArgumentError("Comparator must be a function"))
	}

	stableSortAny(slice, func(x, y any) bool {
		// Call the comparator function with the two elements
		result := Apply2(compFn, x, y)

		// Handle both boolean and numeric comparators
		// Boolean comparator: returns true if i < j
		// Numeric comparator: returns negative if i < j
		if boolResult, ok := result.(bool); ok {
			return boolResult
		}

		// Numeric comparator returns:
		// -1 if first arg is less than second
		//  0 if args are equal
		//  1 if first arg is greater than second
		// We return true for "less than" case
		resultInt, ok := AsInt(result)
		if !ok {
			panic(NewIllegalArgumentError(fmt.Sprintf("Comparator must return a boolean or number, got %T", result)))
		}
		return resultInt < 0
	})
}

func stableSortAny(values []any, less func(any, any) bool) {
	if len(values) < 2 {
		return
	}

	// Match the adaptive behavior Clojure gets from its object-array sort for
	// the two most useful natural runs. The reverse path restores the original
	// order inside comparator-equal groups, so stability is preserved.
	ordered := true
	for i := 1; i < len(values); i++ {
		if less(values[i], values[i-1]) {
			ordered = false
			break
		}
	}
	if ordered {
		return
	}
	reverseOrdered := true
	for i := 1; i < len(values); i++ {
		if less(values[i-1], values[i]) {
			reverseOrdered = false
			break
		}
	}
	if reverseOrdered {
		reverseAny(values)
		runStart := 0
		for i := 1; i <= len(values); i++ {
			if i == len(values) || less(values[i-1], values[i]) {
				reverseAny(values[runStart:i])
				runStart = i
			}
		}
		return
	}

	// Clojure's object-array sort uses temporary merge storage. Do the same
	// for Glojure's []any representation instead of paying the extra swaps
	// required by Go's allocation-free symmetric stable merge.
	scratch := make([]any, len(values))
	source, destination := values, scratch
	sourceIsValues := true

	for width := 1; width < len(values); width *= 2 {
		for start := 0; start < len(values); start += 2 * width {
			middle := min(start+width, len(values))
			end := min(start+2*width, len(values))
			left, right := start, middle
			for output := start; output < end; output++ {
				if right >= end ||
					(left < middle && !less(source[right], source[left])) {
					destination[output] = source[left]
					left++
				} else {
					destination[output] = source[right]
					right++
				}
			}
		}
		source, destination = destination, source
		sourceIsValues = !sourceIsValues
	}

	if !sourceIsValues {
		copy(values, source)
	}
}

func reverseAny(values []any) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

// Compare implements Clojure's compare function.
// Returns a negative number, zero, or a positive number when x is logically
// 'less than', 'equal to', or 'greater than' y.
// Handles nil values (nil is less than everything except nil).
func Compare(x, y any) int {
	// Keep the overwhelmingly common homogeneous integer comparison on a
	// concrete path. This avoids generic interface equality, nil reflection,
	// and numeric dispatch.
	if xInt, ok := x.(int64); ok {
		if yInt, ok := y.(int64); ok {
			switch {
			case xInt < yInt:
				return -1
			case xInt > yInt:
				return 1
			default:
				return 0
			}
		}
	}

	// Identity check
	if x == y {
		return 0
	}

	// Handle nil cases
	if IsNil(x) {
		if IsNil(y) {
			return 0
		}
		return -1
	}
	if IsNil(y) {
		return 1
	}

	// Handle other numeric combinations using the Numbers.Compare method.
	if IsNumber(x) {
		return Numbers.Compare(x, y)
	}

	// Check if x implements Comparer interface
	if xComp, ok := x.(Comparer); ok {
		return xComp.Compare(y)
	}

	// Handle strings (built-in type, doesn't implement Comparer)
	if xStr, xOk := x.(string); xOk {
		if yStr, yOk := y.(string); yOk {
			return strings.Compare(xStr, yStr)
		}
	}

	// Handle characters
	if xChar, xOk := x.(Char); xOk {
		if yChar, yOk := y.(Char); yOk {
			if xChar < yChar {
				return -1
			} else if xChar > yChar {
				return 1
			}
			return 0
		}
	}

	// Default error - cannot compare
	panic(NewIllegalArgumentError(fmt.Sprintf("%T cannot be cast to Comparable", x)))
}

// LenientCompare is like Compare but falls back to string comparison
// for incompatible types instead of panicking. Used internally by
// sorted collections that may contain mixed types.
func LenientCompare(x, y any) (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = strings.Compare(ToString(x), ToString(y))
		}
	}()
	return Compare(x, y)
}
