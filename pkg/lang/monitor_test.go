package lang

import (
	"sync"
	"testing"
)

func TestMonitorReentryAndUnwind(t *testing.T) {
	object := new(int)
	result := WithMonitor(object, FnFunc0(func() any {
		return WithMonitor(object, FnFunc0(func() any { return 42 }))
	}))
	if result != 42 {
		t.Fatalf("nested locking returned %v", result)
	}
	func() {
		defer func() {
			if recover() != "failure" {
				t.Fatal("locking did not propagate panic")
			}
		}()
		WithMonitor(object, FnFunc0(func() any { panic("failure") }))
	}()
	WithMonitor(object, FnFunc0(func() any { return nil }))
	monitors.Lock()
	defer monitors.Unlock()
	if len(monitors.entries) != 0 {
		t.Fatal("monitor registry retained an unused entry")
	}
}

func TestMonitorConcurrentUpdates(t *testing.T) {
	object := new(int)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				WithMonitor(object, FnFunc0(func() any {
					*object++
					return nil
				}))
			}
		}()
	}
	workers.Wait()
	if *object != 800 {
		t.Fatalf("lost updates: %d", *object)
	}
}
