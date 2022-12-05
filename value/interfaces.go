package value

type (
	IPersistentStack interface {
		Peek() interface{}
		Pop() IPersistentStack
	}
)
