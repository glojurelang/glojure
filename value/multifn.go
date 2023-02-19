package value

import (
	"fmt"
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
	val, err := m.Apply(GlobalEnv, args)
	if err != nil {
		panic(err)
	}
	return val
}

func (m *MultiFn) Apply(env Environment, args []interface{}) (interface{}, error) {
	return m.getFn(m.dispatchFn.Invoke(args...)).Invoke(args...), nil
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

	entry := m.methodTable.EntryAt(dispatchVal)
	if entry == nil {
		entry = m.methodTable.EntryAt(m.defaultDispatchVal)
	}
	return entry.Val().(IFn)
}
