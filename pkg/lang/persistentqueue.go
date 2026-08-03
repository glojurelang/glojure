package lang

import (
	"fmt"
	"reflect"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

// PersistentQueue is an immutable FIFO collection compatible with
// clojure.lang.PersistentQueue.
type PersistentQueue struct {
	meta  IPersistentMap
	items []any
}

var EmptyPersistentQueue = &PersistentQueue{}

var (
	_ IPersistentCollection = (*PersistentQueue)(nil)
	_ IPersistentStack      = (*PersistentQueue)(nil)
	_ Counted               = (*PersistentQueue)(nil)
	_ Sequential            = (*PersistentQueue)(nil)
	_ IObj                  = (*PersistentQueue)(nil)
	_ IFn                   = (*PersistentQueue)(nil)
)

func (q *PersistentQueue) xxx_sequential() {}
func (q *PersistentQueue) xxx_counted()    {}

func (q *PersistentQueue) Count() int { return len(q.items) }

// JVM static field syntax is emitted both as PersistentQueue/EMPTY and as the
// zero-argument form (PersistentQueue/EMPTY). Accept the latter while keeping
// the value usable directly as a collection.
func (q *PersistentQueue) Invoke(args ...any) any {
	if len(args) != 0 {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"PersistentQueue.EMPTY: wrong number of arguments (%d)", len(args))))
	}
	return q
}

func (q *PersistentQueue) ApplyTo(args ISeq) any {
	if args != nil {
		panic(NewIllegalArgumentError(fmt.Sprintf(
			"PersistentQueue.EMPTY: wrong number of arguments (%d)", args.Count())))
	}
	return q
}

func (q *PersistentQueue) Seq() ISeq {
	if len(q.items) == 0 {
		return nil
	}
	return NewSliceSeq(q.items)
}

// Cons implements Clojure's conj behavior for queues by appending at the rear.
func (q *PersistentQueue) Cons(value any) Conser {
	items := make([]any, len(q.items)+1)
	copy(items, q.items)
	items[len(q.items)] = value
	return &PersistentQueue{meta: q.meta, items: items}
}

func (q *PersistentQueue) Empty() IPersistentCollection {
	return EmptyPersistentQueue.WithMeta(q.meta).(IPersistentCollection)
}

func (q *PersistentQueue) Peek() any {
	if len(q.items) == 0 {
		return nil
	}
	return q.items[0]
}

func (q *PersistentQueue) Pop() IPersistentStack {
	if len(q.items) == 0 {
		panic(NewIllegalStateError("Can't pop empty queue"))
	}
	items := append([]any(nil), q.items[1:]...)
	return &PersistentQueue{meta: q.meta, items: items}
}

func (q *PersistentQueue) Equiv(other any) bool {
	seq := Seq(other)
	for _, item := range q.items {
		if seq == nil || !Equiv(item, seq.First()) {
			return false
		}
		seq = seq.Next()
	}
	return seq == nil
}

func (q *PersistentQueue) Equals(other any) bool { return q.Equiv(other) }
func (q *PersistentQueue) Meta() IPersistentMap  { return q.meta }

func (q *PersistentQueue) WithMeta(meta IPersistentMap) any {
	if q.meta == meta {
		return q
	}
	copy := *q
	copy.meta = meta
	return &copy
}

func (q *PersistentQueue) String() string { return PrintString(q) }

func init() {
	class := NewClass(reflect.TypeOf((*PersistentQueue)(nil)),
		"clojure.lang.PersistentQueue")
	pkgmap.SetHostClassPackage("PersistentQueue", "clojure.lang")
	pkgmap.SetHostClass("PersistentQueue", class)
	pkgmap.Set("clojure.lang.PersistentQueue.EMPTY", EmptyPersistentQueue)
}
