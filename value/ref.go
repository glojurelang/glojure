package value

// Ref is a reference to a value that can be updated transactionally.
type Ref struct{}

func NewRef(val interface{}) *Ref {
	return &Ref{}
}
