package lang

// Volatile is a mutable cell for single threaded use, matching Clojure's
// volatile! which offers no coordination between threads beyond
// visibility. Callers needing synchronization should use an atom.
type Volatile struct {
	val interface{}
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
	return v.val
}

func (v *Volatile) Reset(val interface{}) interface{} {
	v.val = val
	return val
}
