package value

// TODO: nix this file. If we keep this information, it should be in
// metadata.

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
}

func (p Section) Pos() Pos { return p.StartPos }
func (p Section) End() Pos { return p.EndPos }
