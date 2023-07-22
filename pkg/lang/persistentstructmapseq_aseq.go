// GENERATED CODE. DO NOT EDIT
package lang

func (s *persistentStructMapSeq) xxx_sequential() {}

func (s *persistentStructMapSeq) More() ISeq {
	sq := s.Next()
	if sq == nil {
		return emptyList
	}
	return sq
}

func (s *persistentStructMapSeq) Seq() ISeq {
	return s
}

func (s *persistentStructMapSeq) Meta() IPersistentMap {
	return s.meta
}

func (s *persistentStructMapSeq) WithMeta(meta IPersistentMap) interface{} {
	if Equal(s.meta, meta) {
		return s
	}
	cpy := *s
	cpy.meta = meta
	return &cpy
}
