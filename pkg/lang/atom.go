package lang

import "sync/atomic"

type (
	Atom struct {
		state atomic.Value
	}
)

var (
	_ IAtom2 = (*Atom)(nil)
	_ IRef   = (*Atom)(nil)
)

func NewAtom(val interface{}) *Atom {
	a := &Atom{}
	a.state.Store(Box{val})
	return a
}

func (a *Atom) Deref() interface{} {
	return a.state.Load().(Box).val
}

func (a *Atom) SetValidator(vf IFn)              { panic("not implemented") }
func (a *Atom) Validator() IFn                   { panic("not implemented") }
func (a *Atom) Watches() IPersistentMap          { panic("not implemented") }
func (a *Atom) AddWatch(key interface{}, fn IFn) { panic("not implemented") }
func (a *Atom) RemoveWatch(key interface{})      { panic("not implemented") }

func (a *Atom) Swap(f IFn, args ISeq) interface{} {
	for {
		old := a.state.Load().(Box)
		nw := f.ApplyTo(NewCons(old.val, args))
		if a.CompareAndSet(old.val, nw) {
			return nw
		}
	}
}

func (a *Atom) CompareAndSet(oldv, newv interface{}) bool {
	// TODO: validate
	// TODO: notifyWatches
	return a.state.CompareAndSwap(Box{val: oldv}, Box{val: newv})
}

func (a *Atom) Reset(newVal interface{}) interface{} {
	// old := a.state.Load().(Box)
	// TODO: validate

	a.state.Store(Box{newVal})
	// TODO: notifyWatches
	return newVal
}
