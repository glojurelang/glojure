package value

import (
	"fmt"
	"sync/atomic"
)

var (
	RT = &RTMethods{}
)

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
	// TODO: implement
	fmt.Println("load", scriptBase)
}

func (rt *RTMethods) FindVar(qualifiedSym *Symbol) *Var {
	if qualifiedSym.Namespace() == "" {
		panic(fmt.Errorf("qualified symbol required: %v", qualifiedSym))
	}
	ns := GlobalEnv.FindNamespace(NewSymbol(qualifiedSym.Namespace()))
	if ns == nil {
		panic(fmt.Errorf("namespace not found: %v", qualifiedSym.Namespace()))
	}

	return ns.FindInternedVar(NewSymbol(qualifiedSym.Name()))
}
