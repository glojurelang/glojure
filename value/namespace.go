package value

import (
	"fmt"
	"sync/atomic"
)

type Namespace struct {
	name *Symbol

	// atomic references to maps
	mappings atomic.Value
	aliases  atomic.Value
}

var (
	SymbolCoreNamespace = NewSymbol("glojure.core")

	FindNamespace = ApplyerFunc(func(env Environment, args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("find-ns expects 1 argument, got %d", len(args))
		}
		sym, ok := args[0].(*Symbol)
		if !ok {
			return nil, fmt.Errorf("find-ns expects a symbol, got %T", args[0])
		}
		return env.FindNamespace(sym), nil
	})
)

func NewNamespace(name *Symbol) *Namespace {
	ns := &Namespace{
		name: name,
	}

	ns.mappings.Store(NewMap(nil))
	ns.aliases.Store(NewMap(nil))

	// TODO: add default mappings (see RT.java in clojure)

	return ns
}

func (ns *Namespace) Name() *Symbol {
	return ns.name
}

func (ns *Namespace) Mappings() *Map {
	return ns.mappings.Load().(*Map)
}

func (ns *Namespace) Aliases() *Map {
	return ns.aliases.Load().(*Map)
}

func (ns *Namespace) isInternedMapping(sym *Symbol, v interface{}) bool {
	vr, ok := v.(*Var)
	return ok && vr.Namespace() == ns && Equal(vr.Symbol(), sym)
}

// Intern creates a new Var in this namespace with the given name.
func (ns *Namespace) Intern(env Environment, sym *Symbol) *Var {
	if sym.Namespace() != "" {
		panic(fmt.Errorf("can't intern qualified name: %s", sym))
	}
	m := ns.Mappings()

	var v *Var
	var o interface{}
	for {
		var ok bool
		o, ok = m.ValueAt(sym)
		if ok {
			break
		}

		if v == nil {
			v = NewVar(ns, sym)
		}
		newMap := m.Assoc(sym, v)
		ns.mappings.CompareAndSwap(m, newMap)
		m = ns.Mappings()
	}
	if ns.isInternedMapping(sym, o) {
		return o.(*Var)
	}
	if v == nil {
		v = NewVar(ns, sym)
	}
	if ns.checkReplacement(env, sym, o, v) {
		for !ns.mappings.CompareAndSwap(m, m.Assoc(sym, v)) {
			m = ns.Mappings()
		}
		return v
	}

	return o.(*Var)
}

func (ns *Namespace) checkReplacement(env Environment, sym *Symbol, old, neu interface{}) bool {
	/*
		 This method checks if a namespace's mapping is applicable and warns on problematic cases.
		 It will return a boolean indicating if a mapping is replaceable.
		 The semantics of what constitutes a legal replacement mapping is summarized as follows:

		| classification | in namespace ns        | newval = anything other than ns/name | newval = ns/name                    |
		|----------------+------------------------+--------------------------------------+-------------------------------------|
		| native mapping | name -> ns/name        | no replace, warn-if newval not-core  | no replace, warn-if newval not-core |
		| alias mapping  | name -> other/whatever | warn + replace                       | warn + replace                      |
	*/

	errOut := env.Stderr()

	if _, ok := old.(*Var); ok {
		var nns *Namespace
		if neuVar, ok := neu.(*Var); ok {
			nns = neuVar.Namespace()
		}
		if ns.isInternedMapping(sym, old) {
			if nns != env.FindNamespace(SymbolCoreNamespace) {
				fmt.Fprintf(errOut, "REJECTED: attempt to replace interned var %s with %s in %s, you must ns-unmap first\n", old, neu, ns.name)
			}
			return false
		}
	}

	fmt.Fprintf(errOut, "WARNING: %s already refers to %s in namespace: %s, being replaced by: %s\n", sym, old, ns.name, neu)
	return true
}

func (ns *Namespace) InternWithValue(env Environment, sym *Symbol, value interface{}, replaceRoot bool) *Var {
	v := ns.Intern(env, sym)
	if !v.HasRoot() || replaceRoot {
		v.BindRoot(value)
	}
	return v
}

func (ns *Namespace) GetMapping(sym *Symbol) interface{} {
	m := ns.Mappings()
	v, ok := m.ValueAt(sym)
	if !ok {
		return nil
	}
	return v
}

func (ns *Namespace) FindInternedVar(sym *Symbol) *Var {
	m := ns.Mappings()
	v, ok := m.ValueAt(sym)
	if !ok {
		return nil
	}
	vr, ok := v.(*Var)
	if !ok {
		return nil
	}
	if vr.Namespace() != ns {
		return nil
	}
	return vr
}

func (ns *Namespace) LookupAlias(sym *Symbol) *Namespace {
	m := ns.Aliases()
	v, ok := m.ValueAt(sym)
	if !ok {
		return nil
	}
	return v.(*Namespace)
}

// Refer adds a reference to an existing Var, possibly in another
// namespace, to this namespace.
func (ns *Namespace) Refer(sym *Symbol, v *Var) *Var {
	return ns.reference(sym, v).(*Var)
}

func (ns *Namespace) reference(sym *Symbol, v interface{}) interface{} {
	if sym.Namespace() != "" {
		panic(fmt.Errorf("can't intern qualified name: %s", sym))
	}

	m := ns.Mappings()
	var o interface{}
	for {
		var ok bool
		o, ok = m.ValueAt(sym)
		if ok {
			break
		}
		newMap := m.Assoc(sym, v)
		ns.mappings.CompareAndSwap(m, newMap)
		m = ns.Mappings()
	}
	if ns.isInternedMapping(sym, o) {
		return o.(*Var)
	}
	if o == v {
		return o
	}

	// TODO: this will panic :). need to plumb through env, live with a
	// global env, or make the check less strict if env is nil.
	if ns.checkReplacement(nil, sym, o, v) {
		for !ns.mappings.CompareAndSwap(m, m.Assoc(sym, v)) {
			m = ns.Mappings()
		}
		return v
	}

	return o
}
