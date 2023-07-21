package value

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/modern-go/gls"
)

type (
	Var struct {
		ns   *Namespace
		sym  *Symbol
		root atomic.Value

		meta atomic.Value

		// TODO: populate this from meta in the right places
		dynamic      bool
		dynamicBound atomic.Bool

		syncLock sync.Mutex
	}

	UnboundVar struct {
		v *Var
	}

	varBindings map[*Var]*Box
	glStorage   struct {
		bindings []varBindings
	}

	// TODO: public rev counter
)

func (uv *UnboundVar) String() string {
	return "Unbound: " + uv.v.String()
}

var (
	NSCore = FindOrCreateNamespace(SymbolCoreNamespace)

	VarNS   = InternVar(NSCore, NewSymbol("ns"), false, true)
	VarInNS = InternVar(NSCore, NewSymbol("in-ns"), false, true)

	VarCurrentNS        = InternVarReplaceRoot(NSCore, NewSymbol("*ns*"), NSCore).SetDynamic()
	VarWarnOnReflection = InternVarReplaceRoot(NSCore, NewSymbol("*warn-on-reflection*"), false).SetDynamic()
	VarUncheckedMath    = InternVarReplaceRoot(NSCore, NewSymbol("*unchecked-math*"), false).SetDynamic()
	VarAgent            = InternVarReplaceRoot(NSCore, NewSymbol("*agent*"), nil).SetDynamic()
	VarPrintReadably    = InternVarReplaceRoot(NSCore, NewSymbol("*print-readably*"), true).SetDynamic()
	VarOut              = InternVarReplaceRoot(NSCore, NewSymbol("*out*"), os.Stdout).SetDynamic()
	VarIn               = InternVarReplaceRoot(NSCore, NewSymbol("*in*"), os.Stdin).SetDynamic()
	VarAssert           = InternVarReplaceRoot(NSCore, NewSymbol("*assert*"), false).SetDynamic()
	VarCompileFiles     = InternVarReplaceRoot(NSCore, NewSymbol("*compile-files*"), false).SetDynamic()
	VarFile             = InternVarReplaceRoot(NSCore, NewSymbol("*file*"), "NO_SOURCE_FILE").SetDynamic()
	VarDataReaders      = InternVarReplaceRoot(NSCore, NewSymbol("*data-readers*"), emptyMap).SetDynamic()

	// TODO: use variant of InternVar that doesn't replace root.
	VarPrintInitialized = InternVarName(NSCore.Name(), NewSymbol("print-initialized"))
	VarPrOn             = InternVarName(NSCore.Name(), NewSymbol("pr-on"))
	VarParents          = InternVarName(NSCore.Name(), NewSymbol("parents"))

	// TODO: use an atomic and CAS
	glsBindings    = make(map[int64]*glStorage)
	glsBindingsMtx sync.RWMutex

	_ IRef = (*Var)(nil)
	_ IFn  = (*Var)(nil)
)

func InternVarReplaceRoot(ns *Namespace, sym *Symbol, root interface{}) *Var {
	return InternVar(ns, sym, root, true)
}

func InternVar(ns *Namespace, sym *Symbol, root interface{}, replaceRoot bool) *Var {
	dvout := ns.Intern(sym)
	if !dvout.HasRoot() || replaceRoot {
		dvout.BindRoot(root)
	}
	return dvout
}

func InternVarName(nsSym, nameSym *Symbol) *Var {
	ns := FindOrCreateNamespace(nsSym)
	return ns.Intern(nameSym)
}

func NewVar(ns *Namespace, sym *Symbol) *Var {
	v := &Var{
		ns:  ns,
		sym: sym,
	}
	v.root.Store(Box{val: &UnboundVar{v: v}})
	v.meta.Store(NewBox(emptyMap))
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
	box := v.root.Load().(Box)
	_, ok := box.val.(*UnboundVar)
	return !ok
}

func (v *Var) BindRoot(root interface{}) {
	// TODO: handle metadata correctly
	v.root.Store(Box{val: root})
}

func (v *Var) IsBound() bool {
	return v.HasRoot() || v.dynamicBound.Load() && v.getDynamicBinding() != nil
}

func (v *Var) getRoot() interface{} {
	return v.root.Load().(Box).val
}

func (v *Var) Get() interface{} {
	if !v.dynamicBound.Load() {
		return v.getRoot()
	}
	return v.Deref()
}

func (v *Var) Set(val interface{}) interface{} {
	// TODO: validate
	b := v.getDynamicBinding()
	if b == nil {
		fmt.Println("current gid", mustGoroutineID())
		for k, v := range glsBindings {
			fmt.Printf("glsBindings[%d] = %v\n", k, v)
		}
		panic(fmt.Sprintf("can't change/establish root binding of: %s", v))
	}
	b.val = val
	return val
}

func (v *Var) Meta() IPersistentMap {
	return v.meta.Load().(*Box).val.(IPersistentMap)
}

func (v *Var) SetMeta(meta IPersistentMap) {
	// TODO: ResetMeta
	meta = Assoc(meta, KWNS, v.ns).(IPersistentMap)
	v.meta.Store(NewBox(meta))
}

func (v *Var) AlterMeta(alter IFn, args ISeq) IPersistentMap {
	meta := alter.ApplyTo(NewCons(v.Meta(), args)).(IPersistentMap)
	v.SetMeta(meta)
	return meta
}

func (v *Var) IsMacro() bool {
	meta := v.Meta()
	isMacro := meta.EntryAt(KWMacro)
	if isMacro == nil {
		return false
	}
	return isMacro.Val() == true
}

func (v *Var) SetMacro() {
	v.SetMeta(v.Meta().Assoc(KWMacro, true).(IPersistentMap))
}

func (v *Var) IsPublic() bool {
	meta := v.Meta()
	isPrivate := meta.EntryAt(KWPrivate)
	if isPrivate == nil {
		return true
	}
	return !BooleanCast(isPrivate.Val())
}

func (v *Var) isDynamic() bool {
	return v.dynamic
}

func (v *Var) SetDynamic() *Var {
	v.dynamic = true
	return v
}

func (v *Var) Deref() interface{} {
	if b := v.getDynamicBinding(); b != nil {
		return b.val
	}
	return v.getRoot()
}

func (v *Var) getDynamicBinding() *Box {
	if !v.dynamicBound.Load() {
		return nil
	}
	var storage *glStorage
	gid := mustGoroutineID()

	glsBindingsMtx.RLock()
	storage, ok := glsBindings[gid]
	glsBindingsMtx.RUnlock()

	if !ok {
		return nil
	}
	return storage.get(v)
}

func (v *Var) AlterRoot(alter IFn, args ISeq) interface{} {
	v.syncLock.Lock()
	defer v.syncLock.Unlock()

	newRoot := alter.ApplyTo(NewCons(v.Get(), args))
	// TODO: validate, ++rev, notifyWatches
	// oldRoot := v.Get()
	v.Set(newRoot)
	return newRoot
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

func (v *Var) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(v)))
}

func (v *Var) fn() IFn {
	return v.Deref().(IFn)
}

func (v *Var) Invoke(args ...interface{}) interface{} {
	return v.fn().Invoke(args...)
}

func (v *Var) ApplyTo(args ISeq) interface{} {
	return v.fn().ApplyTo(args)
}

////////////////////////////////////////////////////////////////////////////////
// Dynamic binding

func (s *glStorage) get(v *Var) *Box {
	for i := len(s.bindings) - 1; i >= 0; i-- {
		if b, ok := s.bindings[i][v]; ok {
			return b
		}
	}
	return nil
}

func mustGoroutineID() int64 {
	return gls.GoID()
}

func PushThreadBindings(bindings IPersistentMap) {
	gid := mustGoroutineID()

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
			panic("cannot dynamically bind non-dynamic var: " + vr.String())
		}
		// TODO: validate
		vr.dynamicBound.Store(true)
		store[vr] = &Box{val: val}
	}
}

func PopThreadBindings() {
	gid := mustGoroutineID()
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
