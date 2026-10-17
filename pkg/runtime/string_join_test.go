package runtime

import (
	"github.com/glojurelang/glojure/pkg/lang"
	"testing"
)

type joinChunkOnly struct{ lang.IChunkedSeq }

func (s *joinChunkOnly) Seq() lang.ISeq { return s }
func (*joinChunkOnly) Next() lang.ISeq  { panic("join must traverse chunks") }
func (*joinChunkOnly) More() lang.ISeq  { panic("join must traverse chunks") }

func TestNativeStringJoinChunksAndRootFallback(t *testing.T) {
	mapVar := lang.NewVar(lang.NSCore, lang.NewSymbol("map"))
	strVar := lang.NewVar(lang.NSCore, lang.NewSymbol("str"))
	mapFn := lang.FnFunc2(nativeMapSeq)
	mapVar.BindRoot(mapFn)
	strVar.BindRoot(nativeCoreStr{})
	fallback := lang.FnFunc2(func(separator, coll any) any { return "fallback" })
	join := newNativeStringJoin(fallback, mapVar, strVar)
	chunk := lang.NewSliceChunk([]any{"a", nil, lang.NewChar('λ')})
	source := &joinChunkOnly{lang.NewChunkedCons(chunk, nil)}
	if got := join("-", source); got != "a--λ" {
		t.Fatalf("chunked join = %v", got)
	}
	mapVar.BindRoot(mapFn)
	if join("-", nil) != "fallback" {
		t.Fatal("join ignored changed map root")
	}
	join = newNativeStringJoin(fallback, mapVar, strVar)
	strVar.BindRoot(nativeCoreStr{})
	if join("-", nil) != "fallback" {
		t.Fatal("join ignored changed str root")
	}
}
