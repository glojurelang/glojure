package value

import (
	"fmt"
	"reflect"
	"runtime/debug"
)

func Apply(env Environment, fn interface{}, args []interface{}) (_ interface{}, err error) {
	if applyer, ok := fn.(Applyer); ok {
		return applyer.Apply(env, args)
	}

	if rt, ok := fn.(reflect.Type); ok {
		return applyType(env, rt, args)
	}

	if fn == nil {
		return nil, fmt.Errorf("cannot apply nil")
	}

	goVal := reflect.ValueOf(fn)

	gvKind := goVal.Kind()
	gvType := goVal.Type()

	if gvKind == reflect.Slice {
		return applySlice(goVal, args)
	}

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

		argGoVal, err := coerceGoValue(env, targetType, args[i])
		if err != nil {
			return nil, fmt.Errorf("argument %d: %s", i, err)
		}
		goArgs = append(goArgs, argGoVal)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v. Stack:\n%s", r, string(debug.Stack()))
		}
	}()

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
	res, err := ConvertToGo(env, typ, arg)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func applySlice(goVal reflect.Value, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for slice: expected 1, got %d", len(args))
	}
	idx, ok := AsInt(args[0])
	if !ok {
		return nil, fmt.Errorf("slice index must be an integer")
	}
	if idx < 0 || idx >= goVal.Len() {
		return nil, fmt.Errorf("slice index out of bounds")
	}
	return goVal.Index(idx).Interface(), nil
}

// TODO: reconsider the semantics of the functions below.

// ConvertToGo converts a Value to a Go value of the given type, if
// possible.
func ConvertToGo(env Environment, typ reflect.Type, val interface{}) (interface{}, error) {
	rval, err := coerceGoValue(env, typ, val)
	if err != nil {
		return nil, err
	}
	return rval.Interface(), nil
}

// coerceGoValue attempts to coerce a Go value to be assignable to a
// target type. If the value is already assignable, it is returned.
func coerceGoValue(env Environment, targetType reflect.Type, val interface{}) (reflect.Value, error) {
	if val == nil {
		if !isNilableKind(targetType.Kind()) {
			return reflect.Value{}, fmt.Errorf("cannot assign nil to non-nilable type %s", targetType)
		}
		return reflect.Zero(targetType), nil
	}

	if reflect.TypeOf(val).AssignableTo(targetType) {
		return reflect.ValueOf(val), nil
	}
	switch targetType.Kind() {
	case reflect.Slice:
		if reflect.TypeOf(val).Kind() == reflect.String {
			// convert string to []byte
			val = []byte(val.(string))
		}
		if goValuer, ok := val.(GoValuer); ok {
			val = goValuer.GoValue()
		}
		if iseq, ok := val.(ISeq); ok {
			var slc []interface{}
			for iseq = Seq(iseq); iseq != nil; iseq = iseq.Next() {
				slc = append(slc, iseq.First())
			}
			val = slc
		}

		if reflect.TypeOf(val).Kind() != reflect.Slice {
			return reflect.Value{}, fmt.Errorf("cannot coerce %s to %s", reflect.TypeOf(val), targetType)
		}
		// use reflect.MakeSlice to create a new slice of the target type
		// and copy the values into it
		targetSlice := reflect.MakeSlice(targetType, reflect.ValueOf(val).Len(), reflect.ValueOf(val).Len())
		sourceSlice := reflect.ValueOf(val)
		for i := 0; i < sourceSlice.Len(); i++ {
			// try to coerce each element of the slice
			coerced, err := coerceGoValue(env, targetType.Elem(), sourceSlice.Index(i).Interface())
			if err != nil {
				return reflect.Value{}, err
			}
			targetSlice.Index(i).Set(coerced)
		}
		return targetSlice, nil
	case reflect.Func:
		if applyer, ok := val.(Applyer); ok {
			val := reflect.MakeFunc(targetType, reflectFuncFromApplyer(env, targetType, applyer))
			if val.Type().AssignableTo(targetType) {
				return val, nil
			}
		}
	default:
		iseqType := reflect.TypeOf((*ISeq)(nil)).Elem()
		if targetType == iseqType {
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				val = NewSliceIterator(val)
			} else if seqable, ok := val.(ISeqable); ok {
				val = seqable.Seq()
			}
		}

		val := reflect.ValueOf(val)
		if val.Type().ConvertibleTo(targetType) {
			return val.Convert(targetType), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("cannot coerce %s to %s", reflect.TypeOf(val), targetType)
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

func reflectFuncFromApplyer(env Environment, targetType reflect.Type, applyer Applyer) func(args []reflect.Value) []reflect.Value {
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
			// if target type has no return values, return nil
			if targetType.NumOut() == 0 {
				return nil
			}
			// if target type has return values, return zero values
			zeroValues := make([]reflect.Value, targetType.NumOut())
			for i := 0; i < targetType.NumOut(); i++ {
				zeroValues[i] = reflect.Zero(targetType.Out(i))
			}
			return zeroValues
		}

		if goValuerRes, ok := res.(GoValuer); ok {
			res = goValuerRes.GoValue()
		}
		return []reflect.Value{reflect.ValueOf(res)}
	}
}
