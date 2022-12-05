package value

type (
	IPersistentStack interface {
		Peek() interface{}
		Pop() IPersistentStack
	}

	Comparer interface {
		Compare(other interface{}) int
	}
)
