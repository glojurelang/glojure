package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
	"testing"
)

func TestPartitionFoldRetainsPersistentGroupsAndDropsIncompleteTail(t *testing.T) {
	reducer := lang.FnFunc2(func(acc, group any) any {
		if _, counted := group.(lang.Counted); counted {
			t.Fatal("partition group became counted")
		}
		return acc.(lang.IPersistentVector).Cons(group)
	})
	for _, size := range []int64{1, 2, 3, 31, 32, 33} {
		for _, count := range []int64{0, 1, 2, 3, 32, 33, 100} {
			source := lang.NewLongRange(0, count, 1)
			got := FoldPartition(PartitionFoldReduce, reducer, lang.NewVector(), size, source)
			var want any = lang.NewVector()
			for seq := lang.Seq(nativePartition(size, source)); seq != nil; seq = seq.Next() {
				want = reducer(want, seq.First())
			}
			if !lang.Equals(got, want) {
				t.Fatalf("fold %d/%d changed retained groups: %v", size, count, got)
			}
		}
	}
}

func TestPartitionFoldStopsBeforeInputTail(t *testing.T) {
	source := lang.NewCons(1, lang.NewCons(2, lang.NewLazySeq(func() any { panic("tail forced") })))
	for _, mode := range []PartitionFoldMode{PartitionFoldReduce, PartitionFoldSome,
		PartitionFoldEvery, PartitionFoldNotAny, PartitionFoldNotEvery} {
		var fn any = lang.FnFunc1(func(any) any { return mode == PartitionFoldSome || mode == PartitionFoldNotAny })
		if mode == PartitionFoldReduce {
			fn = lang.FnFunc2(func(acc, group any) any { return lang.NewReduced(42) })
		}
		got := FoldPartition(mode, fn, nil, 2, source)
		if mode == PartitionFoldReduce && got != 42 {
			t.Fatal("fold lost reduced result")
		}
	}
}
