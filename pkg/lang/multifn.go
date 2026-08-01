package lang

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

type protocolMethodCache struct {
	generation uint64
	dispatch   any
	method     IFn
}

type MultiFn struct {
	// TODO: take a pass at thread-safety. the java impl relies on
	// volatiles.

	name               string
	dispatchFn         IFn
	defaultDispatchVal any
	hierarchy          IRef
	methodTable        IPersistentMap
	preferTable        IPersistentMap
	methodCache        IPersistentMap
	cachedHierarchy    any
	protocol           bool
	generation         atomic.Uint64
	protocolGeneration atomic.Uint64
	protocolMethod     atomic.Pointer[protocolMethodCache]

	mtx sync.RWMutex
}

var (
	_ IFn           = (*MultiFn)(nil)
	_ FixedArityFn0 = (*MultiFn)(nil)
	_ FixedArityFn1 = (*MultiFn)(nil)
	_ FixedArityFn2 = (*MultiFn)(nil)
	_ FixedArityFn3 = (*MultiFn)(nil)
	_ FixedArityFn4 = (*MultiFn)(nil)
	_ FixedArityFn5 = (*MultiFn)(nil)

	varIsA = InternVarName(NSCore.Name(), NewSymbol("isa?"))

	multiFnParentsKeyword     = NewKeyword("parents")
	multiFnAncestorsKeyword   = NewKeyword("ancestors")
	multiFnDescendantsKeyword = NewKeyword("descendants")
)

func NewMultiFn(name string, dispatchFn IFn, defaultDispatchVal any, hierarchy IRef) *MultiFn {
	mf := &MultiFn{
		name:               name,
		dispatchFn:         dispatchFn,
		defaultDispatchVal: defaultDispatchVal,
		methodTable:        emptyMap,
		preferTable:        emptyMap,
		methodCache:        emptyMap,
		hierarchy:          hierarchy,
	}
	registerWellKnownMethods(mf)
	return mf
}

// NewProtocolMultiFn returns a multimethod whose dispatch value is the
// concrete type of its first argument. Protocol extension remains dynamic,
// while fixed-arity calls avoid the variadic dispatch wrapper used by
// defmulti.
func NewProtocolMultiFn(name string, hierarchy IRef) *MultiFn {
	mf := NewMultiFn(
		name,
		protocolDispatchFn{},
		NewKeyword("default"),
		hierarchy,
	)
	mf.protocol = true
	return mf
}

// IsProtocol reports whether m uses protocol type dispatch.
func (m *MultiFn) IsProtocol() bool {
	return m.protocol
}

// ProtocolGeneration changes whenever protocol method selection may change.
// Compiled monomorphic call sites use it to retain dynamic re-extension
// semantics while calling a cached method directly.
func (m *MultiFn) ProtocolGeneration() uint64 {
	return m.protocolGeneration.Load()
}

// registerWellKnownMethods seeds a freshly created MultiFn with any
// Go-side default methods that the stdlib alone can't supply. Currently
// just installs a *Class print-method so host-class values seeded into
// `(ns-imports *ns*)` print as their FQ Java name instead of falling
// through to the catch-all Object handler.
func registerWellKnownMethods(mf *MultiFn) {
	switch mf.name {
	case "print-method", "print-dup":
		mf.AddMethod(reflect.TypeOf((*Class)(nil)), classPrintMethod)
	}
}

// IsAutoRegisteredMethod reports whether (dispatchVal, method) is an
// entry seeded by registerWellKnownMethods for a MultiFn named mfName.
// AOT codegen uses this to skip re-emitting these entries: the compiled
// binary's lang.NewMultiFn call seeds them automatically, and the method
// values are opaque Go FnFuncs that codegen cannot serialize.
func IsAutoRegisteredMethod(mfName string, dispatchVal any, method any) bool {
	switch mfName {
	case "print-method", "print-dup":
		dv, ok := dispatchVal.(reflect.Type)
		if !ok || dv != reflect.TypeOf((*Class)(nil)) {
			return false
		}
		m, ok := method.(FnFunc)
		if !ok {
			return false
		}
		return reflect.ValueOf(m).Pointer() == reflect.ValueOf(classPrintMethod).Pointer()
	}
	return false
}

func (m *MultiFn) resetCache() {
	m.methodCache = emptyMap
	m.cachedHierarchy = m.hierarchy.Deref()
	m.generation.Add(1)
	if m.protocol {
		m.protocolGeneration.Add(1)
		m.protocolMethod.Store(nil)
	}
}

// Generation changes whenever method selection may change.
func (m *MultiFn) Generation() uint64 {
	return m.generation.Load()
}

// IsGeneration verifies both the method/preference generation and the
// hierarchy snapshot. Compiled call sites use it before selecting a method
// directly; a changed hierarchy invalidates the ordinary method cache and the
// compiled fast path together.
func (m *MultiFn) IsGeneration(expected uint64) bool {
	if m.generation.Load() != expected {
		return false
	}
	hierarchy := m.hierarchy.Deref()
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.generation.Load() != expected {
		return false
	}
	if m.cachedHierarchy != hierarchy {
		m.resetCache()
		return false
	}
	return true
}

// ExactGeneration snapshots a generation only when method selection has no
// preferences or hierarchy relationships. Compiled exact-value dispatchers
// call this once when a namespace loads, then use IsGeneration at call sites.
func (m *MultiFn) ExactGeneration() (uint64, bool) {
	hierarchy := m.hierarchy.Deref()
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.cachedHierarchy != hierarchy {
		m.resetCache()
	}
	exact := m.preferTable.Count() == 0
	for _, keyword := range []Keyword{
		multiFnParentsKeyword,
		multiFnAncestorsKeyword,
		multiFnDescendantsKeyword,
	} {
		value := Get(hierarchy, keyword)
		if !IsNil(value) && Count(value) != 0 {
			exact = false
			break
		}
	}
	return m.generation.Load(), exact
}

func (m *MultiFn) GetMethodTable() IPersistentMap {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.methodTable
}

func (m *MultiFn) GetDispatchFn() IFn {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.dispatchFn
}

func (m *MultiFn) GetDefaultDispatchVal() any {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.defaultDispatchVal
}

func (m *MultiFn) GetHierarchy() IRef {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.hierarchy
}

func (m *MultiFn) GetName() string {
	return m.name
}

func (m *MultiFn) PreferTable() IPersistentMap {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.preferTable
}

func (m *MultiFn) AddMethod(dispatchVal any, method IFn) *MultiFn {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.methodTable = m.methodTable.Assoc(dispatchVal, method).(IPersistentMap)
	m.resetCache()

	return m
}

func (m *MultiFn) PreferMethod(dispatchValX, dispatchValY any) *MultiFn {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.prefers(m.hierarchy.Deref(), dispatchValY, dispatchValX) {
		panic(fmt.Errorf("Preference conflict in multimethod '%s': %s is already preferred to %s", m.name, dispatchValY, dispatchValX))
	}

	m.preferTable = m.preferTable.Assoc(dispatchValX, GetDefault(m.preferTable, dispatchValX, emptySet).(Conser).Cons(dispatchValY)).(IPersistentMap)

	m.resetCache()

	return m
}

func (m *MultiFn) prefers(hierarchy, x, y any) (res bool) {
	xprefs := m.preferTable.ValAt(x)
	if xprefs != nil && xprefs.(IPersistentSet).Contains(y) {
		return true
	}

	// TODO: how much of this even makes sense for go

	for ps := Seq(VarParents.Invoke(hierarchy, y)); ps != nil; ps = ps.Next() {
		if m.prefers(hierarchy, x, ps.First()) {
			return true
		}
	}
	for ps := Seq(VarParents.Invoke(hierarchy, x)); ps != nil; ps = ps.Next() {
		if m.prefers(hierarchy, ps.First(), y) {
			return true
		}
	}

	// Some go-specific logic
	// TODO: Vet go-specific multi-method preference logic.
	// for now, prefer x if x is more specific than y
	xType, ok := x.(reflect.Type)
	if !ok {
		return false
	}
	yType, ok := y.(reflect.Type)
	if !ok {
		return false
	}
	if xType.AssignableTo(yType) || reflect.PointerTo(xType).AssignableTo(yType) {
		return true
	}

	return false
}

func (m *MultiFn) Invoke(args ...any) any {
	switch len(args) {
	case 0:
		return m.Invoke0()
	case 1:
		return m.Invoke1(args[0])
	case 2:
		return m.Invoke2(args[0], args[1])
	case 3:
		return m.Invoke3(args[0], args[1], args[2])
	case 4:
		return m.Invoke4(args[0], args[1], args[2], args[3])
	case 5:
		return m.Invoke5(args[0], args[1], args[2], args[3], args[4])
	}
	return m.invokeArgs(args)
}

func (m *MultiFn) invokeArgs(args []any) any {
	return m.getFn(m.dispatchFn.Invoke(args...)).Invoke(args...)
}

func (m *MultiFn) Invoke0() any {
	return Apply0(m.getFn(Apply0(m.dispatchFn)))
}

func (m *MultiFn) Invoke1(a0 any) any {
	if !hasDirectFixedArity(m.dispatchFn, 1) {
		return m.invokeArgs([]any{a0})
	}
	target := m.getFn(Apply1(m.dispatchFn, a0))
	if hasDirectFixedArity(target, 1) {
		return Apply1(target, a0)
	}
	return target.Invoke(a0)
}

func (m *MultiFn) Invoke2(a0, a1 any) any {
	if !hasDirectFixedArity(m.dispatchFn, 2) {
		return m.invokeArgs([]any{a0, a1})
	}
	target := m.getFn(Apply2(m.dispatchFn, a0, a1))
	if hasDirectFixedArity(target, 2) {
		return Apply2(target, a0, a1)
	}
	return target.Invoke(a0, a1)
}

func (m *MultiFn) Invoke3(a0, a1, a2 any) any {
	if !hasDirectFixedArity(m.dispatchFn, 3) {
		return m.invokeArgs([]any{a0, a1, a2})
	}
	target := m.getFn(Apply3(m.dispatchFn, a0, a1, a2))
	if hasDirectFixedArity(target, 3) {
		return Apply3(target, a0, a1, a2)
	}
	return target.Invoke(a0, a1, a2)
}

func (m *MultiFn) Invoke4(a0, a1, a2, a3 any) any {
	if !hasDirectFixedArity(m.dispatchFn, 4) {
		return m.invokeArgs([]any{a0, a1, a2, a3})
	}
	target := m.getFn(Apply4(m.dispatchFn, a0, a1, a2, a3))
	if hasDirectFixedArity(target, 4) {
		return Apply4(target, a0, a1, a2, a3)
	}
	return target.Invoke(a0, a1, a2, a3)
}

func (m *MultiFn) Invoke5(a0, a1, a2, a3, a4 any) any {
	if !hasDirectFixedArity(m.dispatchFn, 5) {
		return m.invokeArgs([]any{a0, a1, a2, a3, a4})
	}
	target := m.getFn(Apply5(m.dispatchFn, a0, a1, a2, a3, a4))
	if hasDirectFixedArity(target, 5) {
		return Apply5(target, a0, a1, a2, a3, a4)
	}
	return target.Invoke(a0, a1, a2, a3, a4)
}

func (m *MultiFn) ApplyTo(args ISeq) any {
	return m.getFn(m.dispatchFn.ApplyTo(args)).ApplyTo(args)
}

func (m *MultiFn) getMethod(dispatchVal any) IFn {
	var protocolGeneration uint64
	if m.protocol {
		protocolGeneration = m.protocolGeneration.Load()
		if cached := m.protocolMethod.Load(); cached != nil &&
			cached.generation == protocolGeneration &&
			cached.dispatch == dispatchVal {
			return cached.method
		}
	}
	m.mtx.Lock()
	if m.cachedHierarchy != m.hierarchy.Deref() {
		m.resetCache()
	}
	targetFn := m.methodCache.ValAt(dispatchVal)
	m.mtx.Unlock()
	if targetFn != nil {
		method := targetFn.(IFn)
		m.cacheProtocolMethod(protocolGeneration, dispatchVal, method)
		return method
	}
	method := m.findAndCacheBestMethod(dispatchVal)
	m.cacheProtocolMethod(protocolGeneration, dispatchVal, method)
	return method
}

func (m *MultiFn) cacheProtocolMethod(
	generation uint64,
	dispatch any,
	method IFn,
) {
	if !m.protocol || method == nil ||
		m.protocolGeneration.Load() != generation {
		return
	}
	m.protocolMethod.Store(&protocolMethodCache{
		generation: generation,
		dispatch:   dispatch,
		method:     method,
	})
}

func (m *MultiFn) getFn(dispatchVal any) IFn {
	targetFn := m.getMethod(dispatchVal)
	if targetFn == nil {
		panic(fmt.Errorf("No method in multimethod '%s' for dispatch value: %v", m.name, ToString(dispatchVal)))
	}
	return targetFn
}

// MethodForDispatch performs ordinary multimethod selection for an already
// evaluated dispatch value. A compiled speculative dispatcher uses this to
// fall back without executing an effectful dispatch function a second time.
func (m *MultiFn) MethodForDispatch(dispatchVal any) IFn {
	return m.getFn(dispatchVal)
}

func (m *MultiFn) findAndCacheBestMethod(dispatchVal any) IFn {
	m.mtx.RLock()
	mt := m.methodTable
	pt := m.preferTable
	ch := m.cachedHierarchy
	m.mtx.RUnlock()

	bestMethod := m.findBestMethod(dispatchVal)

	m.mtx.Lock()
	if mt != m.methodTable || pt != m.preferTable || ch != m.cachedHierarchy || m.cachedHierarchy != m.hierarchy.Deref() {
		m.resetCache()
		m.mtx.Unlock()
		return m.findAndCacheBestMethod(dispatchVal)
	}
	defer m.mtx.Unlock()

	m.methodCache = m.methodCache.Assoc(dispatchVal, bestMethod).(IPersistentMap)
	return bestMethod
}

func (m *MultiFn) findBestMethod(dispatchVal any) IFn {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	hierarchy := m.cachedHierarchy
	var bestValue any
	var bestEntry IMapEntry
	for seq := Seq(m.methodTable); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		if m.isA(hierarchy, dispatchVal, entry.Key()) {
			if bestEntry == nil || m.dominates(hierarchy, entry.Key(), bestEntry.Key()) {
				bestEntry = entry
			}
			if !m.dominates(hierarchy, bestEntry.Key(), entry.Key()) {
				panic(fmt.Errorf("Multiple methods in multimethod '%s' match dispatch value: %v -> %v and %v, and neither is preferred", m.name, dispatchVal, entry.Key(), bestEntry.Key()))
			}
		}
	}
	if bestEntry == nil {
		bestValue = m.methodTable.ValAt(m.defaultDispatchVal)
		if bestValue == nil {
			return nil
		}
	} else {
		bestValue = bestEntry.Val()
	}

	return bestValue.(IFn)
}

func (m *MultiFn) isA(h, x, y any) bool {
	return varIsA.Invoke(h, x, y).(bool)
}

func (m *MultiFn) dominates(h, x, y any) bool {
	return m.prefers(h, x, y) || m.isA(h, x, y)
}

type protocolDispatchFn struct{}

func (protocolDispatchFn) Invoke(args ...any) any {
	if len(args) == 0 {
		panic(NewIllegalArgumentError("protocol method requires a target"))
	}
	return protocolDispatchValue(args[0])
}

func (protocolDispatchFn) Invoke1(a0 any) any {
	return protocolDispatchValue(a0)
}

func (protocolDispatchFn) Invoke2(a0, _ any) any {
	return protocolDispatchValue(a0)
}

func (protocolDispatchFn) Invoke3(a0, _, _ any) any {
	return protocolDispatchValue(a0)
}

func (protocolDispatchFn) Invoke4(a0, _, _, _ any) any {
	return protocolDispatchValue(a0)
}

func (protocolDispatchFn) Invoke5(a0, _, _, _, _ any) any {
	return protocolDispatchValue(a0)
}

func (protocolDispatchFn) ApplyTo(args ISeq) any {
	if args == nil {
		panic(NewIllegalArgumentError("protocol method requires a target"))
	}
	return protocolDispatchValue(args.First())
}

func (protocolDispatchFn) Meta() IPersistentMap {
	return nil
}

func (p protocolDispatchFn) WithMeta(IPersistentMap) any {
	return p
}

func (protocolDispatchFn) IsFnValue() {}

func protocolDispatchValue(target any) any {
	if target == nil {
		return nil
	}
	return reflect.TypeOf(target)
}
