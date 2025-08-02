package lang

import (
	"fmt"
	"sort"
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
	// Handle nil cases first
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
	xNum, xIsNum := AsNumber(x)
	yNum, yIsNum := AsNumber(y)
	if xIsNum && yIsNum {
		return Numbers.Compare(xNum, yNum)
	}

	// Handle strings
	if xStr, xOk := x.(string); xOk {
		if yStr, yOk := y.(string); yOk {
			if xStr < yStr {
				return -1
			} else if xStr > yStr {
				return 1
			}
			return 0
		}
	}

	// Handle keywords
	if xKw, xOk := x.(Keyword); xOk {
		if yKw, yOk := y.(Keyword); yOk {
			return Compare(xKw.String(), yKw.String())
		}
	}

	// Handle symbols
	if xSym, xOk := x.(*Symbol); xOk {
		if ySym, yOk := y.(*Symbol); yOk {
			// Compare namespace first
			nsComp := Compare(xSym.Namespace(), ySym.Namespace())
			if nsComp != 0 {
				return nsComp
			}
			// Then compare name
			return Compare(xSym.Name(), ySym.Name())
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

	// Handle vectors (including MapEntry which is a vector)
	if xVec, xOk := x.(IPersistentVector); xOk {
		if yVec, yOk := y.(IPersistentVector); yOk {
			xCount := xVec.Count()
			yCount := yVec.Count()
			minCount := xCount
			if yCount < minCount {
				minCount = yCount
			}
			// Compare element by element
			for i := 0; i < minCount; i++ {
				cmp := Compare(xVec.Nth(i), yVec.Nth(i))
				if cmp != 0 {
					return cmp
				}
			}
			// If all compared elements are equal, shorter vector is less
			if xCount < yCount {
				return -1
			} else if xCount > yCount {
				return 1
			}
			return 0
		}
	}

	// If we can't compare, panic with an error
	panic(NewIllegalArgumentError(fmt.Sprintf("Cannot compare %T with %T", x, y)))
}
