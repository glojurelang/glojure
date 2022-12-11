package value

import (
	"fmt"
	"reflect"
	"unicode"
)

// FieldOrMethod returns the field or method of the given name on the
// given value or pointer to a value. If the value is a pointer, it is
// dereferenced. If the value or pointer target is not a struct, or if
// no such field or method exists, nil is returned.
func FieldOrMethod(v interface{}, name string) interface{} {
	if unicode.IsLower(rune(name[0])) {
		// unexported field or method
		return nil
	}

	target := reflect.ValueOf(v)

	val := target.MethodByName(name)
	if val.IsValid() {
		return val.Interface()
	}

	// dereference the value if it's a pointer
	for target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	if target.Kind() != reflect.Struct {
		return nil
	}

	val = target.FieldByName(name)
	if val.IsValid() {
		return val.Interface()
	}

	return nil
}

func SetField(target interface{}, name string, val interface{}) error {
	targetVal := reflect.ValueOf(target)

	// dereference the value if it's a pointer
	for targetVal.Kind() == reflect.Ptr {
		targetVal = targetVal.Elem()
	}

	if targetVal.Kind() != reflect.Struct {
		return fmt.Errorf("cannot set field on non-struct")
	}

	field := targetVal.FieldByName(name)
	if field.IsValid() {
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", name)
		}
		goVal := reflect.ValueOf(val)
		if !goVal.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("cannot assign %s to %s", goVal.Type(), field.Type())
		}
		field.Set(goVal)
		return nil
	}

	return fmt.Errorf("no such field %s", name)
}
