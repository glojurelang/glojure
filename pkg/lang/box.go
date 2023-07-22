package lang

// Box is a wrapper around a value that can be used to store values of
// different types in atomic.Value, which requires all stored values
// to have the same concrete type.
type Box struct {
	val interface{}
}

// NewBox returns a new Box that wraps the given value.
func NewBox(val interface{}) *Box {
	return &Box{val}
}
