package runtime

import "github.com/glojurelang/glojure/pkg/lang"

type PartitionFoldMode uint8

const (
	PartitionFoldReduce PartitionFoldMode = iota
	PartitionFoldSome
	PartitionFoldEvery
	PartitionFoldNotAny
	PartitionFoldNotEvery
)

// FoldPartition consumes a temporary positive, nonoverlapping partition.
// Groups remain persistent uncounted sequences, so a callback can retain
// them. Only the unobservable outer lazy sequence and its thunks disappear.
func FoldPartition(mode PartitionFoldMode, fn, initial any, size int64, coll any) any {
	var storage [32]any
	values := storage[:0:min(size, 32)]
	rest := coll
	for {
		values = values[:0]
		for int64(len(values)) < size {
			seq := lang.Seq(rest)
			if seq == nil {
				switch mode {
				case PartitionFoldReduce:
					return initial
				case PartitionFoldEvery, PartitionFoldNotAny:
					return true
				case PartitionFoldNotEvery:
					return false
				default:
					return nil
				}
			}
			values = append(values, seq.First())
			rest = seq.More()
		}
		group := lang.NewConsSequence(values)
		if mode == PartitionFoldReduce {
			initial = lang.Apply2(fn, initial, group)
			if lang.IsReduced(initial) {
				return initial.(lang.IDeref).Deref()
			}
			continue
		}
		value := lang.Apply1(fn, group)
		truthy := lang.IsTruthy(value)
		switch mode {
		case PartitionFoldSome:
			if truthy {
				return value
			}
		case PartitionFoldEvery:
			if !truthy {
				return false
			}
		case PartitionFoldNotAny:
			if truthy {
				return false
			}
		case PartitionFoldNotEvery:
			if !truthy {
				return true
			}
		}
	}
}
