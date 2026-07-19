package runtime

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
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

	useAot = func() bool {
		// default to true
		gua := strings.ToLower(os.Getenv("GLOJURE_USE_AOT"))
		return !(gua == "0" || gua == "false" || gua == "no" || gua == "off")
	}()

	warnNotAot = os.Getenv("GLOJURE_WARN_NOT_AOT") == "1"
)

func init() {
	stdlibPath := os.Getenv("GLOJURE_STDLIB_PATH")
	if stdlibPath != "" {
		AddLoadPath(os.DirFS(stdlibPath))
	} else {
		AddLoadPath(stdlib.StdLib)
	}
}

func GetUseAOT() bool {
	return useAot
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
	id                  atomic.Int32
	resolvedMethodsOnce sync.Once
	resolvedMethods     rtResolvedMethods
}

type rtResolvedMethods struct {
	nth        interface{}
	nthDefault interface{}
	get        interface{}
	contains   interface{}
	dissoc     interface{}
	find       interface{}
}

// ResolveFieldOrMethod keeps the collection accessors used by rewritten
// clojure.core forms on direct Go calls instead of reflected bound methods.
func (rt *RTMethods) ResolveFieldOrMethod(name string) (interface{}, bool) {
	rt.resolvedMethodsOnce.Do(func() {
		rt.resolvedMethods.nth = lang.FnFunc2(func(x, i any) any {
			return rt.Nth(x, lang.MustAsInt(i))
		})
		rt.resolvedMethods.nthDefault = lang.FnFunc3(func(x, i, def any) any {
			return rt.NthDefault(x, lang.MustAsInt(i), def)
		})
		getVariadic := lang.FnFunc(func(args ...any) any {
			if len(args) < 2 {
				panic(lang.NewIllegalArgumentError(fmt.Sprintf(
					"wrong number of arguments: expected at least 2, got %d",
					len(args),
				)))
			}
			return rt.Get(args[0], args[1], args[2:]...)
		})
		rt.resolvedMethods.get = getVariadic
		rt.resolvedMethods.contains = lang.FnFunc2(func(coll, key any) any {
			return rt.Contains(coll, key)
		})
		rt.resolvedMethods.dissoc = lang.FnFunc2(func(coll, key any) any {
			return rt.Dissoc(coll, key)
		})
		rt.resolvedMethods.find = lang.FnFunc2(func(coll, key any) any {
			return rt.Find(coll, key)
		})
	})

	switch name {
	case "nth", "Nth":
		return rt.resolvedMethods.nth, true
	case "nthdefault", "nthDefault", "NthDefault":
		return rt.resolvedMethods.nthDefault, true
	case "get", "Get":
		return rt.resolvedMethods.get, true
	case "contains", "Contains":
		return rt.resolvedMethods.contains, true
	case "dissoc", "Dissoc":
		return rt.resolvedMethods.dissoc, true
	case "find", "Find":
		return rt.resolvedMethods.find, true
	default:
		return nil, false
	}
}

func (rt *RTMethods) NextID() int {
	return int(rt.id.Add(1))
}

func (rt *RTMethods) Nth(x any, i int) any {
	if lang.IsNil(x) {
		return nil
	}
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

func (rt *RTMethods) ByteCast(x any) int8 {
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
	case string:
		n := lang.MustAsInt(key)
		return n >= 0 && n < lang.Count(coll)
	}
	panic(fmt.Errorf("contains? not supported on type: %T", coll))
}

// Get provides the clojure.lang.RT-compatible entry point referenced by the
// inherited inline definition of clojure.core/get.
func (rt *RTMethods) Get(coll, key any, notFound ...any) any {
	if len(notFound) == 0 {
		return Get(coll, key)
	}
	return GetDefault(coll, key, notFound[0])
}

func (rt *RTMethods) Subs(s string, start int) string {
	runes := []rune(s)
	if start < 0 || start > len(runes) {
		panic(lang.NewIllegalArgumentError("String index out of range"))
	}
	return string(runes[start:])
}

func (rt *RTMethods) SubsEnd(s string, start, end int) string {
	runes := []rune(s)
	if start < 0 || start > len(runes) || end < start || end > len(runes) {
		panic(lang.NewIllegalArgumentError("String index out of range"))
	}
	return string(runes[start:end])
}

func (rt *RTMethods) Subvec(v IPersistentVector, start, end any) IPersistentVector {
	return Subvec(v, subvecIndex(start), subvecIndex(end))
}

// subvecIndex converts a value to an int index for subvec, treating
// NaN as 0 to match Clojure's behavior.
func subvecIndex(x any) int {
	if f, ok := x.(float64); ok && math.IsNaN(f) {
		return 0
	}
	return IntCast(x)
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

	resourceBase := strings.ReplaceAll(strings.TrimPrefix(scriptBase, "/"), ".", "/")

	if useAot {
		// check nsloaders
		if loader := GetNSLoader(resourceBase); loader != nil {
			loader()
			return
		}
	}

	var buf []byte
	var foundFS fs.FS
	var filename string

	loadPathLock.Lock()
	lp := loadPath
	loadPathLock.Unlock()
	for _, fs := range lp {
		for _, testFilename := range []string{
			resourceBase + ".glj",
			resourceBase + ".clj",
			resourceBase + ".cljc",
		} {
			b, err := readFile(fs, testFilename)
			if err == nil {
				buf = b
				foundFS = fs
				filename = testFilename
				break
			}
		}
	}
	if filename == "" {
		panic(fmt.Errorf("failed to load %s: not found in load path", scriptBase))
	}
	if warnNotAot {
		fmt.Fprintf(os.Stderr, "WARNING: loading %s (not AOT)\n", filename)
	}
	ReadEval(string(buf), WithFilename(filename))

	// if compileFiles is set, compile the namespace to a .go file
	compileFiles := VarCompileFiles.Get().(bool)
	if !compileFiles {
		return
	}

	compileNSToFile(foundFS, scriptBase)
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

func (rt *RTMethods) Aclone(x any) any {
	xVal := reflect.ValueOf(x)
	switch xVal.Kind() {
	case reflect.Slice:
		cloned := reflect.MakeSlice(xVal.Type(), xVal.Len(), xVal.Cap())
		reflect.Copy(cloned, xVal)
		return cloned.Interface()
	case reflect.Array:
		cloned := reflect.New(xVal.Type()).Elem()
		reflect.Copy(cloned, xVal)
		return cloned.Interface()
	}
	panic(fmt.Errorf("Aclone not supported on type: %T", x))
}

func (rt *RTMethods) ToArray(coll any) any {
	return lang.ToSlice(coll)
}

// ObjectArray creates an array of objects ([]any)
func (rt *RTMethods) ObjectArray(sizeOrSeq any) any {
	if lang.IsNumber(sizeOrSeq) {
		return make([]any, lang.MustAsInt(sizeOrSeq))
	}
	s := lang.Seq(sizeOrSeq)
	size := lang.Count(s)
	ret := make([]any, size)
	for i := 0; i < len(ret) && s != nil; i, s = i+1, s.Next() {
		ret[i] = s.First()
	}
	return ret
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
