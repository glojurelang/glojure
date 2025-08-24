package runtime

import (
	"fmt"
	"io"
	"io/fs"
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
	RT = &RTMethods{}

	loadPath     = []fs.FS{stdlib.StdLib}
	loadPathLock sync.Mutex
)

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

	filename := scriptBase + ".glj"

	var buf []byte
	var err error

	loadPathLock.Lock()
	lp := loadPath
	loadPathLock.Unlock()
	for _, fs := range lp {
		buf, err = readFile(fs, filename)
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
