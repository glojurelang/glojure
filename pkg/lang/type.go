package lang

import "reflect"

func HasType(t reflect.Type, v interface{}) bool {
	if v == nil {
		return false
	}
	vType := reflect.TypeOf(v)
	switch {
	// TODO: should this be AssignableTo?
	case vType == t, vType.ConvertibleTo(t), vType.Kind() == reflect.Pointer && vType.Elem().ConvertibleTo(t):
		return true
	default:
		return false
	}
}
