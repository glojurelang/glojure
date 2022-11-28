package value

var (
	True  = &Bool{Value: true}
	False = &Bool{Value: false}
)

// Bool is a boolean value.
type Bool struct {
	Section
	Value bool
}

func NewBool(b bool, opts ...Option) *Bool {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.section == (Section{}) {
		if b {
			return True
		}
		return False
	}
	return &Bool{
		Section: o.section,
		Value:   b,
	}
}

func (b *Bool) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

func (b *Bool) Equal(v interface{}) bool {
	other, ok := v.(*Bool)
	if !ok {
		return false
	}
	return b.Value == other.Value
}

func (b *Bool) GoValue() interface{} {
	return b.Value
}
