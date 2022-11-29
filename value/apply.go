package value

import (
	"fmt"
	"reflect"
)

func Apply(env Environment, fn interface{}, args []interface{}) (interface{}, error) {
	if applyer, ok := fn.(Applyer); ok {
		return applyer.Apply(env, args)
	}

	if rt, ok := fn.(reflect.Type); ok {
		return applyType(env, rt, args)
	}

	goVal := reflect.ValueOf(fn)

	gvKind := goVal.Kind()
	gvType := goVal.Type()

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

	reflectRes := goVal.Call(goArgs)
	res := make([]interface{}, len(reflectRes))
	for i, val := range reflectRes {
		res[i] = val.Interface()
	}
	if len(res) == 0 {
		return nil, nil
	}
	if len(res) == 1 {
		return res[0], nil
	}
	return NewVector(res), nil
}

func applyType(env Environment, typ reflect.Type, args []interface{}) (interface{}, error) {
	if len(args) == 0 {
		return reflect.Zero(typ).Interface(), nil
	}

	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments")
	}

	arg := args[0]
	if arg, ok := arg.(GoValuer); ok {
		val := reflect.ValueOf(arg.GoValue())
		if val.Type().ConvertibleTo(typ) {
			return val.Convert(typ).Interface(), nil
		}
	}
	res, err := ConvertToGo(env, typ, arg)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// TODO: reconsider the semantics of the functions below.

// ConvertToGo converts a Value to a Go value of the given type, if
// possible.
func ConvertToGo(env Environment, typ reflect.Type, val interface{}) (interface{}, error) {
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

func ConvertFromGo(val interface{}) interface{} {
	return fromGo(val)
}

func fromGo(val interface{}) interface{} {
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
		var vec []interface{}
		for i := 0; i < reflect.ValueOf(val).Len(); i++ {
			vec = append(vec, fromGo(reflect.ValueOf(val).Index(i).Interface()))
		}
		return NewVector(vec)
	}
	if v, ok := val.(interface{}); ok {
		return v
	}
	return val
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
		var glojureArgs []interface{}
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
