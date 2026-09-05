package lang

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/glojurelang/glojure/internal/goid"
)

type (
	Var struct {
		ns   *Namespace
		sym  *Symbol
		root atomic.Pointer[varRoot]

		meta atomic.Value

		// TODO: populate this from meta in the right places
		dynamic         bool
		dynamicBindings atomic.Int64

		// isMacroCached: 0=unknown, 1=false, 2=true
		isMacroCached atomic.Int32

		watches IPersistentMap

		syncLock sync.Mutex
	}

	UnboundVar struct {
		v *Var
	}

	// VarRootVersion is an opaque identity for one root binding. Generated
	// optimized code uses it to verify that a Var still has the root it was
	// compiled against before taking a direct-call path.
	VarRootVersion struct {
		marker byte
	}

	varRoot struct {
		val     interface{}
		version *VarRootVersion
		// cell backs version for roots made by newVarRoot, so a
		// rebinding costs one allocation instead of two.
		cell VarRootVersion
	}

	lazyVarMeta struct {
		once sync.Once
		fn   func() IPersistentMap
		meta IPersistentMap
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

	VarLoadFile = InternVar(NSCore, NewSymbol("load-file"), nil, true)

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

	unboundVarResolver atomic.Value

	_ IRef = (*Var)(nil)
	_ IFn  = (*Var)(nil)
)

// SetUnboundVarResolver installs the runtime hook used to load an AOT
// namespace when one of its Vars is first dereferenced.
func SetUnboundVarResolver(resolver func(*Var)) {
	unboundVarResolver.Store(resolver)
}

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

// internVarNamespace caches the namespace of the last InternVarName
// call. Generated loaders intern every var of a namespace in a row, so
// the cache turns the namespace registry lookup into a pointer compare.
var internVarNamespace atomic.Pointer[Namespace]

func InternVarName(nsSym, nameSym *Symbol) *Var {
	ns := internVarNamespace.Load()
	if ns == nil || (ns.name != nsSym && ns.name.String() != nsSym.String()) {
		ns = FindOrCreateNamespace(nsSym)
		internVarNamespace.Store(ns)
	}
	return ns.Intern(nameSym)
}

// varBlock allocates a Var together with its initial root, unbound
// marker, root version and meta box in one object. A program interns
// thousands of vars at startup, so this saves four allocations each.
type varBlock struct {
	v       Var
	root    varRoot
	unbound UnboundVar
	meta    Box
}

func NewVar(ns *Namespace, sym *Symbol) *Var {
	b := &varBlock{}
	v := &b.v
	v.ns = ns
	v.sym = sym
	v.watches = emptyMap
	b.unbound.v = v
	b.root.val = &b.unbound
	b.root.version = &b.root.cell
	v.root.Store(&b.root)
	b.meta.val = emptyMap
	v.meta.Store(&b.meta)
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

func (v *Var) ToSymbol() *Symbol {
	return InternSymbol(v.ns.Name().String(), v.sym.Name())
}

func (v *Var) String() string {
	return "#'" + v.ns.Name().String() + "/" + v.sym.Name()
}

func (v *Var) HasRoot() bool {
	root := v.root.Load()
	_, ok := root.val.(*UnboundVar)
	return !ok
}

func (v *Var) BindRoot(root interface{}) {
	// TODO: handle metadata correctly
	old := v.root.Swap(newVarRoot(root))
	v.notifyWatches(old.val, root)
}

func newVarRoot(val interface{}) *varRoot {
	r := &varRoot{val: val}
	r.version = &r.cell
	return r
}

func (v *Var) IsBound() bool {
	return v.HasRoot() || v.dynamicBindings.Load() > 0 && v.getDynamicBinding() != nil
}

func (v *Var) getRoot() interface{} {
	root := v.root.Load().val
	if _, unbound := root.(*UnboundVar); !unbound {
		return root
	}
	if resolver, ok := unboundVarResolver.Load().(func(*Var)); ok {
		resolver(v)
		root = v.root.Load().val
	}
	return root
}

// RootVersion returns the identity of the current root binding. The returned
// pointer changes atomically whenever BindRoot or AlterRoot installs a root.
func (v *Var) RootVersion() *VarRootVersion {
	return v.root.Load().version
}

func (v *Var) Get() interface{} {
	if v.dynamicBindings.Load() == 0 {
		return v.getRoot()
	}
	return v.Deref()
}

func (v *Var) Set(val interface{}) interface{} {
	// TODO: validate
	b := v.getDynamicBinding()
	if b == nil {
		panic(fmt.Sprintf("can't change/establish root binding of: %s", v))
	}
	old := b.val
	b.val = val
	v.notifyWatches(old, val)
	return val
}

func (v *Var) Meta() IPersistentMap {
	value := v.meta.Load().(*Box).val
	if lazy, ok := value.(*lazyVarMeta); ok {
		lazy.once.Do(func() {
			lazy.meta = lazy.fn()
			lazy.meta = lazy.meta.Assoc(KWNS, v.ns).(IPersistentMap)
			lazy.fn = nil
		})
		return lazy.meta
	}
	return value.(IPersistentMap)
}

func (v *Var) SetMeta(meta IPersistentMap) {
	// TODO: ResetMeta
	v.isMacroCached.Store(0) // invalidate IsMacro cache
	meta = Assoc(meta, KWNS, v.ns).(IPersistentMap)
	v.meta.Store(NewBox(meta))
}

// SetMetaLazy defers construction of immutable metadata until it is observed.
// Generated AOT loaders use this for docstrings, arglists, and source metadata
// that most programs never inspect.
func (v *Var) SetMetaLazy(fn func() IPersistentMap) {
	v.isMacroCached.Store(0)
	v.meta.Store(newLazyMetaBox(fn))
}

// lazyMetaBox allocates the meta box and its lazy metadata together.
type lazyMetaBox struct {
	box  Box
	lazy lazyVarMeta
}

func newLazyMetaBox(fn func() IPersistentMap) *Box {
	b := &lazyMetaBox{}
	b.lazy.fn = fn
	b.box.val = &b.lazy
	return &b.box
}

// SetMetaLazyMacro is SetMetaLazy for generated code that already knows
// whether the metadata marks a macro, so IsMacro can answer without
// realizing the metadata.
func (v *Var) SetMetaLazyMacro(fn func() IPersistentMap, macro bool) {
	v.meta.Store(newLazyMetaBox(fn))
	if macro {
		v.isMacroCached.Store(2)
	} else {
		v.isMacroCached.Store(1)
	}
}

func (v *Var) AlterMeta(alter IFn, args ISeq) IPersistentMap {
	meta := alter.ApplyTo(NewCons(v.Meta(), args)).(IPersistentMap)
	v.SetMeta(meta)
	return meta
}

func (v *Var) IsMacro() bool {
	if cached := v.isMacroCached.Load(); cached != 0 {
		return cached == 2
	}
	meta := v.Meta()
	isMacro := meta.EntryAt(KWMacro)
	result := isMacro != nil && isMacro.Val() == true
	if result {
		v.isMacroCached.Store(2)
	} else {
		v.isMacroCached.Store(1)
	}
	return result
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

func (v *Var) IsDynamic() bool {
	return v.dynamic
}

func (v *Var) Deref() interface{} {
	if b := v.getDynamicBinding(); b != nil {
		return b.val
	}
	return v.getRoot()
}

func (v *Var) getDynamicBinding() *Box {
	if v.dynamicBindings.Load() == 0 {
		return nil
	}
	var storage *glStorage
	gid := getGoroutineID()

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

	oldRoot := v.Get()
	newRoot := alter.ApplyTo(NewCons(oldRoot, args))
	// TODO: validate
	v.root.Store(newVarRoot(newRoot))
	v.notifyWatches(oldRoot, newRoot)
	return newRoot
}

func (v *Var) SetValidator(vf IFn) {
	panic("not implemented")
}

func (v *Var) Validator() IFn {
	panic("not implemented")
}

func (v *Var) Watches() IPersistentMap {
	return v.watches
}

func (v *Var) AddWatch(key interface{}, fn IFn) IRef {
	v.watches = v.watches.Assoc(key, fn).(IPersistentMap)
	return v
}

func (v *Var) RemoveWatch(key interface{}) {
	v.watches = v.watches.Without(key)
}

func (v *Var) notifyWatches(oldVal, newVal interface{}) {
	watches := v.watches
	if watches == nil || watches.Count() == 0 {
		return
	}

	for seq := watches.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		key := entry.Key()
		fn := entry.Val().(IFn)
		// Call watch function with key, ref, old-state, new-state
		fn.Invoke(key, v, oldVal, newVal)
	}
}

func (v *Var) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(v)))
}

func (v *Var) fn() IFn {
	val := v.Deref()
	if _, ok := val.(*UnboundVar); ok {
		panic(fmt.Errorf("cannot call unbound var: %s/%s", v.ns.Name(), v.sym.Name()))
	}
	if val == nil {
		panic(fmt.Errorf("var %s/%s is bound to nil", v.ns.Name(), v.sym.Name()))
	}
	return val.(IFn)
}

func (v *Var) Invoke(args ...interface{}) interface{} {
	if !v.IsBound() {
		panic(fmt.Errorf("cannot call unbound var: %s/%s", v.ns.Name(), v.sym.Name()))
	}
	fn := v.fn()
	if fn == nil {
		panic(fmt.Errorf("var %s/%s is bound to nil", v.ns.Name(), v.sym.Name()))
	}
	return fn.Invoke(args...)
}

func (v *Var) ApplyTo(args ISeq) interface{} {
	if !v.IsBound() {
		panic(fmt.Errorf("cannot call unbound var: %s/%s", v.ns.Name(), v.sym.Name()))
	}
	fn := v.fn()
	if fn == nil {
		panic(fmt.Errorf("var %s/%s is bound to nil", v.ns.Name(), v.sym.Name()))
	}
	return fn.ApplyTo(args)
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

func getGoroutineID() int64 {
	return goid.Get()
}

func PushThreadBindings(bindings IPersistentMap) {
	gid := getGoroutineID()

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
		store[vr] = &Box{val: val}
		vr.dynamicBindings.Add(1)
	}
}

func PopThreadBindings() {
	gid := getGoroutineID()
	glsBindingsMtx.RLock()
	storage := glsBindings[gid]
	glsBindingsMtx.RUnlock()

	popped := storage.bindings[len(storage.bindings)-1]
	if len(storage.bindings) > 1 {
		storage.bindings = storage.bindings[:len(storage.bindings)-1]
	} else {
		glsBindingsMtx.Lock()
		delete(glsBindings, gid)
		glsBindingsMtx.Unlock()
	}

	for vr := range popped {
		vr.dynamicBindings.Add(-1)
	}
}

func GetThreadBindings() IPersistentMap {
	gid := getGoroutineID()
	glsBindingsMtx.RLock()
	storage := glsBindings[gid]
	glsBindingsMtx.RUnlock()

	var ret IPersistentMap = emptyMap
	if storage == nil {
		return ret
	}
	for i := len(storage.bindings) - 1; i >= 0; i-- {
		for v, b := range storage.bindings[i] {
			// most recent binding wins
			if ret.EntryAt(v) == nil {
				ret = ret.Assoc(v, b.val).(IPersistentMap)
			}
		}
	}
	return ret
}

func CloneThreadBindingFrame() any {
	gid := getGoroutineID()
	glsBindingsMtx.RLock()
	defer glsBindingsMtx.RUnlock()
	bindings := glsBindings[gid]
	if bindings == nil {
		return nil
	}
	return cloneGLStorage(bindings)
}

func ResetThreadBindingFrame(frame any) {
	gid := getGoroutineID()
	glsBindingsMtx.Lock()
	defer glsBindingsMtx.Unlock()
	old := glsBindings[gid]
	adjustDynamicBindingCounts(old, -1)
	if frame == nil {
		delete(glsBindings, gid)
		return
	}
	replacement := cloneGLStorage(frame.(*glStorage))
	glsBindings[gid] = replacement
	adjustDynamicBindingCounts(replacement, 1)
}

func adjustDynamicBindingCounts(storage *glStorage, delta int64) {
	if storage == nil {
		return
	}
	for _, bindings := range storage.bindings {
		for vr := range bindings {
			vr.dynamicBindings.Add(delta)
		}
	}
}

func cloneGLStorage(storage *glStorage) *glStorage {
	clone := &glStorage{
		bindings: make([]varBindings, len(storage.bindings)),
	}
	for i, bindings := range storage.bindings {
		frame := make(varBindings, len(bindings))
		for vr, box := range bindings {
			frame[vr] = &Box{val: box.val}
		}
		clone.bindings[i] = frame
	}
	return clone
}
