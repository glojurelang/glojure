package lang

// ArrayList is a minimal implementation of a subset of Java's
// ArrayList to replace uses of java.util.ArrayList in the Clojure
// standard library.
type ArrayList struct {
	data []any
}

func NewArrayList(items []any) *ArrayList {
	return &ArrayList{
		data: items,
	}
}

func (al *ArrayList) Add(item any) {
	al.data = append(al.data, item)
}

func (al *ArrayList) Clear() {
	al.data = []any{}
}

func (al *ArrayList) IsEmpty() bool {
	return len(al.data) == 0
}

func (al *ArrayList) ToArray() []any {
	return al.data
}
