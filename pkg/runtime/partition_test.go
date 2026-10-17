package runtime

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestNativePartitionGroupsAndTrailingValues(t *testing.T) {
	for _, size := range []int64{1, 2, 3, 31, 32, 33, 100, 1000000} {
		for _, count := range []int64{0, 1, 2, 3, 31, 32, 33, 100} {
			source := lang.NewLongRange(0, count, 1)
			groups := nativePartition(size, source)
			groupCount := int64(0)
			for seq := lang.Seq(groups); seq != nil; seq = seq.Next() {
				group := seq.First()
				if _, counted := group.(lang.Counted); counted {
					t.Fatal("partition group changed counted? semantics")
				}
				values := lang.ToSlice(group)
				if int64(len(values)) != size {
					t.Fatalf("partition %d/%d has %d items", size, count, len(values))
				}
				for i, value := range values {
					if value != groupCount*size+int64(i) {
						t.Fatalf("partition value = %v", value)
					}
				}
				groupCount++
			}
			if groupCount != count/size {
				t.Fatalf("partition %d/%d has %d groups", size, count, groupCount)
			}
		}
	}
}

func TestNativePartitionLazinessAndCaching(t *testing.T) {
	calls := 0
	var input func(int64) lang.ISeq
	input = func(n int64) lang.ISeq {
		return lang.NewLazySeq(func() any {
			calls++
			return lang.NewCons(n, input(n+1))
		})
	}
	groups := nativePartition(2, input(0))
	if calls != 0 {
		t.Fatal("partition realized its source eagerly")
	}
	for i := 0; i < 2; i++ {
		if lang.Count(groups.First()) != 2 || calls != 2 {
			t.Fatalf("first group made %d input calls, want 2", calls)
		}
	}
	if lang.Count(groups.Next().First()) != 2 || calls != 4 {
		t.Fatalf("second group made %d input calls, want 4", calls)
	}
}
