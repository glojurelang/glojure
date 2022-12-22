package value

// MapEntry represents a key-value pair in a map.
type MapEntry struct {
	key, val interface{}
}

func (me *MapEntry) Key() interface{} {
	if me.key == nil {
		return nil
	}
	return me.key
}

func (me *MapEntry) Val() interface{} {
	if me.val == nil {
		return nil
	}
	return me.val
}
