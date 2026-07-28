package lang

import (
	"slices"
	"sort"
	"testing"
)

func TestSortSliceComparators(t *testing.T) {
	tests := []struct {
		name string
		comp any
		want []any
	}{
		{
			name: "numeric",
			comp: FnFunc2(func(x, y any) any {
				return Compare(x, y)
			}),
			want: []any{int64(1), int64(2), int64(2), int64(3)},
		},
		{
			name: "boolean",
			comp: FnFunc2(func(x, y any) any {
				return x.(int64) > y.(int64)
			}),
			want: []any{int64(3), int64(2), int64(2), int64(1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := []any{int64(2), int64(1), int64(3), int64(2)}
			SortSlice(got, tt.comp)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("SortSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortSliceStable(t *testing.T) {
	type item struct {
		key   int
		order int
	}
	got := []any{
		item{key: 2, order: 0},
		item{key: 1, order: 1},
		item{key: 2, order: 2},
		item{key: 1, order: 3},
	}
	SortSlice(got, FnFunc2(func(x, y any) any {
		return int64(x.(item).key - y.(item).key)
	}))
	want := []any{
		item{key: 1, order: 1},
		item{key: 1, order: 3},
		item{key: 2, order: 0},
		item{key: 2, order: 2},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SortSlice() = %v, want stable order %v", got, want)
	}
}

func TestSortSliceReverseRunRemainsStable(t *testing.T) {
	type item struct {
		key   int64
		order int
	}
	got := []any{
		item{key: 1, order: 0},
		item{key: 1, order: 1},
		item{key: 2, order: 2},
		item{key: 2, order: 3},
		item{key: 3, order: 4},
	}
	SortSlice(got, FnFunc2(func(x, y any) any {
		return x.(item).key > y.(item).key
	}))
	want := []any{
		item{key: 3, order: 4},
		item{key: 2, order: 2},
		item{key: 2, order: 3},
		item{key: 1, order: 0},
		item{key: 1, order: 1},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SortSlice() = %v, want stable reverse order %v", got, want)
	}
}

func TestSortSliceMatchesStableReference(t *testing.T) {
	type item struct {
		key   int64
		order int
	}
	got := make([]any, 513)
	want := make([]item, len(got))
	for i := range got {
		value := item{
			key:   int64((i*48271 + i*i*31) % 29),
			order: i,
		}
		got[i] = value
		want[i] = value
	}
	sort.SliceStable(want, func(i, j int) bool {
		return want[i].key < want[j].key
	})
	SortSlice(got, FnFunc2(func(x, y any) any {
		return x.(item).key < y.(item).key
	}))
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortSlice()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestSortSliceFixedComparatorDoesNotAllocatePerCall(t *testing.T) {
	values := make([]any, 256)
	comp := FnFunc2(func(x, y any) any {
		return x.(int64) < y.(int64)
	})
	allocs := testing.AllocsPerRun(100, func() {
		for i := range values {
			values[i] = int64(len(values) - i)
		}
		SortSlice(values, comp)
	})
	if allocs > 8 {
		t.Fatalf("SortSlice allocated %v times for a fixed comparator; want only fixed sort setup allocations", allocs)
	}
}

func TestCompareInt64DoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		if got := Compare(int64(1), int64(2)); got != -1 {
			t.Fatalf("Compare(1, 2) = %d, want -1", got)
		}
	})
	if allocs != 0 {
		t.Fatalf("Compare(int64, int64) allocated %v times, want zero", allocs)
	}
}
