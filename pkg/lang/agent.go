package lang

import "time"

type (
	Agent struct {
		meta IPersistentMap

		watches IPersistentMap
	}

	future struct {
		done chan struct{}
		res  interface{}
	}
)

var (
	// _ ARef = (*Agent)(nil)

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

////////////////////////////////////////////////////////////////////////////////
// Agent

func (a *Agent) Deref() any {
	panic("not implemented")
}

func (a *Agent) Watches() IPersistentMap {
	return a.watches
}

// func (a *Agent) AddWatch(key interface{}, fn IFn) IRef {
// 	a.watches = a.watches.Assoc(key, fn).(IPersistentMap)
// 	return a
// }

func (a *Agent) RemoveWatch(key interface{}) {
	a.watches = a.watches.Without(key)
}

func (a *Agent) notifyWatches(oldVal, newVal interface{}) {
	watches := a.watches
	if watches == nil || watches.Count() == 0 {
		return
	}

	for seq := watches.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First().(IMapEntry)
		key := entry.Key()
		fn := entry.Val().(IFn)
		// Call watch function with key, ref, old-state, new-state
		fn.Invoke(key, a, oldVal, newVal)
	}
}

////////////////////////////////////////////////////////////////////////////////

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
