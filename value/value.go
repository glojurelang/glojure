package value

// Conjer is an interface for values that can be conjed onto.
type Conjer interface {
	Conj(...interface{}) Conjer
}

// Counter is an interface for compound values whose elements can be
// counted.
type Counter interface {
	Count() int
}

// GoValuer is an interface for values that can be converted to a Go
// value.
type GoValuer interface {
	GoValue() interface{}
}
