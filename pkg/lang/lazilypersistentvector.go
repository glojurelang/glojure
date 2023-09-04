package lang

import "reflect"

// CreateOwningLazilyPersistentVector creates a persistent vector that
// owns the items in items. items must be a slice or array.
func CreateOwningLazilyPersistentVector(items any) IPersistentVector {
	itemsVal := reflect.ValueOf(items)
	if itemsVal.Kind() != reflect.Slice && itemsVal.Kind() != reflect.Array {
		panic(NewIllegalArgumentError("CreateOwningLazilyPersistentVector argument must be a slice or array"))
	}
	// TODO: optimize by building the tree directly within a PersistentVector.
	args := make([]any, itemsVal.Len())
	for i := 0; i < itemsVal.Len(); i++ {
		args[i] = itemsVal.Index(i).Interface()
	}
	return NewVector(args...)
}

func CreateLazilyPersistentVector(obj any) IPersistentVector {
	switch obj := obj.(type) {
	case IReduceInit:
		return obj.ReduceInit(IFnFunc(func(args ...any) any {
			acc, item := args[0], args[1]
			return acc.(IPersistentVector).Cons(item)
		}), emptyVector).(IPersistentVector)
	case ISeq:
		// TODO: optimize for ISeq by building the tree directly with a
		// PersistentVector implementaiton
		slc := seqToSlice(obj)
		return NewVector(slc...)
	case Seqable:
		return CreateLazilyPersistentVector(obj.Seq())
	default:
		return NewVector(ToSlice(obj)...)
	}
}
