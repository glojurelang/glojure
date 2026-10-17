//go:build !glj_aot_runtime

package runtime

import (
	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
	"testing"
)

func TestConstantCollectionsRetainRuntimeHostExportReads(t *testing.T) {
	host := &ast.Node{Op: ast.OpConst, Sub: &ast.ConstNode{
		Value: "compile-time placeholder", HostSymbol: lang.NewSymbol("os.Args"),
	}}
	vector := &ast.Node{Op: ast.OpVector, Sub: &ast.VectorNode{Items: []*ast.Node{host}}}
	set := &ast.Node{Op: ast.OpSet, Sub: &ast.SetNode{Items: []*ast.Node{host}}}
	key := &ast.Node{Op: ast.OpConst, Sub: &ast.ConstNode{Value: lang.NewKeyword("args")}}
	mapping := &ast.Node{Op: ast.OpMap, Sub: &ast.MapNode{Keys: []*ast.Node{key}, Vals: []*ast.Node{vector}}}
	for _, node := range []*ast.Node{host, vector, set, mapping} {
		if _, ok := constantCollectionValue(node); ok {
			t.Fatal("host export was frozen inside a collection")
		}
		if purePartitionFoldArgument(node) {
			t.Fatal("host export was classified as a constant fold argument")
		}
	}
}
