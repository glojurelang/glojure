package value

import (
	"errors"
	"sync/atomic"
)

// Ref is a reference to a value that can be updated transactionally.
type Ref struct {
	val interface{}
}

func NewRef(val interface{}) *Ref {
	// TODO: implement for real
	return &Ref{
		val: val,
	}
}

func (r *Ref) Deref() interface{} {
	return r.val
}

func (r *Ref) Commute(fn IFn, args ISeq) interface{} {
	return LockingTransaction.doCommute(r, fn, args)
}

type LockingTransactor struct {
	txCount atomic.Int64
}

var (
	LockingTransaction = &LockingTransactor{}

	ErrNoTransaction = errors.New("no transaction running")
)

func (lt *LockingTransactor) RunInTransaction(fn IFn) interface{} {
	lt.txCount.Add(1)
	defer lt.txCount.Add(-1)
	return fn.Invoke()
}

func (lt *LockingTransactor) doCommute(ref *Ref, fn IFn, args ISeq) interface{} {
	if lt.txCount.Load() <= 0 {
		panic(ErrNoTransaction)
	}
	// TODO: implement for real. for now, just commute.
	ret := fn.ApplyTo(NewCons(ref.Deref(), args))

	// TODO: this is not concurrency-safe. nor is it correct for transctions.
	ref.val = ret
	return ret
}
