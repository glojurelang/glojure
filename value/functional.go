package value

// Reduce applies the given function to each element of the ISeq.
func Reduce(f func(interface{}, interface{}) interface{}, seq ISeq) interface{} {
	if Seq(seq) == nil {
		panic("reduce of empty sequence without initial value")
	}
	return ReduceInit(f, seq.First(), seq.Next())
}

func ReduceInit(f func(interface{}, interface{}) interface{}, init interface{}, seq ISeq) interface{} {
	var res interface{} = init
	for ; seq != nil; seq = seq.Next() {
		res = f(res, seq.First())
	}
	return res
}
