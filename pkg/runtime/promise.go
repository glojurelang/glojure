package runtime

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gloathub/glojure/pkg/lang"
)

// Promise implements Clojure's promise semantics using Go sync primitives.
type Promise struct {
	val       atomic.Value
	delivered atomic.Bool
	once      sync.Once
	done      chan struct{}
}

func NewPromise() *Promise {
	return &Promise{
		done: make(chan struct{}),
	}
}

func (p *Promise) Deref() any {
	<-p.done
	return p.val.Load()
}

func (p *Promise) DerefWithTimeout(timeoutMs int64, timeoutVal any) any {
	select {
	case <-p.done:
		return p.val.Load()
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return timeoutVal
	}
}

func (p *Promise) IsRealized() bool {
	return p.delivered.Load()
}

func (p *Promise) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(lang.NewIllegalArgumentError("promise invoke expects 1 argument"))
	}
	delivered := false
	p.once.Do(func() {
		p.val.Store(args[0])
		p.delivered.Store(true)
		close(p.done)
		delivered = true
	})
	if delivered {
		return p
	}
	return nil
}

func (p *Promise) ApplyTo(args lang.ISeq) any {
	var a []any
	for s := args; s != nil; s = s.Next() {
		a = append(a, s.First())
	}
	return p.Invoke(a...)
}
