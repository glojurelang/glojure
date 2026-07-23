package goid

import (
	"runtime"
	"sync"
	"testing"
)

func TestGetStableWithinGoroutine(t *testing.T) {
	want := Get()
	for range 100 {
		runtime.Gosched()
		if got := Get(); got != want {
			t.Fatalf("Get() changed from %d to %d in one goroutine", want, got)
		}
	}
}

func TestGetDistinctForConcurrentGoroutines(t *testing.T) {
	const goroutines = 32
	ids := make(chan int64, goroutines)
	release := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(goroutines)

	for range goroutines {
		go func() {
			ids <- Get()
			ready.Done()
			<-release
		}()
	}
	ready.Wait()
	close(release)
	close(ids)

	seen := make(map[int64]struct{}, goroutines)
	for id := range ids {
		if id == 0 {
			t.Fatal("Get() returned zero")
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("Get() returned duplicate concurrent identity %d", id)
		}
		seen[id] = struct{}{}
	}
}
