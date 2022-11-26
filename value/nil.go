package value

var (
	NilValue = NewNil()
)

// Nil is the nil value.
type Nil struct {
	Section
}

// NewNil creates a new nil value.
func NewNil(opts ...Option) *Nil {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	return &Nil{
		Section: o.section,
	}
}

func (n *Nil) String() string {
	return "nil"
}

func (n *Nil) Equal(other Value) bool {
	_, ok := other.(*Nil)
	return ok
}

func (n *Nil) GoValue() interface{} {
	return nil
}
