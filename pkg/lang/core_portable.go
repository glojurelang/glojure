package lang

import (
	"reflect"
	"runtime"
	"time"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

func MakeArray(element any, dimensions ...any) any {
	typ, ok := ReflectType(element)
	if !ok || typ == nil || len(dimensions) == 0 {
		panic(NewIllegalArgumentError("make-array requires a type and dimensions"))
	}
	sizes := make([]int, len(dimensions))
	for i, size := range dimensions {
		sizes[i] = int(LongCast(size))
		if sizes[i] < 0 {
			panic(NewIllegalArgumentError("negative array dimension"))
		}
	}
	var build func(int) reflect.Value
	build = func(depth int) reflect.Value {
		itemType := typ
		for i := depth + 1; i < len(sizes); i++ {
			itemType = reflect.SliceOf(itemType)
		}
		result := reflect.MakeSlice(reflect.SliceOf(itemType), sizes[depth], sizes[depth])
		if depth+1 < len(sizes) {
			for i := 0; i < result.Len(); i++ {
				result.Index(i).Set(build(depth + 1))
			}
		}
		return result
	}
	return build(0).Interface()
}

func InstMillis(value any) int64 {
	switch instant := value.(type) {
	case time.Time:
		return instant.UnixMilli()
	case *time.Time:
		return instant.UnixMilli()
	case interface{ GetTime() int64 }:
		return instant.GetTime()
	default:
		panic(NewIllegalArgumentError("inst-ms requires an instant"))
	}
}

// FilePosition reports a Go call frame for clojure.test's deprecated helper.
func FilePosition(depth int64) IPersistentVector {
	_, file, line, ok := runtime.Caller(int(depth) + 1)
	if !ok {
		return NewVector(nil, nil)
	}
	return NewVector(file, int64(line))
}

type Iteration struct {
	step, some, value, key IFn
	initial                any
}

func NewIteration(step, some, value, key IFn, initial any) *Iteration {
	return &Iteration{step, some, value, key, initial}
}

func (it *Iteration) Seq() ISeq {
	var next func(any) ISeq
	next = func(result any) ISeq {
		if !BooleanCast(it.some.Invoke(result)) {
			return nil
		}
		var tail ISeq
		key := it.key.Invoke(result)
		if !IsNil(key) {
			tail = NewLazySeq(func() any { return next(it.step.Invoke(key)) })
		}
		return NewCons(it.value.Invoke(result), tail)
	}
	return next(it.step.Invoke(it.initial))
}

func (it *Iteration) ReduceInit(fn IFn, initial any) any {
	result := it.step.Invoke(it.initial)
	for BooleanCast(it.some.Invoke(result)) {
		initial = fn.Invoke(initial, it.value.Invoke(result))
		if reduced, ok := initial.(*Reduced); ok {
			return reduced.Deref()
		}
		key := it.key.Invoke(result)
		if IsNil(key) {
			return initial
		}
		result = it.step.Invoke(key)
	}
	return initial
}

func init() {
	for name, value := range map[string]any{
		"MakeArray": MakeArray, "InstMillis": InstMillis,
		"FilePosition": FilePosition, "NewIteration": NewIteration,
	} {
		pkgmap.Set("github.com/glojurelang/glojure/pkg/lang."+name, value)
	}
}
