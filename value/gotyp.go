package value

import (
	"fmt"
	"reflect"
)

// GoTyp is a Value that represents a Go type.
type GoTyp struct {
	typ reflect.Type
}

// NewGoTyp creates a new GoTyp.
func NewGoTyp(typ reflect.Type) *GoTyp {
	return &GoTyp{typ: typ}
}

func (gt *GoTyp) Pos() Pos {
	return Pos{}
}

func (gt *GoTyp) End() Pos {
	return Pos{}
}

func (gt *GoTyp) String() string {
	return gt.typ.String()
}

func (gt *GoTyp) Equal(other interface{}) bool {
	if other, ok := other.(*GoTyp); ok {
		return gt.typ == other.typ
	}
	return false
}

func (gt *GoTyp) GoValue() interface{} {
	return gt.typ
}

func (gt *GoTyp) New() reflect.Value {
	return reflect.New(gt.typ)
}

func (gt *GoTyp) Type() reflect.Type {
	return gt.typ
}

// Apply returns a *GoVal of the GoTyp's type. When called with no
// arguments, it returns a zero value of the type. When called with
// one argument, it attempts to convert the argument to the type. If
// the conversion fails, it returns an error. If called with more than
// one argument, it returns an error.
func (gt *GoTyp) Apply(env Environment, args []Value) (Value, error) {
	if len(args) == 0 {
		return NewGoVal(reflect.Zero(gt.typ).Interface()), nil
	}

	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments")
	}

	arg := args[0]
	if arg, ok := arg.(GoValuer); ok {
		val := reflect.ValueOf(arg.GoValue())
		if val.Type().ConvertibleTo(gt.typ) {
			return NewGoVal(val.Convert(gt.typ).Interface()), nil
		}
	}
	res, err := ConvertToGo(env, gt.typ, arg)
	if err != nil {
		return nil, err
	}
	return NewGoVal(res), nil
}
