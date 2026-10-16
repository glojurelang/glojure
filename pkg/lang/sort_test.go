package lang

import (
	"reflect"
	"testing"
)

func TestSortSliceStableComparators(t *testing.T) {
	type item struct {
		key  int
		name string
	}
	for _, numeric := range []bool{false, true} {
		values := []any{item{2, "a"}, item{1, "b"}, item{2, "c"}}
		compare := FnFunc2(func(a, b any) any {
			left, right := a.(item).key, b.(item).key
			if numeric {
				return left - right
			}
			return left < right
		})
		SortSlice(values, compare)
		want := []any{item{1, "b"}, item{2, "a"}, item{2, "c"}}
		if !reflect.DeepEqual(values, want) {
			t.Fatalf("sorted = %v, want %v", values, want)
		}
	}
}
