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
	KeywordMacro   = NewKeyword("macro")
	KeywordPrivate = NewKeyword("private")

	_ IRef = (*Var)(nil)
)

func NewVar(ns *Namespace, sym *Symbol) *Var {
	v := &Var{
		ns:  ns,
		sym: sym,
	}
	v.root.Store(varBox{val: UnboundVar{v: v}})
	v.meta.Store(IPersistentMap(NewMap()))
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

func (v *Var) Set(val interface{}) interface{} {
	// TODO: validate
	// TODO: thread-local bindings
	v.BindRoot(val)
	return val
}

func (v *Var) Meta() IPersistentMap {
	return v.meta.Load().(IPersistentMap)
}

func (v *Var) SetMeta(meta IPersistentMap) {
	v.meta.Store(meta)
}

func (v *Var) AlterMeta(alter IFn, args ISeq) IPersistentMap {
	meta := alter.ApplyTo(NewCons(v.Meta(), args)).(IPersistentMap)
	v.SetMeta(meta)
	return meta
}

func (v *Var) IsMacro() bool {
	meta := v.Meta()
	isMacro := meta.EntryAt(KeywordMacro)
	if isMacro == nil {
		return false
	}
	return isMacro.Val() == true
}

func (v *Var) SetMacro() {
	v.SetMeta(v.Meta().Assoc(KeywordMacro, true).(IPersistentMap))
}

func (v *Var) IsPublic() bool {
	meta := v.Meta()
	isPrivate := meta.EntryAt(KeywordPrivate)
	if isPrivate == nil {
		return true
	}
	return !booleanCast(isPrivate.Val())
}

func booleanCast(x interface{}) bool {
	if xb, ok := x.(bool); ok {
		return xb
	}
	return x != nil
}

func (v *Var) Deref() interface{} {
	return v.Get()
}

// SetValidator(vf Applyer)
// Validator() Applyer
// Watches() IPersistentMap
// AddWatch(key interface{}, fn Applyer)
// RemoveWatch(key interface{})

// implementations of the above methods that panic with "not implemented"

func (v *Var) SetValidator(vf Applyer) {
	panic("not implemented")
}

func (v *Var) Validator() Applyer {
	panic("not implemented")
}

func (v *Var) Watches() IPersistentMap {
	panic("not implemented")
}

func (v *Var) AddWatch(key interface{}, fn Applyer) {
	panic("not implemented")
}

func (v *Var) RemoveWatch(key interface{}) {
	panic("not implemented")
}
