package lang

import "sync/atomic"

type (
	Atom struct {
		state   atomic.Value
		watches IPersistentMap

		meta IPersistentMap
	}
)

var (
	_ IAtom2 = (*Atom)(nil)
	_ IRef   = (*Atom)(nil)
)

func NewAtom(val any) *Atom {
	a := &Atom{}
	a.state.Store(Box{val})
	a.watches = emptyMap
	return a
}

func NewAtomWithMeta(val any, meta IPersistentMap) *Atom {
	a := NewAtom(val)
	if meta != nil {
		a.meta = meta
	}
	return a
}

func (a *Atom) Deref() interface{} {
	return a.state.Load().(Box).val
}

func (a *Atom) SetValidator(vf IFn) { panic("not implemented") }
func (a *Atom) Validator() IFn      { panic("not implemented") }
func (a *Atom) Watches() IPersistentMap {
	return a.watches
}

func (a *Atom) AddWatch(key interface{}, fn IFn) IRef {
	a.watches = a.watches.Assoc(key, fn).(IPersistentMap)
	return a
}

func (a *Atom) RemoveWatch(key interface{}) {
	a.watches = a.watches.Without(key)
}

func (a *Atom) notifyWatches(oldVal, newVal interface{}) {
	watches := a.watches
	if watches == nil || watches.Count() == 0 {
		return
	}

	for seq := watches.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		key := entry.Key()
		fn := entry.Val().(IFn)
		// Call watch function with key, ref, old-state, new-state
		fn.Invoke(key, a, oldVal, newVal)
	}
}

func (a *Atom) Swap(f IFn, args ISeq) interface{} {
	for {
		old := a.state.Load().(Box)
		nw := f.ApplyTo(NewCons(old.val, args))
		if a.CompareAndSet(old.val, nw) {
			return nw
		}
	}
}

// Swap0, Swap1, and Swap2 are fixed-arity counterparts to Swap. They avoid
// constructing an argument sequence when callers already know the number of
// additional arguments, while preserving Atom's compare-and-set retry loop.
func (a *Atom) Swap0(f IFn) interface{} {
	for {
		old := a.state.Load().(Box)
		nw := Apply1(f, old.val)
		if a.CompareAndSet(old.val, nw) {
			return nw
		}
	}
}

func (a *Atom) Swap1(f IFn, x interface{}) interface{} {
	for {
		old := a.state.Load().(Box)
		nw := Apply2(f, old.val, x)
		if a.CompareAndSet(old.val, nw) {
			return nw
		}
	}
}

func (a *Atom) Swap2(f IFn, x, y interface{}) interface{} {
	for {
		old := a.state.Load().(Box)
		nw := Apply3(f, old.val, x, y)
		if a.CompareAndSet(old.val, nw) {
			return nw
		}
	}
}

func (a *Atom) CompareAndSet(oldv, newv interface{}) bool {
	// TODO: validate
	swapped := a.state.CompareAndSwap(Box{val: oldv}, Box{val: newv})
	if swapped {
		a.notifyWatches(oldv, newv)
	}
	return swapped
}

func (a *Atom) Reset(newVal interface{}) interface{} {
	old := a.state.Load().(Box)
	// TODO: validate

	a.state.Store(Box{newVal})
	a.notifyWatches(old.val, newVal)
	return newVal
}

func (a *Atom) Meta() IPersistentMap {
	if a.meta == nil {
		return nil
	}
	return a.meta
}
