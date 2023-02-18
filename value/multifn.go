package value

import (
	"fmt"
	"sync"
)

type MultiFn struct {
	mtx sync.RWMutex

	methodTable IPersistentMap
	// TODO: cache
}

func NewMultiFn(name string, dispatchFn Applyer, defaultDispatchVal interface{}, hierarchy IRef) *MultiFn {
	return &MultiFn{
		methodTable: emptyMap,
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

func (m *MultiFn) Invoke(args ...interface{}) (interface{}, error) {
	fmt.Println("MultiFn.Invoke", args)
	panic("not implemented")
}

func (m *MultiFn) Apply(env Environment, args []interface{}) (interface{}, error) {
	fmt.Println("MultiFn.Invoke", args)
	panic("not implemented")
}
