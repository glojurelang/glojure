package codegen

import (
	"bytes"
	"testing"
)

func TestVarAllocator(t *testing.T) {
	var buf bytes.Buffer
	gen := New(&buf)

	// Test initial allocation
	if name := gen.allocateVar("x"); name != "v0" {
		t.Errorf("expected first var to be v0, got %s", name)
	}
	if name := gen.allocateVar("y"); name != "v1" {
		t.Errorf("expected second var to be v1, got %s", name)
	}
	
	// Test that same name in same scope returns same variable name
	if name := gen.allocateVar("x"); name != "v0" {
		t.Errorf("expected x to still be v0, got %s", name)
	}

	// Test pushing a new scope
	gen.pushVarScope()
	
	// New scope should start from where the previous scope left off
	if name := gen.allocateVar("z"); name != "v2" {
		t.Errorf("expected first var in new scope to be v2, got %s", name)
	}
	
	// Same name in new scope should get new variable name
	if name := gen.allocateVar("x"); name != "v3" {
		t.Errorf("expected x in new scope to be v3, got %s", name)
	}
	
	// Test popping scope
	gen.popVarScope()
	
	// Back in original scope, allocating new var should continue from where we left off
	if name := gen.allocateVar("w"); name != "v2" {
		t.Errorf("expected w to be v2 after popping scope, got %s", name)
	}
	
	// Original x should still be v0
	if name := gen.allocateVar("x"); name != "v0" {
		t.Errorf("expected x to be v0 after popping scope, got %s", name)
	}
}

// TestVarName is no longer needed since allocateVar returns the name directly

func TestPopRootScopePanics(t *testing.T) {
	var buf bytes.Buffer
	gen := New(&buf)
	
	// Should panic when trying to pop the root scope
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when popping root scope")
		}
	}()
	
	gen.popVarScope()
}