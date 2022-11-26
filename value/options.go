package value

type options struct {
	// where the value was defined
	section Section
}

// Option represents an option that can be passed to Value
// constructors.
type Option func(*options)

// WithSection returns an Option that sets the section of the value.
func WithSection(s Section) Option {
	return func(o *options) {
		o.section = s
	}
}

type Pos struct {
	Filename string
	Line     int
	Column   int
}

func (p Pos) Valid() bool {
	return p.Line != 0 && p.Column != 0
}

type Section struct {
	StartPos, EndPos Pos
	// TODO: consider adding information about whitespace and comments.
}

func (p Section) Pos() Pos { return p.StartPos }
func (p Section) End() Pos { return p.EndPos }
