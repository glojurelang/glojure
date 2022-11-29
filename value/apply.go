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
