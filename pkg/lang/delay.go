package lang

import "sync"

type (
	Delay struct {
		val       any
		exception any
		fn        IFn
		mtx       *sync.Mutex
	}
)

var (
	_ IDeref   = (*Delay)(nil)
	_ IPending = (*Delay)(nil)
)

func NewDelay(fn IFn) *Delay {
	return &Delay{
		fn:  fn,
		mtx: &sync.Mutex{},
	}
}

func (d *Delay) realize() {
	l := d.mtx
	if l == nil {
		return
	}

	l.Lock()
	defer l.Unlock()

	if d.fn != nil {
		defer func() {
			if r := recover(); r != nil {
				d.exception = r
			}
		}()
		d.val = d.fn.Invoke()
		d.fn = nil
		d.mtx = nil
	}
}

func (d *Delay) Deref() any {
	if d.mtx != nil {
		d.realize()
	}
	if d.exception != nil {
		// TODO: look into Util.sneakyThrow in clojure. can we do something similar in Go?
		panic(d.exception)
	}
	return d.val
}

func (d *Delay) IsRealized() bool {
	return d.mtx == nil
}

func ForceDelay(x any) any {
	if d, ok := x.(*Delay); ok {
		return d.Deref()
	}
	return x
}
