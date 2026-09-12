package lang

import (
	"reflect"
	"runtime"
	"sync"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type monitorKey struct {
	typ     reflect.Type
	pointer uintptr
	value   any
}

type monitorState struct {
	owner int64
	depth int
	users int
}

var monitors = struct {
	sync.Mutex
	entries map[monitorKey]*monitorState
}{entries: make(map[monitorKey]*monitorState)}
var monitorChanged = sync.NewCond(&monitors.Mutex)

func objectMonitorKey(object any) monitorKey {
	if IsNil(object) {
		panic(NewIllegalArgumentError("locking requires a non-nil object"))
	}
	value := reflect.ValueOf(object)
	key := monitorKey{typ: value.Type()}
	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		key.pointer = value.Pointer()
	default:
		if !value.Comparable() {
			panic(NewIllegalArgumentError("locking requires an identity or comparable value"))
		}
		key.value = object
	}
	return key
}

// WithMonitor serializes access to an object, allowing nested acquisition by
// the same goroutine. Waiting users retain the entry until everyone leaves.
func WithMonitor(object any, body IFn) any {
	key := objectMonitorKey(object)
	owner := getGoroutineID()
	monitors.Lock()
	state := monitors.entries[key]
	if state == nil {
		state = &monitorState{}
		monitors.entries[key] = state
	}
	state.users++
	for state.depth != 0 && state.owner != owner {
		monitorChanged.Wait()
	}
	state.owner = owner
	state.depth++
	monitors.Unlock()
	defer func() {
		monitors.Lock()
		state.depth--
		state.users--
		if state.depth == 0 {
			state.owner = 0
			monitorChanged.Broadcast()
		}
		if state.users == 0 {
			delete(monitors.entries, key)
		}
		monitors.Unlock()
		// Keep pointer-backed objects alive until the monitor is released.
		runtime.KeepAlive(object)
	}()
	return body.Invoke()
}

func init() {
	pkgmap.Set("github.com/glojurelang/glojure/pkg/lang.WithMonitor", WithMonitor)
}
