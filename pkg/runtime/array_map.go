package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// nativeArrayMap retains array-map's ordered construction and odd-arity
// error while passing apply's sequence directly to the map constructor.
type nativeArrayMap struct{}

func (nativeArrayMap) IsFnValue() {}
func (nativeArrayMap) Invoke(args ...any) any {
	if len(args)%2 != 0 {
		panic(lang.NewIllegalArgumentError("No value supplied for key: " + nativeStrValue(args[len(args)-1])))
	}
	if len(args) == 0 {
		return lang.NewMap()
	}
	return lang.NewPersistentArrayMapAsIfByAssoc(args)
}
func (f nativeArrayMap) ApplyTo(args lang.ISeq) any { return f.Invoke(lang.ToSlice(args)...) }
func (f nativeArrayMap) Invoke0() any               { return lang.NewMap() }
func (f nativeArrayMap) Invoke2(a, b any) any       { return f.Invoke(a, b) }
func (f nativeArrayMap) Invoke4(a, b, c, d any) any { return f.Invoke(a, b, c, d) }
