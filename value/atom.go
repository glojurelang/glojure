package value

import "sync/atomic"

type (
	Atom struct {
		state atomic.Value
	}

	// atomBox is a wrapper around a value stored in an atom. Because
	// atomic.Value requires that all values loaded and stored must be
	// of the same concrete type, we need to wrap the value in a struct.
	atomBox struct {
		val interface{}
	}
)

var (
	_ IAtom2 = (*Atom)(nil)
	_ IRef   = (*Atom)(nil)
)

func NewAtom(val interface{}) *Atom {
	a := &Atom{}
	a.state.Store(atomBox{val})
	return a
}

func (a *Atom) Deref() interface{} {
	return a.state.Load().(atomBox).val
}

func (a *Atom) SetValidator(vf IFn)              { panic("not implemented") }
func (a *Atom) Validator() IFn                   { panic("not implemented") }
func (a *Atom) Watches() IPersistentMap          { panic("not implemented") }
func (a *Atom) AddWatch(key interface{}, fn IFn) { panic("not implemented") }
func (a *Atom) RemoveWatch(key interface{})      { panic("not implemented") }

func (a *Atom) Reset(newVal interface{}) interface{} {
	// old := a.state.Load().(atomBox)
	// TODO: validate

	a.state.Store(atomBox{newVal})
	// TODO: notifyWatches
	return newVal
}
