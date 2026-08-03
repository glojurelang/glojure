package lang

import "testing"

func TestPersistentQueueFIFO(t *testing.T) {
	queue := EmptyPersistentQueue.Cons(1).(*PersistentQueue)
	queue = queue.Cons(2).(*PersistentQueue)
	queue = queue.Cons(3).(*PersistentQueue)
	if got := queue.Peek(); got != 1 {
		t.Fatalf("Peek = %v, want 1", got)
	}
	queue = queue.Pop().(*PersistentQueue)
	if got := queue.Peek(); got != 2 {
		t.Fatalf("Peek after Pop = %v, want 2", got)
	}
	if got := queue.Count(); got != 2 {
		t.Fatalf("Count = %d, want 2", got)
	}
	if !queue.Equiv(NewList(2, 3)) {
		t.Fatal("queue is not sequentially equivalent to its items")
	}
}
