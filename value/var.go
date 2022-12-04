package value

type Var struct {
	ns  *Namespace
	sym *Symbol
}

func NewVar(ns *Namespace, sym *Symbol) *Var {
	return &Var{
		ns:  ns,
		sym: sym,
	}
}

func (v *Var) Namespace() *Namespace {
	return v.ns
}

func (v *Var) Symbol() *Symbol {
	return v.sym
}
