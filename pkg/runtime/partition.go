package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// The nonoverlapping partition arity can consume each group once instead of
// constructing take and drop pipelines and counting the temporary take seq.
type partitionProducer struct {
	lang.LazySeq
	size int64
	coll any
}

func nativePartition(size int64, coll any) lang.ISeq {
	state := &partitionProducer{size: size, coll: coll}
	state.InitProducer(state)
	return &state.LazySeq
}

func (p *partitionProducer) RealizeSeq() any {
	size, coll := p.size, p.coll

	var buffer [32]any
	values := buffer[:0:min(size, 32)]
	rest := coll
	for int64(len(values)) < size {
		seq := lang.Seq(rest)
		if seq == nil {
			p.coll = nil
			return nil
		}
		values = append(values, seq.First())
		rest = seq.More()
	}
	// Like take's result, a partition group is an uncounted seq.
	group := lang.NewConsSequence(values)
	result := lang.NewCons(group, nativePartition(size, rest))
	p.coll = nil
	return result
}
