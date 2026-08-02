package lang

import (
	"fmt"
	"reflect"
	"sync"
)

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

	mtx sync.RWMutex
}

var (
	_ IFn = (*MultiFn)(nil)

	varIsA = InternVarName(NSCore.Name(), NewSymbol("isa?"))
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

// registerWellKnownMethods seeds a freshly created MultiFn with Go-side
// methods that the stdlib alone can't supply. It handles host-class printing
// and bridges the Clojure IKVReduce protocol to Glojure's native interface.
func registerWellKnownMethods(mf *MultiFn) {
	switch mf.name {
	case "print-method", "print-dup":
		mf.AddMethod(reflect.TypeOf((*Class)(nil)), classPrintMethod)
	case "kv-reduce":
		mf.AddMethod(nil, kvReduceNilMethod)
		mf.AddMethod(reflect.TypeOf((*Object)(nil)).Elem(), kvReduceObjectMethod)
		mf.AddMethod(reflect.TypeOf((*IKVReduce)(nil)).Elem(), kvReduceInterfaceMethod)
	}
}

var (
	kvReduceNilMethod = FnFunc3(func(_, _, init any) any {
		return init
	})
	kvReduceObjectMethod = FnFunc3(func(amap, f, init any) any {
		for entries := Seq(amap); entries != nil; entries = entries.Next() {
			entry := entries.First().(IMapEntry)
			init = Apply3(f, init, entry.Key(), entry.Val())
			if IsReduced(init) {
				return init.(IDeref).Deref()
			}
		}
		return init
	})
	kvReduceInterfaceMethod = FnFunc3(func(amap, f, init any) any {
		return amap.(IKVReduce).KVReduce(MustHostCast[IFn](f), init)
	})
)

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
	case "kv-reduce":
		m, ok := method.(FnFunc3)
		if !ok {
			return false
		}
		switch dispatchVal {
		case nil:
			return reflect.ValueOf(m).Pointer() == reflect.ValueOf(kvReduceNilMethod).Pointer()
		case reflect.TypeOf((*Object)(nil)).Elem():
			return reflect.ValueOf(m).Pointer() == reflect.ValueOf(kvReduceObjectMethod).Pointer()
		case reflect.TypeOf((*IKVReduce)(nil)).Elem():
			return reflect.ValueOf(m).Pointer() == reflect.ValueOf(kvReduceInterfaceMethod).Pointer()
		}
	}
	return false
}

func (m *MultiFn) resetCache() {
	m.methodCache = emptyMap
	m.cachedHierarchy = m.hierarchy.Deref()
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
	return m.getFn(m.dispatchFn.Invoke(args...)).Invoke(args...)
}

func (m *MultiFn) ApplyTo(args ISeq) any {
	return m.Invoke(seqToSlice(args)...)
}

func (m *MultiFn) getMethod(dispatchVal any) IFn {
	m.mtx.Lock()
	if m.cachedHierarchy != m.hierarchy.Deref() {
		m.resetCache()
	}
	targetFn := m.methodCache.ValAt(dispatchVal)
	m.mtx.Unlock()
	if targetFn != nil {
		return targetFn.(IFn)
	}
	return m.findAndCacheBestMethod(dispatchVal)
}

func (m *MultiFn) getFn(dispatchVal any) IFn {
	targetFn := m.getMethod(dispatchVal)
	if targetFn == nil {
		panic(fmt.Errorf("No method in multimethod '%s' for dispatch value: %v", m.name, ToString(dispatchVal)))
	}
	return targetFn
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
