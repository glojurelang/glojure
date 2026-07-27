package lang

import (
	"sync"
	"sync/atomic"
)

type (
	Atom struct {
		state   atomic.Pointer[Box]
		initial Box

		referenceMu sync.RWMutex
		watches     IPersistentMap
		validator   IFn
		meta        IPersistentMap
	}
)

var (
	_ IAtom2 = (*Atom)(nil)
	_ IRef   = (*Atom)(nil)
)

func NewAtom(val any) *Atom {
	a := &Atom{initial: Box{val: val}}
	a.state.Store(&a.initial)
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
	return a.state.Load().val
}

func (a *Atom) SetValidator(vf IFn) {
	if vf != nil && !IsTruthy(Apply1(vf, a.Deref())) {
		panic(NewIllegalStateError("Invalid reference state"))
	}
	a.referenceMu.Lock()
	a.validator = vf
	a.referenceMu.Unlock()
}

func (a *Atom) Validator() IFn {
	a.referenceMu.RLock()
	defer a.referenceMu.RUnlock()
	return a.validator
}

// GetValidator is the JVM-style IRef alias used by Clojure host interop.
func (a *Atom) GetValidator() IFn {
	return a.Validator()
}

func (a *Atom) Watches() IPersistentMap {
	a.referenceMu.RLock()
	defer a.referenceMu.RUnlock()
	return a.watches
}

func (a *Atom) AddWatch(key interface{}, fn IFn) IRef {
	a.referenceMu.Lock()
	defer a.referenceMu.Unlock()
	a.watches = a.watches.Assoc(key, fn).(IPersistentMap)
	return a
}

func (a *Atom) RemoveWatch(key interface{}) {
	a.referenceMu.Lock()
	defer a.referenceMu.Unlock()
	a.watches = a.watches.Without(key)
}

func (a *Atom) notifyWatches(oldVal, newVal interface{}) {
	a.referenceMu.RLock()
	watches := a.watches
	a.referenceMu.RUnlock()
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

func (a *Atom) validate(newVal interface{}) {
	a.referenceMu.RLock()
	validator := a.validator
	a.referenceMu.RUnlock()
	if validator != nil && !IsTruthy(Apply1(validator, newVal)) {
		panic(NewIllegalStateError("Invalid reference state"))
	}
}

func (a *Atom) hasWatches() bool {
	a.referenceMu.RLock()
	defer a.referenceMu.RUnlock()
	return a.watches != nil && a.watches.Count() != 0
}

func (a *Atom) Swap(f IFn, args ISeq) interface{} {
	for {
		old := a.state.Load()
		nw := f.ApplyTo(NewCons(old.val, args))
		a.validate(nw)
		if a.compareAndSetBox(old, nw) {
			return nw
		}
	}
}

// Swap0, Swap1, and Swap2 are fixed-arity counterparts to Swap. They avoid
// constructing an argument sequence when callers already know the number of
// additional arguments, while preserving Atom's compare-and-set retry loop.
func (a *Atom) Swap0(f IFn) interface{} {
	for {
		old := a.state.Load()
		nw := Apply1(f, old.val)
		a.validate(nw)
		if a.compareAndSetBox(old, nw) {
			return nw
		}
	}
}

func (a *Atom) Swap1(f IFn, x interface{}) interface{} {
	for {
		old := a.state.Load()
		nw := Apply2(f, old.val, x)
		a.validate(nw)
		if a.compareAndSetBox(old, nw) {
			return nw
		}
	}
}

func (a *Atom) Swap2(f IFn, x, y interface{}) interface{} {
	for {
		old := a.state.Load()
		nw := Apply3(f, old.val, x, y)
		a.validate(nw)
		if a.compareAndSetBox(old, nw) {
			return nw
		}
	}
}

// SwapFunc is the direct Go-callback counterpart to Swap0. Compiled code uses
// it when a swap! callback is a non-escaping function literal, avoiding an IFn
// wrapper while retaining retry, watch, and identity semantics.
func (a *Atom) SwapFunc(f func(interface{}) interface{}) interface{} {
	for {
		old := a.state.Load()
		nw := f(old.val)
		if a.compareAndSetBox(old, nw) {
			return nw
		}
	}
}

func (a *Atom) CompareAndSet(oldv, newv interface{}) bool {
	old := a.state.Load()
	if !Identical(old.val, oldv) {
		return false
	}
	a.validate(newv)
	return a.compareAndSetBox(old, newv)
}

func (a *Atom) compareAndSetBox(old *Box, newv interface{}) bool {
	if Identical(old.val, newv) &&
		!a.hasWatches() {
		return a.state.CompareAndSwap(old, old)
	}
	swapped := a.state.CompareAndSwap(old, NewBox(newv))
	if swapped {
		a.notifyWatches(old.val, newv)
	}
	return swapped
}

func (a *Atom) Reset(newVal interface{}) interface{} {
	a.validate(newVal)
	for {
		old := a.state.Load()
		if Identical(old.val, newVal) &&
			!a.hasWatches() {
			if a.state.CompareAndSwap(old, old) {
				return newVal
			}
			continue
		}
		old = a.state.Swap(NewBox(newVal))
		a.notifyWatches(old.val, newVal)
		return newVal
	}
}

func (a *Atom) Meta() IPersistentMap {
	a.referenceMu.RLock()
	defer a.referenceMu.RUnlock()
	return a.meta
}

func (a *Atom) AlterMeta(f IFn, args ISeq) IPersistentMap {
	meta := ApplySeq(f, NewCons(a.Meta(), args))
	if meta == nil {
		return a.ResetMeta(nil)
	}
	return a.ResetMeta(meta.(IPersistentMap))
}

func (a *Atom) ResetMeta(meta IPersistentMap) IPersistentMap {
	a.referenceMu.Lock()
	a.meta = meta
	a.referenceMu.Unlock()
	return meta
}
