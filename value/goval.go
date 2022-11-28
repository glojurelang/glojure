package value

import (
	"fmt"
	"reflect"
)

// GoVal is a Value that wraps a Go value.
type GoVal struct {
	val reflect.Value
}

// NewGoVal creates a new GoVal from a Go host value.
func NewGoVal(val interface{}) *GoVal {
	return &GoVal{val: reflect.ValueOf(val)}
}

func (gv *GoVal) Pos() Pos {
	return Pos{}
}

func (gv *GoVal) End() Pos {
	return Pos{}
}

func (gv *GoVal) String() string {
	// TODO: what's the best way to render a Go value as a string?
	// if strer, ok := gv.val.Interface().(fmt.Stringer); ok {
	// 	return strer.String()
	// }
	return gv.val.String()
}

func (gv *GoVal) Equal(other interface{}) bool {
	ogv, ok := other.(*GoVal)
	if !ok {
		return false
	}
	return gv.val.Interface() == ogv.val.Interface()
}

func (gv *GoVal) GoValue() interface{} {
	return gv.val.Interface()
}

func (gv *GoVal) Value() Value {
	return fromGo(gv.val.Interface())
}

func (gv *GoVal) FieldOrMethod(name string) *GoVal {
	val := gv.val.MethodByName(name)
	if val.IsValid() {
		return &GoVal{val: val}
	}

	// dereference the value if it's a pointer
	for gv.val.Kind() == reflect.Ptr {
		gv = &GoVal{val: gv.val.Elem()}
	}

	if gv.val.Kind() != reflect.Struct {
		return nil
	}

	val = gv.val.FieldByName(name)
	if val.IsValid() {
		return &GoVal{val: val}
	}

	return nil
}

func (gv *GoVal) SetField(name string, val interface{}) error {
	// dereference the value if it's a pointer
	for gv.val.Kind() == reflect.Ptr {
		gv = &GoVal{val: gv.val.Elem()}
	}

	if gv.val.Kind() != reflect.Struct {
		return fmt.Errorf("cannot set field on non-struct")
	}

	field := gv.val.FieldByName(name)
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

func (gv *GoVal) Apply(env Environment, args []Value) (Value, error) {
	gvKind := gv.val.Kind()
	gvType := gv.val.Type()

	if gvKind != reflect.Func {
		return nil, fmt.Errorf("cannot apply non-function")
	}
	if gvType.NumIn() != len(args) && !gvType.IsVariadic() {
		return nil, fmt.Errorf("wrong number of arguments: expected %d, got %d", gvType.NumIn(), len(args))
	}

	var goArgs []reflect.Value
	for i := 0; i < len(args); i++ {
		if i >= gvType.NumIn() && !gvType.IsVariadic() {
			panic(fmt.Sprintf("too many arguments: expected %d, got %d", gvType.NumIn(), len(args)))
		}

		var targetType reflect.Type
		if i < gvType.NumIn()-1 || !gvType.IsVariadic() {
			targetType = gvType.In(i)
		} else {
			targetType = gvType.In(gvType.NumIn() - 1).Elem()
		}

		argGoVal, err := ConvertToGo(env, targetType, args[i])
		if err != nil {
			return nil, fmt.Errorf("argument %d: %s", i, err)
		}
		goArgs = append(goArgs, reflect.ValueOf(argGoVal))
	}

	goRes := gv.val.Call(goArgs)
	res := make([]Value, len(goRes))
	for i, goVal := range goRes {
		switch resVal := goVal.Interface().(type) {
		case Value:
			res[i] = resVal
		case nil:
			res[i] = nil
		case bool:
			res[i] = resVal
		default:
			// ?? this got a server test working, but we now fail tests next
			// up, figure out the desired semantics and make them work!
			res[i] = NewGoVal(resVal)
			//res[i] = fromGo(goVal.Interface()) // TODO: define conversion semantics
		}
	}
	if len(res) == 0 {
		return nil, nil
	}
	if len(res) == 1 {
		return res[0], nil
	}
	return NewVector(res), nil
}

// ConvertToGo converts a Value to a Go value of the given type, if
// possible.
func ConvertToGo(env Environment, typ reflect.Type, val Value) (interface{}, error) {
	var goVal interface{}
	switch val := val.(type) {
	case GoValuer:
		goVal = val.GoValue()
	case Applyer:
		goVal = reflect.MakeFunc(typ, reflectFuncFromApplyer(env, val)).Interface()
	default:
		goVal = val
	}

	return coerceGoValue(typ, goVal)
}

// coerceGoValue attempts to coerce a Go value to be assignable to a
// target type. If the value is already assignable, it is returned.
func coerceGoValue(targetType reflect.Type, val interface{}) (interface{}, error) {
	if val == nil {
		if !isNilableKind(targetType.Kind()) {
			return nil, fmt.Errorf("cannot assign nil to non-nilable type %s", targetType)
		}
		return reflect.Zero(targetType).Interface(), nil
	}

	if reflect.TypeOf(val).AssignableTo(targetType) {
		return val, nil
	}
	switch targetType.Kind() {
	case reflect.Slice:
		if reflect.TypeOf(val).Kind() == reflect.String {
			// convert string to []byte
			val = []byte(val.(string))
		}

		if reflect.TypeOf(val).Kind() != reflect.Slice {
			return nil, fmt.Errorf("cannot coerce %s to %s", reflect.TypeOf(val), targetType)
		}
		// use reflect.MakeSlice to create a new slice of the target type
		// and copy the values into it
		targetSlice := reflect.MakeSlice(targetType, reflect.ValueOf(val).Len(), reflect.ValueOf(val).Len())
		for i := 0; i < reflect.ValueOf(val).Len(); i++ {
			// try to coerce each element of the slice
			coerced, err := coerceGoValue(targetType.Elem(), reflect.ValueOf(val).Index(i).Interface())
			if err != nil {
				return nil, err
			}
			targetSlice.Index(i).Set(reflect.ValueOf(coerced))
		}
		return targetSlice.Interface(), nil
	default:
		if reflect.TypeOf(val).ConvertibleTo(targetType) {
			return reflect.ValueOf(val).Convert(targetType).Interface(), nil
		}
		return nil, fmt.Errorf("cannot coerce %s to %s", reflect.TypeOf(val), targetType)
	}
}

func ConvertFromGo(val interface{}) Value {
	return fromGo(val)
}

func fromGo(val interface{}) Value {
	if gv, ok := val.(*GoVal); ok {
		val = gv.val.Interface()
	}
	// convert the Go value to a Glojure value
	// - integral values are converted to floats
	// - strings are converted to strings
	// - slices are converted to vectors
	// - anything else is converted to a GoVal
	// TODO: don't do this... let the user decide what to do with the Go value
	switch val := val.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return int64(val)
	case uint:
		return int64(val)
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		return int64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	case string:
		return val
	case bool:
		return val
	case nil:
		return nil
	}

	// TODO: support all collection types
	if reflect.TypeOf(val).Kind() == reflect.Slice {
		var vec []Value
		for i := 0; i < reflect.ValueOf(val).Len(); i++ {
			vec = append(vec, fromGo(reflect.ValueOf(val).Index(i).Interface()))
		}
		return NewVector(vec)
	}
	if v, ok := val.(Value); ok {
		return v
	}
	return NewGoVal(val)
}

func isNilableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}

func reflectFuncFromApplyer(env Environment, applyer Applyer) func(args []reflect.Value) []reflect.Value {
	return func(args []reflect.Value) []reflect.Value {
		var glojureArgs []Value
		for _, arg := range args {
			glojureArgs = append(glojureArgs, fromGo(arg.Interface()))
		}
		res, err := applyer.Apply(env, glojureArgs)
		if err != nil {
			panic(err)
		}
		if res == nil || Equal(res, nil) {
			return nil
		}

		if goValuerRes, ok := res.(GoValuer); ok {
			res = goValuerRes.GoValue()
		}
		return []reflect.Value{reflect.ValueOf(res)}
	}
}
