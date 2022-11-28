package value

import "strings"

// Seq is a lazy sequence of values.
type Seq struct {
	Section
	Enumerable
}

func (s *Seq) Equal(v interface{}) bool {
	other, ok := v.(*Seq)
	if !ok {
		return false
	}
	e1, cancel1 := s.Enumerate()
	defer cancel1()
	e2, cancel2 := other.Enumerate()
	defer cancel2()
	for {
		v1, ok1 := <-e1
		v2, ok2 := <-e2
		if ok1 != ok2 {
			return false
		}
		if !ok1 {
			return true
		}
		if !Equal(v1, v2) {
			return false
		}
	}
	return true
}

func (s *Seq) Pos() Pos {
	return Pos{}
}

func (s *Seq) String() string {
	b := strings.Builder{}
	b.WriteString("(")
	e, cancel := s.Enumerate()
	defer cancel()
	first := true
	for {
		v, ok := <-e
		if !ok {
			break
		}
		if !first {
			b.WriteString(" ")
		}
		first = false
		b.WriteString(ToString(v))
	}
	b.WriteString(")")
	return b.String()
}
