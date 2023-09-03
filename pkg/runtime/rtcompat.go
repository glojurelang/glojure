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
	value "github.com/glojurelang/glojure/pkg/lang"
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

func (rt *RTMethods) Pop(x interface{}) interface{} {
	if IsNil(x) {
		return nil
	}
	stk := x.(IPersistentStack)
	return stk.Pop()
}

func (rt *RTMethods) IntCast(x interface{}) int {
	return value.IntCast(x)
}

func (rt *RTMethods) BooleanCast(x interface{}) bool {
	return value.BooleanCast(x)
}

func (rt *RTMethods) ByteCast(x interface{}) byte {
	return value.ByteCast(x)
}

func (rt *RTMethods) CharCast(x interface{}) Char {
	return value.CharCast(x)
}

func (rt *RTMethods) UncheckedCharCast(x interface{}) Char {
	return value.UncheckedCharCast(x)
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
	for _, vr := range []*Var{VarCurrentNS, VarWarnOnReflection, VarUncheckedMath, value.VarDataReaders} {
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

func (rt *RTMethods) Alength(x interface{}) int {
	xVal := reflect.ValueOf(x)
	if xVal.Kind() == reflect.Slice {
		return xVal.Len()
	}
	panic(fmt.Errorf("Alength not supported on type: %T", x))
}

func (rt *RTMethods) ToArray(coll interface{}) interface{} {
	if lang.IsNil(coll) {
		return nil
	}
	switch coll := coll.(type) {
	case []interface{}:
		return coll
	case lang.ISeq, lang.IPersistentCollection:
		res := make([]interface{}, 0, lang.Count(coll))
		for s := lang.Seq(coll); s != nil; s = lang.Next(s) {
			res = append(res, lang.First(s))
		}
		return res
	}
	if v := reflect.ValueOf(coll); v.Kind() == reflect.Slice {
		return coll
	}
	panic(fmt.Errorf("ToArray not supported on type: %T", coll))
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
	rdr := reader.New(strings.NewReader(s), reader.WithGetCurrentNS(func() *value.Namespace {
		return value.VarCurrentNS.Deref().(*value.Namespace)
	}))
	v, err := rdr.ReadOne()
	if err != nil {
		panic(err)
	}
	return v
}
