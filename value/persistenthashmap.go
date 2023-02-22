package value

func CreatePersistentHashMap(keyvals interface{}) interface{} {
	return NewMap(seqToSlice(Seq(keyvals))...)
}
