package value

func CreatePersistentTreeMap(keyvals interface{}) interface{} {
	return NewMap(seqToSlice(Seq(keyvals))...)
}
