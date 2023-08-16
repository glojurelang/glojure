package lang

import "time"

type (
	Agent struct{}

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

func (f *future) DerefWithTimeout(timeoutMS int64, timeoutVal interface{}) interface{} {
	select {
	case <-f.done:
		return f.res
	case <-time.After(time.Duration(timeoutMS) * time.Millisecond):
		return timeoutVal
	}
}

func (f *future) Get() interface{} {
	return f.Deref()
}

func (f *future) GetWithTimeout(timeout int64, timeUnit time.Duration) interface{} {
	select {
	case <-f.done:
		return f.res
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		panic(NewTimeoutError("future timeout"))
	}
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
