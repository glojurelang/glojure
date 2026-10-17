package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// The nonoverlapping partition arity can consume each group once instead of
// constructing take and drop pipelines and counting the temporary take seq.
func nativePartition(size int64, coll any) lang.ISeq {
	return lang.NewLazySeq(func() any {
		values := make([]any, 0, min(size, 32))
		rest := coll
		for int64(len(values)) < size {
			seq := lang.Seq(rest)
			if seq == nil {
				return nil
			}
			values = append(values, seq.First())
			rest = seq.More()
		}
		// Like take's result, a partition group is an uncounted seq.
		var group lang.ISeq = lang.NewList()
		for i := len(values) - 1; i >= 0; i-- {
			group = lang.NewCons(values[i], group)
		}
		return lang.NewCons(group, nativePartition(size, rest))
	})
}
