package value

import (
	"fmt"
	"reflect"
	"sync"
)

type MultiFn struct {
	name               string
	dispatchFn         IFn
	defaultDispatchVal interface{}
	methodTable        IPersistentMap
	// TODO: cache

	mtx sync.RWMutex
}

var (
	_ IFn = (*MultiFn)(nil)
)

func NewMultiFn(name string, dispatchFn IFn, defaultDispatchVal interface{}, hierarchy IRef) *MultiFn {
	return &MultiFn{
		name:               name,
		dispatchFn:         dispatchFn,
		defaultDispatchVal: defaultDispatchVal,
		methodTable:        emptyMap,
	}
}

func (m *MultiFn) AddMethod(dispatchVal interface{}, method IFn) *MultiFn {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.methodTable = m.methodTable.Assoc(dispatchVal, method).(IPersistentMap)

	return m
}

func (m *MultiFn) PreferMethod(dispatchValX, dispatchValY interface{}) *MultiFn {
	// TODO
	return m
}

func (m *MultiFn) Invoke(args ...interface{}) interface{} {
	return m.getFn(m.dispatchFn.Invoke(args...)).Invoke(args...)
}

func (m *MultiFn) ApplyTo(args ISeq) interface{} {
	return m.Invoke(seqToSlice(args)...)
}

func (m *MultiFn) getFn(dispatchVal interface{}) IFn {
	targetFn := m.getMethod(dispatchVal)
	if targetFn == nil {
		panic(fmt.Errorf("No method in multimethod '%s' for dispatch value: %v", m.name, ToString(dispatchVal)))
	}
	return targetFn
}

func (m *MultiFn) getMethod(dispatchVal interface{}) IFn {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	// TODO: proper hierarchy and implement a cache

	entry := m.methodTable.EntryAt(dispatchVal)
	if entry != nil {
		return entry.Val().(IFn)
	}

	var bestMatch IFn
	for seq := Seq(m.methodTable); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		if m.isA(dispatchVal, entry.Key()) {
			bestMatch = entry.Val().(IFn)
			break
		}
	}
	if bestMatch != nil {
		return bestMatch
	}

	entry = m.methodTable.EntryAt(m.defaultDispatchVal)
	if entry == nil {
		return nil
	}
	return entry.Val().(IFn)
}

func (m *MultiFn) isA(x, y interface{}) bool {
	child, ok := x.(reflect.Type)
	if !ok {
		return false
	}
	parent, ok := y.(reflect.Type)
	if !ok {
		return false
	}
	return child.AssignableTo(parent) || child.Kind() == reflect.Pointer && child.Elem().AssignableTo(parent)
}
