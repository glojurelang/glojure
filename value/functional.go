package value

// Reduce applies the given function to each element of the ISeq.
func Reduce(f func(interface{}, interface{}) interface{}, seq ISeq) interface{} {
	if seq.IsEmpty() {
		panic("reduce of empty sequence without initial value")
	}
	return ReduceInit(f, seq.First(), seq.Rest())
}

func ReduceInit(f func(interface{}, interface{}) interface{}, init interface{}, seq ISeq) interface{} {
	var res interface{} = init
	for !seq.IsEmpty() {
		res = f(res, seq.First())
		seq = seq.Rest()
	}
	return res
}
