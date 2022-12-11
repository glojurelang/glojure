package value

import (
	"sync/atomic"
)

type (
	Var struct {
		ns   *Namespace
		sym  *Symbol
		root atomic.Value

		meta atomic.Value
	}

	UnboundVar struct {
		v *Var
	}

	// varBox is a wrapper around a value stored in a var. Because
	// atomic.Value requires that all values loaded and stored must be
	// of the same concrete type, we need to wrap the value in a struct.
	varBox struct {
		val interface{}
	}
)

var (
	KeywordMacro = NewKeyword("macro")
)

func NewVar(ns *Namespace, sym *Symbol) *Var {
	v := &Var{
		ns:  ns,
		sym: sym,
	}
	v.root.Store(varBox{val: UnboundVar{v: v}})
	v.meta.Store(IPersistentMap(NewMap(nil)))
	return v
}

func NewVarWithRoot(ns *Namespace, sym *Symbol, root interface{}) *Var {
	v := NewVar(ns, sym)
	v.BindRoot(root)
	return v
}

func (v *Var) Namespace() *Namespace {
	return v.ns
}

func (v *Var) Symbol() *Symbol {
	return v.sym
}

func (v *Var) String() string {
	return "#'" + v.ns.Name().String() + "/" + v.sym.Name()
}

func (v *Var) HasRoot() bool {
	box := v.root.Load().(varBox)
	_, ok := box.val.(UnboundVar)
	return !ok
}

func (v *Var) BindRoot(root interface{}) {
	// TODO: handle metadata correctly
	v.root.Store(varBox{val: root})
}

func (v *Var) Get() interface{} {
	// TODO: figure out goroutine-local bindings
	box := v.root.Load().(varBox)
	return box.val
}

func (v *Var) Meta() IPersistentMap {
	return v.meta.Load().(IPersistentMap)
}

func (v *Var) SetMeta(meta IPersistentMap) {
	v.meta.Store(meta)
}

func (v *Var) IsMacro() bool {
	meta := v.Meta()
	b, ok := meta.EntryAt(KeywordMacro)
	if !ok {
		return false
	}
	return b.(bool)
}

func (v *Var) SetMacro() {
	v.SetMeta(v.Meta().Assoc(KeywordMacro, true).(IPersistentMap))
}
