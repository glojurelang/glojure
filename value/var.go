package value

import (
	"sync"
	"sync/atomic"

	"github.com/jtolio/gls"
)

type (
	Var struct {
		ns   *Namespace
		sym  *Symbol
		root atomic.Value

		meta atomic.Value

		dynamicBound atomic.Bool
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

	varBindings map[*Var]*varBox
	glStorage   struct {
		bindings []varBindings
	}
)

var (
	KeywordMacro   = NewKeyword("macro")
	KeywordPrivate = NewKeyword("private")
	KeywordDynamic = NewKeyword("dynamic")
	KeywordNS      = NewKeyword("ns")

	// TODO: use an atomic and CAS
	glsBindings    = make(map[uint]*glStorage)
	glsBindingsMtx sync.RWMutex

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

func (v *Var) getRoot() interface{} {
	return v.root.Load().(varBox).val
}

func (v *Var) Get() interface{} {
	if !v.dynamicBound.Load() {
		return v.getRoot()
	}
	return v.Deref()
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
	// TODO: ResetMeta
	meta = Assoc(meta, KeywordNS, v.ns).(IPersistentMap)
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

func (v *Var) isDynamic() bool {
	meta := v.Meta()
	isDynamic := meta.EntryAt(KeywordDynamic)
	if isDynamic == nil {
		return false
	}
	return booleanCast(isDynamic.Val())
}

func (v *Var) Deref() interface{} {
	if b := v.getDynamicBinding(); b != nil {
		return b.val
	}
	return v.getRoot()
}

func (v *Var) getDynamicBinding() *varBox {
	if !v.dynamicBound.Load() {
		return nil
	}
	var storage *glStorage
	var ok bool
	gid := goroutineID()
	glsBindingsMtx.RLock()
	storage, ok = glsBindings[gid]
	glsBindingsMtx.RUnlock()
	if !ok {
		return nil
	}
	return storage.get(v)
}

func (v *Var) SetValidator(vf IFn) {
	panic("not implemented")
}

func (v *Var) Validator() IFn {
	panic("not implemented")
}

func (v *Var) Watches() IPersistentMap {
	panic("not implemented")
}

func (v *Var) AddWatch(key interface{}, fn IFn) {
	panic("not implemented")
}

func (v *Var) RemoveWatch(key interface{}) {
	panic("not implemented")
}

////////////////////////////////////////////////////////////////////////////////
// Dynamic binding

func (s *glStorage) get(v *Var) *varBox {
	for i := len(s.bindings) - 1; i >= 0; i-- {
		if b, ok := s.bindings[i][v]; ok {
			return b
		}
	}
	return nil
}

func goroutineID() uint {
	gid, ok := gls.GetGoroutineId()
	if !ok {
		panic("no goroutine id")
	}
	return gid
}

func PushThreadBindings(bindings IPersistentMap) {
	gid := goroutineID()
	glsBindingsMtx.RLock()
	storage, ok := glsBindings[gid]
	glsBindingsMtx.RUnlock()
	if !ok {
		glsBindingsMtx.Lock()
		storage = &glStorage{}
		glsBindings[gid] = storage
		glsBindingsMtx.Unlock()
	}

	store := make(varBindings)
	storage.bindings = append(storage.bindings, store)

	for seq := Seq(bindings); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		vr := entry.Key().(*Var)
		val := entry.Val()

		if !vr.isDynamic() {
			//panic("cannot dynamically bind non-dynamic var: " + vr.String())
		}
		// TODO: validate
		vr.dynamicBound.Store(true)
		store[vr] = &varBox{val: val}
	}
}

func PopThreadBindings() {
	gid := goroutineID()
	glsBindingsMtx.RLock()
	storage := glsBindings[gid]
	glsBindingsMtx.RUnlock()

	if len(storage.bindings) > 1 {
		storage.bindings = storage.bindings[:len(storage.bindings)-1]
		return
	}

	glsBindingsMtx.Lock()
	delete(glsBindings, gid)
	glsBindingsMtx.Unlock()
}

func booleanCast(x interface{}) bool {
	if xb, ok := x.(bool); ok {
		return xb
	}
	return x != nil
}
