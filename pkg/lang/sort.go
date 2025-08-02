package lang

import (
	"fmt"
	"sort"
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

	// Use sort.SliceStable for stable sorting (maintains relative order of equal elements)
	sort.SliceStable(slice, func(i, j int) bool {
		// Call the comparator function with the two elements
		result := compFn.Invoke(slice[i], slice[j])

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

// Compare implements Clojure's compare function.
// Returns a negative number, zero, or a positive number when x is logically
// 'less than', 'equal to', or 'greater than' y.
// Handles nil values (nil is less than everything except nil).
func Compare(x, y any) int {
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

	// Handle numbers using the Numbers.Compare method
	if xNum, xIsNum := AsNumber(x); xIsNum {
		return Numbers.Compare(xNum, y)
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
