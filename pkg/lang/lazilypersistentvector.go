package lang

func CreateOwningLazilyPersistentVector(items ...any) IPersistentVector {
	if len(items) <= 32 {
		return NewVector(items...)
	}
	return NewVector(items...)
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
