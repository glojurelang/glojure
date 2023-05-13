package value

type Reduced struct {
	val interface{}
}

var (
	_ IDeref = (*Reduced)(nil)
)

func IsReduced(v interface{}) bool {
	_, ok := v.(*Reduced)
	return ok
}

func NewReduced(v interface{}) *Reduced {
	return &Reduced{val: v}
}

func (r *Reduced) Deref() interface{} {
	return r.val
}
