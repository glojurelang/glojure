package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// VectorMap consumes a temporary map sequence directly into its vector.
// Traverse through Seq rather than the source's reduction protocol: map
// promises sequence traversal even when a custom collection reduces differently.
func VectorMap(fn, coll any) any {
	values := make([]any, 0)
	for seq := lang.Seq(coll); seq != nil; {
		if chunked, ok := seq.(lang.IChunkedSeq); ok {
			chunk := chunked.ChunkedFirst()
			for i := 0; i < chunk.Count(); i++ {
				values = append(values, lang.Apply1(fn, chunk.Nth(i)))
			}
			seq = lang.Seq(chunked.ChunkedMore())
		} else {
			values = append(values, lang.Apply1(fn, seq.First()))
			seq = lang.Seq(seq.More())
		}
	}
	return lang.NewVector(values...)
}

// VectorMapcatSeq avoids temporary map-entry vectors and lazy concatenation
// when the default seq function flattens an ordinary map into a vector.
// Other collections keep mapcat's existing chunking and sequence behavior.
func VectorMapcatSeq(seqFn, coll any) any {
	if m, ok := coll.(*lang.Map); ok {
		return m.EntryVector()
	}
	return lang.CreateLazilyPersistentVector(nativeMapcatSeq(seqFn, coll))
}
