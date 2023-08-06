package lang

import (
	"fmt"
	"reflect"
)

var (
	Builtins = map[string]interface{}{
		// Built-in types
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

		// Built-in functions
		"append":  GoAppend,
		"copy":    GoCopy,
		"delete":  GoDelete,
		"len":     GoLen,
		"cap":     GoCap,
		"make":    GoMake,
		"new":     GoNew,
		"complex": GoComplex,
		"real":    GoReal,
		"imag":    GoImag,
		"close":   GoClose,
		"panic":   GoPanic,
		// recover can't be exposed this way, because it only works inside
		// a deferred function. instead, try/catch should be used.

		// Built-in type operators
		"slice-of": reflect.SliceOf, // sliceof(T) -> []T
		"ptr-to":   reflect.PtrTo,   // ptrto(T) -> *T
		"chan-of":  reflect.ChanOf,  // chanof(dir, T) -> chan T
		"map-of":   reflect.MapOf,   // mapof(K, V) -> map[K]V
		"func-of":  reflect.FuncOf,
		"array-of": reflect.ArrayOf, // arrayof(n, T) -> [n]T

		// Built-in operators
		"deref": GoDeref, // deref(ptr) -> val
		// TODO: addr will need some special handling, because it's not a
		// function; it can only be applied to lvalues, which are not
		// first-class in clojure. we'll need a special form to take the
		// address of slice elements and struct fields.
		"index": GoIndex, // index(slc, i) -> val
		"slice": GoSlice, // slice(slc, i, j) -> slc[i:j]
		"send":  GoSend,  // send(ch, val) -> ch <- val
		"recv":  GoRecv,  // recv(ch) -> val, ok <- ch
	}
)

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

func GoIndex(slc interface{}, i interface{}) interface{} {
	return reflect.ValueOf(slc).Index(MustAsInt(i)).Interface()
}

func GoSlice(slc interface{}, indices ...interface{}) interface{} {
	slcVal := reflect.ValueOf(slc)
	i := 0
	j := slcVal.Len()

	if len(indices) > 2 {
		panic(fmt.Errorf("slice: too many indices %d", len(indices)))
	}
	if len(indices) == 0 {
		panic(fmt.Errorf("slice: no indices"))
	}
	if len(indices) == 1 {
		if !IsNil(indices[0]) {
			i = MustAsInt(indices[0])
		}
	}
	if len(indices) == 2 {
		if !IsNil(indices[1]) {
			j = MustAsInt(indices[1])
		}
	}
	return slcVal.Slice(i, j).Interface()
}

func GoSend(ch, val interface{}) {
	reflect.ValueOf(ch).Send(reflect.ValueOf(val))
}

func GoRecv(ch interface{}) (interface{}, bool) {
	val, ok := reflect.ValueOf(ch).Recv()
	return val.Interface(), ok
}
