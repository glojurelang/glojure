package lang

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

// Set replaces the value held by this hosted Ref. Transactional callers still
// use Commute; JVM compatibility adapters use Set when they implement mutable
// host objects whose state is exposed through IDeref.
func (r *Ref) Set(value interface{}) interface{} {
	r.val = value
	return value
}

func (r *Ref) Commute(fn IFn, args ISeq) interface{} {
	return LockingTransaction.doCommute(r, fn, args)
}

// Alter applies fn to the in-transaction value. The hosted transaction model
// is currently serialized, so it shares the same implementation as Commute.
func (r *Ref) Alter(fn IFn, args ISeq) interface{} {
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
