package runtime

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
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

// loadPathEntry represents a single entry in the load path
type loadPathEntry struct {
	fs   fs.FS
	name string
}

var (
	RT = &RTMethods{}

	loadPath     = []loadPathEntry{{fs: stdlib.StdLib, name: "<StdLib>"}}
	loadPathLock sync.Mutex
)

// AddLoadPath adds a filesystem with a name to the end of the load path.
func AddLoadPath(fs fs.FS, name string) {
	loadPathLock.Lock()
	defer loadPathLock.Unlock()

	loadPath = append(loadPath, loadPathEntry{fs: fs, name: name})
}

// AddLoadPathString adds a directory path to the end of the load path.
func AddLoadPathString(path string) {
	AddLoadPath(os.DirFS(path), path)
}

// PrependLoadPathString adds a directory path to the front of the load path.
func PrependLoadPathString(path string) {
	loadPathLock.Lock()
	defer loadPathLock.Unlock()

	loadPath = append([]loadPathEntry{{fs: os.DirFS(path), name: path}}, loadPath...)
}

// GetLoadPath returns a copy of the current load path as a slice of strings.
// Each string represents a filesystem path that can be used by Glojure.
func GetLoadPath() []string {
	loadPathLock.Lock()
	defer loadPathLock.Unlock()

	result := make([]string, len(loadPath))
	for i, entry := range loadPath {
		if entry.name != "" {
			result[i] = entry.name
		} else {
			// Fallback to the old logic for unnamed filesystems
			switch v := entry.fs.(type) {
			case embed.FS:
				result[i] = "<StdLib>"
			default:
				if name := getFSName(v); name != "" {
					result[i] = name
				} else {
					result[i] = fmt.Sprintf("<%v>", v)
				}
			}
		}
	}
	return result
}

// getFSName attempts to extract a meaningful name from a filesystem.
// This is a helper function for GetLoadPath.
func getFSName(fsys fs.FS) string {
	// Try to read the root directory to see if it's a named filesystem
	if entries, err := fs.ReadDir(fsys, "."); err == nil && len(entries) > 0 {
		// If it's a subdirectory filesystem, try to get the parent name
		if subFS, ok := fsys.(interface{ Name() string }); ok {
			return subFS.Name()
		}
		// For embedded filesystems, we might be able to infer the name
		// from the directory structure
		if len(entries) == 1 && entries[0].IsDir() {
			return entries[0].Name()
		}
	}
	return ""
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

func (rt *RTMethods) Nth(x interface{}, i int) interface{} {
	return MustNth(x, i)
}

func (rt *RTMethods) NthDefault(x interface{}, i int, def interface{}) interface{} {
	v, ok := Nth(x, i)
	if !ok {
		return def
	}
	return v
}

func (rt *RTMethods) Peek(x interface{}) interface{} {
	if IsNil(x) {
		return nil
	}
	stk := x.(IPersistentStack)
	return stk.Peek()
}

func (rt *RTMethods) Pop(x interface{}) interface{} {
	if IsNil(x) {
		return nil
	}
	stk := x.(IPersistentStack)
	return stk.Pop()
}

func (rt *RTMethods) IntCast(x interface{}) int {
	return lang.IntCast(x)
}

func (rt *RTMethods) BooleanCast(x interface{}) bool {
	return lang.BooleanCast(x)
}

func (rt *RTMethods) ByteCast(x interface{}) byte {
	return lang.ByteCast(x)
}

func (rt *RTMethods) CharCast(x interface{}) Char {
	return lang.CharCast(x)
}

func (rt *RTMethods) UncheckedCharCast(x interface{}) Char {
	return lang.UncheckedCharCast(x)
}

func (rt *RTMethods) Dissoc(x interface{}, k interface{}) interface{} {
	return Dissoc(x, k)
}

func (rt *RTMethods) Contains(coll, key interface{}) bool {
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

func (rt *RTMethods) Find(coll, key interface{}) interface{} {
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
	kvs := make([]interface{}, 0, 3)
	for _, vr := range []*Var{VarCurrentNS, VarWarnOnReflection, VarUncheckedMath, lang.VarDataReaders} {
		kvs = append(kvs, vr, vr.Deref())
	}
	PushThreadBindings(NewMap(kvs...))
	defer PopThreadBindings()

	filename := scriptBase + ".glj"

	var buf []byte
	var err error

	loadPathLock.Lock()
	lp := loadPath
	loadPathLock.Unlock()
	for _, entry := range lp {
		buf, err = readFile(entry.fs, filename)
		if err == nil {
			break
		}
	}
	if err != nil {
		panic(err)
	}
	ReadEval(string(buf), WithFilename(filename))
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

func (rt *RTMethods) Alength(x interface{}) int {
	xVal := reflect.ValueOf(x)
	if xVal.Kind() == reflect.Slice || xVal.Kind() == reflect.Array {
		return xVal.Len()
	}
	panic(fmt.Errorf("Alength not supported on type: %T", x))
}

func (rt *RTMethods) ToArray(coll interface{}) interface{} {
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

func RTReadString(s string) interface{} {
	rdr := reader.New(strings.NewReader(s), reader.WithGetCurrentNS(func() *lang.Namespace {
		return lang.VarCurrentNS.Deref().(*lang.Namespace)
	}))
	v, err := rdr.ReadOne()
	if err != nil {
		panic(err)
	}
	return v
}
