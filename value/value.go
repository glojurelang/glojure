package value

// GoValuer is an interface for values that can be converted to a Go
// value.
type GoValuer interface {
	GoValue() interface{}
}
