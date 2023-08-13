package main

import (
	// Bootstrap the runtime

	"fmt"

	_ "github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
)

func main() {
	coreNamespace := lang.FindNamespace(lang.NewSymbol("glojure.core"))
	mappings := coreNamespace.Mappings()
	for s := lang.Seq(mappings); s != nil; s = s.Next() {
		entry := s.First().(lang.IMapEntry)
		val := entry.Val().(*lang.Var)
		fmt.Printf("%T\n", val.Get())
	}
}
