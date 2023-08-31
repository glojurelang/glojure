package lang

type (
	AMapEntry interface {
		APersistentVector

		IMapEntry
	}
)

func amapentryNth(a AMapEntry, i int) any {
	if i == 0 {
		return a.Key()
	}
	if i == 1 {
		return a.Val()
	}
	panic(NewIndexOutOfBoundsError())
}

func amapentryNthDefault(a AMapEntry, i int, notFound any) any {
	if i == 0 {
		return a.Key()
	}
	if i == 1 {
		return a.Val()
	}
	return notFound
}

func amapentryAssocN(a AMapEntry, i int, val any) IPersistentVector {
	return amapentryAsVector(a).AssocN(i, val)
}

func amapentryCount(a AMapEntry) int {
	return 2
}

func amapentryAsVector(a AMapEntry) IPersistentVector {
	return NewVector(a.Key(), a.Val())
}

func amapentrySeq(a AMapEntry) ISeq {
	return amapentryAsVector(a).Seq()
}

func amapentryCons(a AMapEntry, o any) IPersistentCollection {
	return amapentryAsVector(a).Cons(o)
}

func amapentryEmpty(a AMapEntry) IPersistentCollection {
	return nil
}

func amapentryPop(a AMapEntry) IPersistentCollection {
	return NewVector(a.Key())
}
