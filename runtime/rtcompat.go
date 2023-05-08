package runtime

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/glojurelang/glojure/stdlib"

	. "github.com/glojurelang/glojure/value"
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

func (rt *RTMethods) IntCast(x interface{}) int {
	if c, ok := x.(Char); ok {
		return int(c)
	}
	return int(AsInt64(x))
}

func (rt *RTMethods) BooleanCast(x interface{}) bool {
	if b, ok := x.(bool); ok {
		return b
	}
	return !IsNil(x)
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
	for _, vr := range []*Var{VarCurrentNS, VarWarnOnReflection, VarUncheckedMath} {
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
