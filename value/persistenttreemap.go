package value

func CreatePersistentTreeMap(keyvals interface{}) interface{} {
	// TODO: implement
	return NewMap(seqToSlice(Seq(keyvals))...)
}
