package lang

import (
	"fmt"
	"reflect"
)

var (
	BuiltinTypes = map[string]reflect.Type{
		"any":        reflect.TypeOf((*interface{})(nil)).Elem(),
		"bool":       reflect.TypeOf(false),
		"uint8":      reflect.TypeOf(uint8(0)),
		"uint16":     reflect.TypeOf(uint16(0)),
		"uint32":     reflect.TypeOf(uint32(0)),
		"uint64":     reflect.TypeOf(uint64(0)),
		"int8":       reflect.TypeOf(int8(0)),
		"int16":      reflect.TypeOf(int16(0)),
		"int32":      reflect.TypeOf(int32(0)),
		"int64":      reflect.TypeOf(int64(0)),
		"float32":    reflect.TypeOf(float32(0)),
		"float64":    reflect.TypeOf(float64(0)),
		"complex64":  reflect.TypeOf(complex64(0)),
		"complex128": reflect.TypeOf(complex128(0)),
		"string":     reflect.TypeOf(""),
		"int":        reflect.TypeOf(int(0)),
		"uint":       reflect.TypeOf(uint(0)),
		"uintptr":    reflect.TypeOf(uintptr(0)),
		"byte":       reflect.TypeOf(byte(0)),
		"rune":       reflect.TypeOf(rune(0)),
		"error":      reflect.TypeOf((*error)(nil)).Elem(),
	}

	Builtins = map[string]interface{}{
		// Built-in functions — wrapped as FnFuncN where arity is fixed
		// so Apply uses the zero-allocation fast path.
		"append": FnFunc(func(args ...any) any { // variadic: keep FnFunc
			return GoAppend(args[0], args[1:]...)
		}),
		"copy": FnFunc2(func(a, b any) any {
			return GoCopy(a, b)
		}),
		"delete": FnFunc2(func(a, b any) any {
			GoDelete(a, b)
			return nil
		}),
		"len": FnFunc1(func(a any) any {
			return GoLen(a)
		}),
		"cap": FnFunc1(func(a any) any {
			return GoCap(a)
		}),
		"make": FnFunc(func(args ...any) any { // variadic: keep FnFunc
			return GoMake(args[0].(reflect.Type), args[1:]...)
		}),
		"new": FnFunc1(func(a any) any {
			return GoNew(a.(reflect.Type))
		}),
		"complex": FnFunc2(func(a, b any) any {
			return GoComplex(a, b)
		}),
		"real": FnFunc1(func(a any) any {
			return GoReal(a)
		}),
		"imag": FnFunc1(func(a any) any {
			return GoImag(a)
		}),
		"close": FnFunc1(func(a any) any {
			GoClose(a)
			return nil
		}),
		"panic": FnFunc1(func(a any) any {
			GoPanic(a)
			return nil // unreachable
		}),
		// recover can't be exposed this way, because it only works inside
		// a deferred function. instead, try/catch should be used.

		// Built-in type operators — wrapped as FnFuncN.
		"slice-of": FnFunc1(func(a any) any { // sliceof(T) -> []T
			return reflect.SliceOf(a.(reflect.Type))
		}),
		"ptr-to": FnFunc1(func(a any) any { // ptrto(T) -> *T
			return reflect.PtrTo(a.(reflect.Type))
		}),
		"chan-of": FnFunc1(func(a any) any { // chanof(T) -> chan T
			return GoChanOf(a.(reflect.Type))
		}),
		"<-chan-of": FnFunc1(func(a any) any { // recvchanof(T) -> <-chan T
			return GoRecvChanOf(a.(reflect.Type))
		}),
		"chan<--of": FnFunc1(func(a any) any { // sendchanof(T) -> chan<- T
			return GoSendChanOf(a.(reflect.Type))
		}),
		"map-of": FnFunc2(func(a, b any) any { // mapof(K, V) -> map[K]V
			return reflect.MapOf(a.(reflect.Type), b.(reflect.Type))
		}),
		"func-of": FnFunc3(func(a, b, c any) any {
			return reflect.FuncOf(a.([]reflect.Type), b.([]reflect.Type), c.(bool))
		}),
		"array-of": FnFunc2(func(a, b any) any { // arrayof(n, T) -> [n]T
			return reflect.ArrayOf(MustAsInt(a), b.(reflect.Type))
		}),

		// Built-in operators — wrapped as FnFuncN.
		"deref": FnFunc1(func(a any) any { // deref(ptr) -> val
			return GoDeref(a)
		}),
		// TODO: addr will need some special handling, because it's not a
		// function; it can only be applied to lvalues, which are not
		// first-class in clojure. we'll need a special form to take the
		// address of slice elements and struct fields.
		"index": FnFunc2(func(a, b any) any { // index(slc, i) -> val
			return GoIndex(a, b)
		}),
		"slice": FnFunc(func(args ...any) any { // variadic: keep FnFunc
			return GoSlice(args[0], args[1:]...)
		}),
		"map-index": FnFunc2(func(a, b any) any { // mapindex(m, key) -> val
			return GoMapIndex(a, b)
		}),
		"set-map-index": FnFunc3(func(a, b, c any) any { // setmapindex(m, key, val) -> m[key] = val
			GoSetMapIndex(a, b, c)
			return nil
		}),
		"send": FnFunc2(func(a, b any) any { // send(ch, val) -> ch <- val
			GoSend(a, b)
			return nil
		}),
		"recv": FnFunc1(func(a any) any { // recv(ch) -> val, ok <- ch
			val, ok := GoRecv(a)
			return NewVector(val, ok)
		}),
	}
)

func init() {
	for name, typ := range BuiltinTypes {
		Builtins[name] = typ
	}
}

func GoAppend(slc interface{}, vals ...interface{}) interface{} {
	slcVal := reflect.ValueOf(slc)
	valsVal := make([]reflect.Value, len(vals))
	for i, v := range vals {
		valsVal[i] = reflect.ValueOf(v)
	}
	return reflect.Append(slcVal, valsVal...).Interface()
}

func GoCopy(dst, src interface{}) int {
	return reflect.Copy(reflect.ValueOf(dst), reflect.ValueOf(src))
}

func GoDelete(m, key interface{}) {
	mVal := reflect.ValueOf(m)
	keyVal := reflect.ValueOf(key)
	mVal.SetMapIndex(keyVal, reflect.Value{})
}

func GoLen(v interface{}) int {
	return reflect.ValueOf(v).Len()
}

func GoCap(v interface{}) int {
	return reflect.ValueOf(v).Cap()
}

func GoMake(typ reflect.Type, args ...interface{}) interface{} {
	switch typ.Kind() {
	case reflect.Slice:
		switch len(args) {
		case 0:
			return reflect.MakeSlice(typ, 0, 0).Interface()
		case 1:
			return reflect.MakeSlice(typ, MustAsInt(args[0]), MustAsInt(args[0])).Interface()
		case 2:
			return reflect.MakeSlice(typ, MustAsInt(args[0]), MustAsInt(args[1])).Interface()
		default:
			panic(fmt.Errorf("make: invalid argument count %d", len(args)))
		}
	case reflect.Map:
		if len(args) == 0 {
			return reflect.MakeMap(typ).Interface()
		} else if len(args) == 1 {
			return reflect.MakeMapWithSize(typ, MustAsInt(args[0])).Interface()
		} else {
			panic(fmt.Errorf("make: invalid argument count %d", len(args)))
		}
	case reflect.Chan:
		if len(args) == 0 {
			return reflect.MakeChan(typ, 0).Interface()
		} else if len(args) == 1 {
			return reflect.MakeChan(typ, MustAsInt(args[0])).Interface()
		} else {
			panic(fmt.Errorf("make: invalid argument count %d", len(args)))
		}
	default:
		panic(fmt.Errorf("make: invalid type %s", typ))
	}
}

func GoNew(typ reflect.Type) interface{} {
	return reflect.New(typ).Interface()
}

func GoComplex(real, imag interface{}) interface{} {
	switch real.(type) {
	case float32:
		return complex(real.(float32), imag.(float32))
	case float64:
		return complex(real.(float64), imag.(float64))
	default:
		panic(fmt.Errorf("complex: invalid type %s", reflect.TypeOf(real)))
	}
}

func GoReal(c interface{}) interface{} {
	switch c.(type) {
	case complex64:
		return real(c.(complex64))
	case complex128:
		return real(c.(complex128))
	default:
		panic(fmt.Errorf("real: invalid type %s", reflect.TypeOf(c)))
	}
}

func GoImag(c interface{}) interface{} {
	switch c.(type) {
	case complex64:
		return imag(c.(complex64))
	case complex128:
		return imag(c.(complex128))
	default:
		panic(fmt.Errorf("imag: invalid type %s", reflect.TypeOf(c)))
	}
}

func GoClose(c interface{}) {
	reflect.ValueOf(c).Close()
}

func GoPanic(v interface{}) {
	panic(v)
}

func GoDeref(ptr interface{}) interface{} {
	return reflect.Indirect(reflect.ValueOf(ptr)).Interface()
}

func GoIndex(slc, i interface{}) interface{} {
	return reflect.ValueOf(slc).Index(MustAsInt(i)).Interface()
}

func GoMapIndex(m, k interface{}) interface{} {
	return reflect.ValueOf(m).MapIndex(reflect.ValueOf(k)).Interface()
}

func GoSetMapIndex(m, k, v interface{}) {
	reflect.ValueOf(m).SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
}

func GoSlice(slc interface{}, indices ...interface{}) interface{} {
	if len(indices) == 0 {
		panic(fmt.Errorf("slice: no indices"))
	}
	if len(indices) > 2 {
		panic(fmt.Errorf("slice: too many indices %d", len(indices)))
	}

	// Fast path for strings — avoid reflect overhead.
	if s, ok := slc.(string); ok {
		i := 0
		if !IsNil(indices[0]) {
			i = MustAsInt(indices[0])
		}
		if len(indices) == 2 {
			j := len(s)
			if !IsNil(indices[1]) {
				j = MustAsInt(indices[1])
			}
			return s[i:j]
		}
		return s[i:]
	}

	slcVal := reflect.ValueOf(slc)
	i := 0
	j := slcVal.Len()
	if !IsNil(indices[0]) {
		i = MustAsInt(indices[0])
	}
	if len(indices) == 2 {
		if !IsNil(indices[1]) {
			j = MustAsInt(indices[1])
		}
	}
	return slcVal.Slice(i, j).Interface()
}

func GoChanOf(typ reflect.Type) reflect.Type {
	return reflect.ChanOf(reflect.BothDir, typ)
}

func GoRecvChanOf(typ reflect.Type) reflect.Type {
	return reflect.ChanOf(reflect.RecvDir, typ)
}

func GoSendChanOf(typ reflect.Type) reflect.Type {
	return reflect.ChanOf(reflect.SendDir, typ)
}

func GoSend(ch, val interface{}) {
	chVal := reflect.ValueOf(ch)
	// handle untyped nil
	valVal := reflect.ValueOf(val)

	if !valVal.IsValid() {
		valVal = reflect.Zero(chVal.Type().Elem())
		if !valVal.IsNil() {
			panic(fmt.Errorf("send: invalid value %v for chan of type %T", val, ch))
		}
	}

	chVal.Send(valVal)
}

func GoRecv(ch interface{}) (interface{}, bool) {
	val, ok := reflect.ValueOf(ch).Recv()
	return val.Interface(), ok
}
