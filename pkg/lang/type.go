package lang

import (
	"bytes"
	"reflect"
	"regexp"
)

var (
	Throwable = reflect.TypeOf((interface{})(nil))

	// TODO: convert use of 'matcher' in core.glj to fit go's
	// regexps. This supresses errors but doesn't actually work.
	Matcher = reflect.TypeOf(&regexp.Regexp{})

	// TODO: rework use of PrintWriter in core.glj
	PrintWriter = reflect.TypeOf(&bytes.Buffer{})
)

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

func TypeOf(v interface{}) reflect.Type {
	return reflect.TypeOf(v)
}
