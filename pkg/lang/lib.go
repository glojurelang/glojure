package lang

func Pop(stk IPersistentStack) IPersistentStack {
	return stk.Pop()
}

func Peek(stk IPersistentStack) interface{} {
	return stk.Peek()
}
