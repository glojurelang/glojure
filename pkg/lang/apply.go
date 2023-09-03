package lang

import (
	"fmt"
	"reflect"
)

func Apply(fn interface{}, args []interface{}) interface{} {
	if applyer, ok := fn.(IFn); ok {
		return applyer.Invoke(args...)
	}

	if rt, ok := fn.(reflect.Type); ok {
		return applyType(rt, args)
	}

	if fn == nil {
		panic(fmt.Errorf("cannot call nil"))
	}

	goVal := reflect.ValueOf(fn)

	gvKind := goVal.Kind()
	gvType := goVal.Type()

	if gvKind == reflect.Slice {
		return applySlice(goVal, args)
	}

	if gvKind != reflect.Func {
		panic(fmt.Errorf("cannot apply non-function %T", fn))
	}
	if gvType.NumIn() != len(args) && !gvType.IsVariadic() {
		panic(fmt.Errorf("wrong number of arguments: expected %d, got %d", gvType.NumIn(), len(args)))
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

		argGoVal, err := coerceGoValue(targetType, args[i])
		if err != nil {
			panic(fmt.Errorf("argument %d: %s", i, err))
		}
		goArgs = append(goArgs, argGoVal)
	}

	reflectRes := goVal.Call(goArgs)
	res := make([]interface{}, len(reflectRes))
	for i, val := range reflectRes {
		res[i] = val.Interface()
	}
	if len(res) == 0 {
		return nil
	}
	if len(res) == 1 {
		return res[0]
	}
	return NewVector(res...)
}

func applyType(typ reflect.Type, args []interface{}) interface{} {
	if len(args) == 0 {
		return reflect.Zero(typ).Interface()
	}

	if len(args) > 1 {
		panic(NewIllegalArgumentError("too many arguments to type conversion"))
	}

	arg := args[0]
	argVal := reflect.ValueOf(arg)
	if !argVal.CanConvert(typ) {
		panic(NewIllegalArgumentError(fmt.Sprintf("cannot convert %T to %s", arg, typ)))
	}
	return argVal.Convert(typ).Interface()
}

func applySlice(goVal reflect.Value, args []interface{}) interface{} {
	if len(args) != 1 {
		panic(fmt.Errorf("wrong number of arguments for slice: expected 1, got %d", len(args)))
	}
	idx, ok := AsInt(args[0])
	if !ok {
		panic(fmt.Errorf("slice index must be an integer"))
	}
	if idx < 0 || idx >= goVal.Len() {
		panic(fmt.Errorf("slice index out of bounds"))
	}
	return goVal.Index(idx).Interface()
}

// coerceGoValue attempts to coerce a Go value to be assignable to a
// target type. If the value is already assignable, it is returned.
// TODO: reconsider semantics of this function
func coerceGoValue(targetType reflect.Type, val interface{}) (reflect.Value, error) {
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
			coerced, err := coerceGoValue(targetType.Elem(), sourceSlice.Index(i).Interface())
			if err != nil {
				return reflect.Value{}, err
			}
			targetSlice.Index(i).Set(coerced)
		}
		return targetSlice, nil
	case reflect.Func:
		if applyer, ok := val.(IFn); ok {
			val := reflect.MakeFunc(targetType, reflectFuncFromIFn(targetType, applyer))
			if val.Type().AssignableTo(targetType) {
				return val, nil
			}
		}
	default:
		iseqType := reflect.TypeOf((*ISeq)(nil)).Elem()
		if targetType == iseqType {
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				val = NewSliceSeq(val)
			} else if seqable, ok := val.(Seqable); ok {
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

func isNilableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}

func reflectFuncFromIFn(targetType reflect.Type, applyer IFn) func(args []reflect.Value) []reflect.Value {
	return func(args []reflect.Value) []reflect.Value {
		var glojureArgs []interface{}
		for _, arg := range args {
			glojureArgs = append(glojureArgs, arg.Interface())
		}
		res := applyer.Invoke(glojureArgs...)
		if IsNil(res) {
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

		if targetType.NumOut() == 0 {
			return nil
		}

		ret := make([]reflect.Value, targetType.NumOut())
		if targetType.NumOut() == 1 {
			// convert return value to expected type of target type
			coerced, err := coerceGoValue(targetType.Out(0), res)
			if err != nil {
				panic(err)
			}
			ret[0] = coerced
		} else {
			i := 0
			for s := Seq(res); s != nil && i < len(ret); s = s.Next() {
				coerced, err := coerceGoValue(targetType.Out(i), s.First())
				if err != nil {
					panic(err)
				}
				ret[i] = coerced
				i++
			}
			for ; i < len(ret); i++ {
				ret[i] = reflect.Zero(targetType.Out(i))
			}
		}

		return ret
	}
}
