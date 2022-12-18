package value

type MultiFn struct {
}

func NewMultiFn(name string, dispatchFn Applyer, defaultDispatchVal interface{}, hierarchy IRef) *MultiFn {
	return &MultiFn{}
}
