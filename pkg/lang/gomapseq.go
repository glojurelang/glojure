package lang

import "reflect"

type (
	GoMapSeq struct {
		meta         IPersistentMap
		hash, hasheq uint32

		gm reflect.Value
	}
)

func NewGoMapSeq(gm any) *GoMapSeq {
	v := reflect.ValueOf(gm)
	if v.Kind() != reflect.Map {
		panic(NewIllegalArgumentError("argument to NewGoMapSeq must be a map"))
	}
	return &GoMapSeq{
		gm: v,
	}
}
