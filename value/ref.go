package value

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
	return nil
}

func (r *Ref) Commute(fn IFn, args ISeq) interface{} {
	return fn.ApplyTo(NewCons(r.Deref(), args))
}

type LockingTransactor struct{}

var (
	LockingTransaction = &LockingTransactor{}
)

func (lt *LockingTransactor) RunInTransaction(fn IFn) interface{} {
	return fn.Invoke()
}
