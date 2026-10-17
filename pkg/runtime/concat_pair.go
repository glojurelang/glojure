package runtime

import "github.com/glojurelang/glojure/pkg/lang"

type concatPairProducer struct {
	lang.LazySeq
	left, right any
}

func nativeConcatPair(left, right any) lang.ISeq {
	state := &concatPairProducer{left: left, right: right}
	state.InitProducer(state)
	return &state.LazySeq
}

func (p *concatPairProducer) RealizeSeq() any {
	var result any = p.right
	if seq := lang.Seq(p.left); seq != nil {
		if chunked, ok := seq.(lang.IChunkedSeq); ok {
			chunk := chunked.ChunkedFirst()
			more := nativeConcatPair(chunked.ChunkedMore(), p.right)
			if chunk.Count() == 0 {
				result = more
			} else {
				result = lang.NewChunkedCons(chunk, more)
			}
		} else {
			result = lang.NewCons(seq.First(), nativeConcatPair(seq.More(), p.right))
		}
	}
	p.left, p.right = nil, nil
	return result
}
