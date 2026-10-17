package runtime

import (
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
)

func installNativeStringJoin(namespace *lang.Namespace) {
	join := namespace.FindInternedVar(lang.NewSymbol("join"))
	if join == nil {
		return
	}
	original := join.Get()
	core := lang.FindNamespace(lang.NewSymbol("clojure.core"))
	mapVar := core.FindInternedVar(lang.NewSymbol("map"))
	strVar := core.FindInternedVar(lang.NewSymbol("str"))
	installFixedArityCoreFunction(namespace, "join", 2,
		newNativeStringJoin(original, mapVar, strVar))
}

func newNativeStringJoin(original any, mapVar, strVar *lang.Var) lang.FnFunc2 {
	mapVersion, strVersion := mapVar.RootVersion(), strVar.RootVersion()
	return lang.FnFunc2(func(separator, coll any) any {
		if mapVar.RootVersion() != mapVersion || strVar.RootVersion() != strVersion ||
			mapVar.IsDynamic() || strVar.IsDynamic() {
			return lang.Apply2(original, separator, coll)
		}
		// The source implementation captures str in a lazy map, then converts
		// the separator before realizing any input elements.
		sep := nativeStrValue(separator)
		var builder strings.Builder
		first := true
		write := func(value any) {
			if !first {
				builder.WriteString(sep)
			}
			first = false
			builder.WriteString(nativeStrValue(value))
		}
		for seq := lang.Seq(coll); seq != nil; {
			if chunked, ok := seq.(lang.IChunkedSeq); ok {
				chunk := chunked.ChunkedFirst()
				for i := 0; i < chunk.Count(); i++ {
					write(chunk.Nth(i))
				}
				seq = lang.Seq(chunked.ChunkedMore())
			} else {
				write(seq.First())
				seq = lang.Seq(seq.More())
			}
		}
		return builder.String()
	})
}
