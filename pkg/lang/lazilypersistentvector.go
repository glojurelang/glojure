package lang

import (
	"reflect"

	"github.com/glojurelang/glojure/internal/persistent/vector"
)

// CreateOwningLazilyPersistentVector creates a persistent vector that
// owns the items in items. items must be a slice or array.
func CreateOwningLazilyPersistentVector(items any) IPersistentVector {
	if values, ok := items.([]any); ok {
		return NewVector(values...)
	}
	itemsVal := reflect.ValueOf(items)
	if itemsVal.Kind() != reflect.Slice && itemsVal.Kind() != reflect.Array {
		panic(NewIllegalArgumentError("CreateOwningLazilyPersistentVector argument must be a slice or array"))
	}
	if itemsVal.Len() == 0 {
		return emptyVector
	}
	return &Vector{vec: vector.Build(itemsVal.Len(), func(i int) any {
		return itemsVal.Index(i).Interface()
	})}
}

var vectorBufferConj = FnFunc2(func(acc, item any) any {
	values := acc.(*[]any)
	*values = append(*values, item)
	return acc
})

func CreateLazilyPersistentVector(obj any) IPersistentVector {
	switch obj := obj.(type) {
	case IReduceInit:
		result := obj.ReduceInit(vectorBufferConj, new([]any)).(*[]any)
		return NewVector((*result)...)
	case ISeq:
		slc := ToSlice(obj)
		return NewVector(slc...)
	case Seqable:
		return CreateLazilyPersistentVector(obj.Seq())
	case string:
		// Build the vector straight from the runes; going through
		// ToSlice would allocate a boxed copy of every character
		// first.
		runes := []rune(obj)
		if len(runes) == 0 {
			return emptyVector
		}
		flat := make([]any, len(runes))
		for i, r := range runes {
			flat[i] = NewChar(r)
		}
		return &Vector{
			attrs: &vectorAttrs{flat: flat},
			vec: vector.Build(len(flat), func(i int) any {
				return flat[i]
			}),
		}
	default:
		return NewVector(ToSlice(obj)...)
	}
}
