package runtime

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/reader"
	"github.com/glojurelang/glojure/pkg/stdlib"

	. "github.com/glojurelang/glojure/pkg/lang"
)

var (
	// TODO: don't use a global RT instance; incurs overhead from dynamic lookup
	// of methods. Instead, generate direct calls to functions in this package.
	RT = &RTMethods{}

	loadPath     = []fs.FS{}
	loadPathLock sync.Mutex

	useAot = os.Getenv("GLOJURE_USE_AOT") == "1"
)

func init() {
	stdlibPath := os.Getenv("GLOJURE_STDLIB_PATH")
	if stdlibPath != "" {
		AddLoadPath(os.DirFS(stdlibPath))
	} else {
		AddLoadPath(stdlib.StdLib)
	}
}

// AddLoadPath adds a filesystem to the load path.
func AddLoadPath(fs fs.FS) {
	loadPathLock.Lock()
	defer loadPathLock.Unlock()

	loadPath = append(loadPath, fs)
}

// RT is a struct with methods that map to Clojure's RT class' static
// methods. This approach is used to make translation of core.clj to
// Glojure easier.
type RTMethods struct {
	id atomic.Int32
}

func (rt *RTMethods) NextID() int {
	return int(rt.id.Add(1))
}

func (rt *RTMethods) Nth(x any, i int) any {
	return MustNth(x, i)
}

func (rt *RTMethods) NthDefault(x any, i int, def any) any {
	v, ok := Nth(x, i)
	if !ok {
		return def
	}
	return v
}

func (rt *RTMethods) Peek(x any) any {
	if IsNil(x) {
		return nil
	}
	stk := x.(IPersistentStack)
	return stk.Peek()
}

func (rt *RTMethods) Pop(x any) any {
	if IsNil(x) {
		return nil
	}
	stk := x.(IPersistentStack)
	return stk.Pop()
}

func (rt *RTMethods) IntCast(x any) int {
	return lang.IntCast(x)
}

func (rt *RTMethods) BooleanCast(x any) bool {
	return lang.BooleanCast(x)
}

func (rt *RTMethods) ByteCast(x any) byte {
	return lang.ByteCast(x)
}

func (rt *RTMethods) CharCast(x any) Char {
	return lang.CharCast(x)
}

func (rt *RTMethods) UncheckedCharCast(x any) Char {
	return lang.UncheckedCharCast(x)
}

func (rt *RTMethods) Dissoc(x any, k any) any {
	return Dissoc(x, k)
}

func (rt *RTMethods) Contains(coll, key any) bool {
	switch coll := coll.(type) {
	case nil:
		return false
	case Associative:
		return coll.ContainsKey(key)
	case IPersistentSet:
		return coll.Contains(key)
		// TODO: other types
	}
	panic(fmt.Errorf("contains? not supported on type: %T", coll))
}

func (rt *RTMethods) Subvec(v IPersistentVector, start, end int) IPersistentVector {
	return Subvec(v, start, end)
}

func (rt *RTMethods) Find(coll, key any) any {
	switch coll := coll.(type) {
	case nil:
		return nil
	case Associative:
		return coll.EntryAt(key)
	default:
		panic(fmt.Errorf("find not supported on type: %T", coll))
	}
}

func (rt *RTMethods) Load(scriptBase string) {
	kvs := make([]any, 0, 3)
	for _, vr := range []*Var{VarCurrentNS, VarWarnOnReflection, VarUncheckedMath, lang.VarDataReaders} {
		kvs = append(kvs, vr, vr.Deref())
	}
	PushThreadBindings(NewMap(kvs...))
	defer PopThreadBindings()

	if useAot {
		// check nsloaders
		if loader := GetNSLoader(strings.TrimPrefix(scriptBase, "/")); loader != nil {
			loader()
			return
		}
	}

	filename := scriptBase + ".glj"

	var buf []byte
	var err error
	var foundFS fs.FS

	loadPathLock.Lock()
	lp := loadPath
	loadPathLock.Unlock()
	for _, fs := range lp {
		buf, err = readFile(fs, filename)
		if err == nil {
			foundFS = fs
			break
		}
	}
	if err != nil {
		panic(err)
	}
	ReadEval(string(buf), WithFilename(filename))

	compileFiles := VarCompileFiles.Get().(bool)
	if !compileFiles {
		return
	}

	compileNSToFile(foundFS, scriptBase)
}

// compileNSToFile compiles the given namespace to a Go source file,
// given a fs.FS and the script base name (without extension).
func compileNSToFile(fs fs.FS, scriptBase string) {
	// check if the found FS is writable
	// we use the fact that os.DirFS(".") is just a named string type under the hood
	if reflect.TypeOf(fs).Kind() != reflect.String {
		panic(fmt.Errorf("cannot compile %s: filesystem is not writable", scriptBase))
	}
	fsDir := fmt.Sprintf("%s", fs)

	// compile to .go files
	targetDir := filepath.Join(fsDir, scriptBase)
	targetFile := filepath.Join(targetDir, "loader.go")
	fmt.Printf("Compiling %s to %s\n", scriptBase, targetFile)

	// ensure directory exists
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	gen := NewGenerator(&buf)
	currNS := VarCurrentNS.Deref().(*lang.Namespace)
	if err = gen.Generate(currNS); err != nil {
		panic(fmt.Errorf("failed to generate code for namespace %s: %w", currNS.Name(), err))
	}
	err = os.WriteFile(targetFile, buf.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write generated code to %s: %w", targetFile, err))
	}
}

func readFile(fs fs.FS, filename string) ([]byte, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

func (rt *RTMethods) FindVar(qualifiedSym *Symbol) *Var {
	if qualifiedSym.Namespace() == "" {
		panic(fmt.Errorf("qualified symbol required: %v", qualifiedSym))
	}
	ns := FindNamespace(NewSymbol(qualifiedSym.Namespace()))
	if ns == nil {
		panic(fmt.Errorf("namespace not found: %v", qualifiedSym.Namespace()))
	}

	return ns.FindInternedVar(NewSymbol(qualifiedSym.Name()))
}

func (rt *RTMethods) Alength(x any) int {
	xVal := reflect.ValueOf(x)
	if xVal.Kind() == reflect.Slice || xVal.Kind() == reflect.Array {
		return xVal.Len()
	}
	panic(fmt.Errorf("Alength not supported on type: %T", x))
}

func (rt *RTMethods) ToArray(coll any) any {
	return lang.ToSlice(coll)
}

var (
	mungeCharMap = map[rune]string{
		'-':  "_",
		':':  "_COLON_",
		'+':  "_PLUS_",
		'>':  "_GT_",
		'<':  "_LT_",
		'=':  "_EQ_",
		'~':  "_TILDE_",
		'!':  "_BANG_",
		'@':  "_CIRCA_",
		'#':  "_SHARP_",
		'\'': "_SINGLEQUOTE_",
		'"':  "_DOUBLEQUOTE_",
		'%':  "_PERCENT_",
		'^':  "_CARET_",
		'&':  "_AMPERSAND_",
		'*':  "_STAR_",
		'|':  "_BAR_",
		'{':  "_LBRACE_",
		'}':  "_RBRACE_",
		'[':  "_LBRACK_",
		']':  "_RBRACK_",
		'/':  "_SLASH_",
		'\\': "_BSLASH_",
		'?':  "_QMARK_",
	}
)

func (rt *RTMethods) Munge(name string) string {
	sb := strings.Builder{}
	for _, c := range name {
		sub, ok := mungeCharMap[c]
		if ok {
			sb.WriteString(sub)
		} else {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

func RTReadString(s string) any {
	rdr := reader.New(strings.NewReader(s), reader.WithGetCurrentNS(func() *lang.Namespace {
		return lang.VarCurrentNS.Deref().(*lang.Namespace)
	}))
	v, err := rdr.ReadOne()
	if err != nil {
		panic(err)
	}
	return v
}
