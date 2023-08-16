package lang

import "time"

type (
	future struct {
		done chan struct{}
		res  interface{}
	}
)

var (
	_ IBlockingDeref = (*future)(nil)
	_ IDeref         = (*future)(nil)
	_ IPending       = (*future)(nil)
	_ Future         = (*future)(nil)
)

func (f *future) Deref() interface{} {
	<-f.done
	return f.res
}

func (f *future) DerefWithTimeout(ms int64, timeoutValue interface{}) interface{} {
	select {
	case <-f.done:
		return f.res
	case <-time.After(time.Duration(ms) * time.Millisecond):
		return timeoutValue
	}
}

func (f *future) Get() interface{} {
	return f.Deref()
}

func (f *future) GetWithTimeout(ms int64, timeoutValue interface{}) interface{} {
	return f.DerefWithTimeout(ms, timeoutValue)
}

func (f *future) IsRealized() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

func ShutdownAgents() {
	// TODO
}

func AgentSubmit(fn IFn) IBlockingDeref {
	fut := &future{
		done: make(chan struct{}),
	}
	go func() {
		fut.res = fn.Invoke()
		close(fut.done)
	}()
	return fut
}
