package classes

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestCommonClassesAreRegistered(t *testing.T) {
	for _, name := range []string{
		"Object", "Comparable", "Set", "List", "Map", "Keyword", "IDeref", "Associative",
		"Counted", "IAtom", "IBlockingDeref", "IChunkedSeq", "IEditableCollection", "IFn",
		"ILookup", "IMapEntry", "IMeta", "IObj", "IPending", "IPersistentCollection",
		"IPersistentMap", "IPersistentSet", "IPersistentVector", "IRecord", "IReduceInit",
		"IRef", "ISeq", "Indexed", "Named", "Reversible", "Seqable", "Sequential", "Sorted",
		"Ratio", "Symbol", "ExceptionInfo", "Atom", "Volatile",
	} {
		if _, ok := pkgmap.HostClass(name); !ok {
			t.Fatalf("host class %s is not registered", name)
		}
	}
}
