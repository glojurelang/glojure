package lang

import "sync"

type Volatile struct {
	val interface{}
	mtx sync.RWMutex
}

var (
	_ IDeref = (*Volatile)(nil)
)

func NewVolatile(val interface{}) *Volatile {
	return &Volatile{
		val: val,
	}
}

func (v *Volatile) Deref() interface{} {
	v.mtx.RLock()
	defer v.mtx.RUnlock()
	return v.val
}

func (v *Volatile) Reset(val interface{}) interface{} {
	v.mtx.Lock()
	defer v.mtx.Unlock()
	v.val = val
	return val
}
