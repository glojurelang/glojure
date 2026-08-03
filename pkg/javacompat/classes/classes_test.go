package classes

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func TestCommonClassesAreRegistered(t *testing.T) {
	for _, name := range []string{
		"Object", "Byte", "Short", "Float", "Comparable", "CharSequence", "Number", "Class", "ClassNotFoundException", "URL", "URLClassLoader", "NumberFormatException", "Set", "Collection", "List", "Map", "Keyword", "IDeref", "Associative",
		"Counted", "IAtom", "IBlockingDeref", "IChunkedSeq", "IEditableCollection", "IFn",
		"ILookup", "IMapEntry", "IMeta", "IObj", "IPending", "IPersistentCollection",
		"IPersistentMap", "IPersistentSet", "IPersistentVector", "IRecord", "IReduce", "IReduceInit",
		"IRef", "ISeq", "Indexed", "Named", "Reversible", "Seqable", "Sequential", "Sorted",
		"Ratio", "Symbol", "Namespace", "Var", "BigInteger", "BigInt", "AtomicInteger", "AtomicLong", "PersistentVector", "ExceptionInfo", "Atom", "Volatile",
	} {
		if _, ok := pkgmap.HostClass(name); !ok {
			t.Fatalf("host class %s is not registered", name)
		}
	}
}

func TestBigIntegerSignumMagnitudeConstructor(t *testing.T) {
	value := newBigInteger(int64(1), []int8{-1}).(*big.Int)
	if got, want := value.Cmp(big.NewInt(255)), 0; got != want {
		t.Fatalf("BigInteger(1, [0xff]) = %v", value)
	}
	if got, want := value.Text(16), "ff"; got != want {
		t.Fatalf("hex = %q, want %q", got, want)
	}
}

func TestKeywordClassMatchesKeywordValues(t *testing.T) {
	class, ok := pkgmap.HostClass("Keyword")
	if !ok {
		t.Fatal("Keyword host class is not registered")
	}
	typ, ok := lang.ReflectType(class)
	if !ok {
		t.Fatalf("Keyword host class %T has no reflect type", class)
	}
	if typ != reflect.TypeOf(lang.NewKeyword("example")) {
		t.Fatalf("Keyword class type = %v, want %v",
			typ, reflect.TypeOf(lang.NewKeyword("example")))
	}
}

func TestNamedClassMatchesKeywordsAndSymbols(t *testing.T) {
	class, ok := pkgmap.HostClass("Named")
	if !ok {
		t.Fatal("Named host class is not registered")
	}
	if !lang.HasType(class, lang.NewKeyword("key")) {
		t.Fatal("keyword is not a clojure.lang.Named")
	}
	if !lang.HasType(class, lang.NewSymbol("symbol")) {
		t.Fatal("symbol is not a clojure.lang.Named")
	}
	if got := len(class.(*lang.Class).Types()); got != 2 {
		t.Fatalf("Named represented types = %d, want 2", got)
	}
}

func TestVarThreadBindingStaticsAreRegistered(t *testing.T) {
	for _, name := range []string{
		"Var.pushThreadBindings", "Var.popThreadBindings",
		"clojure.lang.Var.pushThreadBindings", "clojure.lang.Var.popThreadBindings",
	} {
		if _, ok := pkgmap.Get(name); !ok {
			t.Fatalf("static method %s is not registered", name)
		}
	}
	class, _ := pkgmap.HostClass("Var")
	if _, ok := lang.FieldOrMethod(class, "pushThreadBindings"); !ok {
		t.Fatal("Var class does not resolve pushThreadBindings")
	}
	if _, ok := lang.FieldOrMethod(class, "PushThreadBindings"); !ok {
		t.Fatal("Var class does not resolve rewritten PushThreadBindings")
	}
}
