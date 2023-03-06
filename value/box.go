package value

type Box struct {
	val interface{}
}

func NewBox(val interface{}) *Box {
	return &Box{val}
}
