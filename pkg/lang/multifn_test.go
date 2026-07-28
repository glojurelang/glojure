package lang

import (
	"fmt"
	"reflect"
	"testing"
)

func TestMultiFnFixedArityDispatchDoesNotAllocate(t *testing.T) {
	oldIsA := varIsA.Deref()
	defer varIsA.BindRoot(oldIsA)
	varIsA.BindRoot(FnFunc3(func(_, child, parent any) any {
		return Equals(child, parent)
	}))
	oldParents := VarParents.Deref()
	defer VarParents.BindRoot(oldParents)
	VarParents.BindRoot(FnFunc2(func(_, _ any) any { return nil }))

	hierarchyVar := NewVar(NSCore, NewSymbol("test-multifn-fixed-hierarchy"))
	hierarchyVar.BindRoot(NewMap())
	mf := NewMultiFn(
		"test-fixed-arity",
		FnFunc3(func(dispatch, _, _ any) any { return dispatch }),
		NewKeyword("default"),
		hierarchyVar,
	)
	mf.AddMethod("match", FnFunc3(func(_, left, right any) any {
		return left.(int64) + right.(int64)
	}))

	if got := Apply3(mf, "match", int64(20), int64(22)); got != int64(42) {
		t.Fatalf("fixed-arity result = %v, want 42", got)
	}
	if got := testing.AllocsPerRun(1_000, func() {
		if result := Apply3(
			mf,
			"match",
			int64(20),
			int64(22),
		); result != int64(42) {
			panic(result)
		}
	}); got != 0 {
		t.Fatalf("fixed-arity dispatch allocated %v objects per call, want 0", got)
	}
}

func TestProtocolMultiFnDispatchesDirectlyByTargetType(t *testing.T) {
	oldIsA := varIsA.Deref()
	defer varIsA.BindRoot(oldIsA)
	varIsA.BindRoot(FnFunc3(func(_, child, parent any) any {
		return Equals(child, parent)
	}))
	oldParents := VarParents.Deref()
	defer VarParents.BindRoot(oldParents)
	VarParents.BindRoot(FnFunc2(func(_, _ any) any { return nil }))

	hierarchyVar := NewVar(NSCore, NewSymbol("test-protocol-hierarchy"))
	hierarchyVar.BindRoot(NewMap())
	protocol := NewProtocolMultiFn("test-protocol", hierarchyVar)
	protocol.AddMethod(nil, FnFunc3(func(_, left, right any) any {
		return left.(int64) + right.(int64)
	}))

	if !protocol.IsProtocol() {
		t.Fatal("protocol multimethod did not retain its dispatch kind")
	}
	if got := Apply3(
		protocol,
		nil,
		int64(20),
		int64(22),
	); got != int64(42) {
		t.Fatalf("nil protocol result = %v, want 42", got)
	}
	if got := testing.AllocsPerRun(1_000, func() {
		if result := Apply3(
			protocol,
			nil,
			int64(20),
			int64(22),
		); result != int64(42) {
			panic(result)
		}
	}); got != 0 {
		t.Fatalf("protocol dispatch allocated %v objects per call, want 0", got)
	}

	protocol.AddMethod(
		TypeOf(""),
		FnFunc3(func(target, _, _ any) any { return target }),
	)
	if got := Apply3(protocol, "typed", nil, nil); got != "typed" {
		t.Fatalf("typed protocol result = %v, want typed", got)
	}

	protocol.AddMethod(
		TypeOf(""),
		FnFunc3(func(_, left, right any) any {
			return left.(int64) * right.(int64)
		}),
	)
	if got := Apply3(protocol, "typed", int64(6), int64(7)); got != int64(42) {
		t.Fatalf("re-extended protocol result = %v, want 42", got)
	}

	if got := protocol.ApplyTo(NewList("typed", int64(6), int64(7))); got != int64(42) {
		t.Fatalf("protocol ApplyTo result = %v, want 42", got)
	}
}

type protocolStringer string

func (s protocolStringer) String() string {
	return string(s)
}

func TestProtocolMultiFnUsesInterfaceAndDefaultMethods(t *testing.T) {
	oldIsA := varIsA.Deref()
	defer varIsA.BindRoot(oldIsA)
	varIsA.BindRoot(FnFunc3(func(_, child, parent any) any {
		childType, childIsType := child.(reflect.Type)
		parentType, parentIsType := parent.(reflect.Type)
		if childIsType && parentIsType {
			return childType.AssignableTo(parentType)
		}
		return Equals(child, parent)
	}))
	oldParents := VarParents.Deref()
	defer VarParents.BindRoot(oldParents)
	VarParents.BindRoot(FnFunc2(func(_, _ any) any { return nil }))

	hierarchyVar := NewVar(NSCore, NewSymbol("test-protocol-interface-hierarchy"))
	hierarchyVar.BindRoot(NewMap())
	protocol := NewProtocolMultiFn("test-protocol-interface", hierarchyVar)
	stringerType := reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	protocol.AddMethod(
		stringerType,
		FnFunc1(func(target any) any { return target.(fmt.Stringer).String() }),
	)
	protocol.AddMethod(
		NewKeyword("default"),
		FnFunc1(func(any) any { return "default" }),
	)

	if got := Apply1(protocol, protocolStringer("interface")); got != "interface" {
		t.Fatalf("interface protocol result = %v, want interface", got)
	}
	if got := Apply1(protocol, int64(42)); got != "default" {
		t.Fatalf("default protocol result = %v, want default", got)
	}
}

func TestMultiFnRefreshesHierarchyBeforeMethodSelection(t *testing.T) {
	oldIsA := varIsA.Deref()
	defer varIsA.BindRoot(oldIsA)
	oldParents := VarParents.Deref()
	defer VarParents.BindRoot(oldParents)

	hierarchyVar := NewVar(NSCore, NewSymbol("test-multifn-hierarchy"))
	mf := NewMultiFn(
		"test-hierarchy-refresh",
		FnFunc1(func(value any) any { return value }),
		NewKeyword("default"),
		hierarchyVar,
	)
	mf.AddMethod("match", FnFunc1(func(any) any { return "matched" }))

	hierarchy := NewMap()
	hierarchyVar.BindRoot(hierarchy)
	varIsA.BindRoot(FnFunc3(func(gotHierarchy, child, parent any) any {
		if gotHierarchy != hierarchy {
			t.Fatalf("isa? hierarchy = %v, want current hierarchy", gotHierarchy)
		}
		return child == parent
	}))
	VarParents.BindRoot(FnFunc2(func(gotHierarchy, _ any) any {
		if gotHierarchy != hierarchy {
			t.Fatalf("parents hierarchy = %v, want current hierarchy", gotHierarchy)
		}
		return nil
	}))

	if got := mf.Invoke("match"); got != "matched" {
		t.Fatalf("multimethod result = %v, want matched", got)
	}
}

func TestMultiFnGenerationTracksMethodsAndHierarchy(t *testing.T) {
	hierarchyVar := NewVar(NSCore, NewSymbol("test-multifn-generation"))
	firstHierarchy := NewMap()
	hierarchyVar.BindRoot(firstHierarchy)
	mf := NewMultiFn(
		"test-generation",
		FnFunc1(func(value any) any { return value }),
		NewKeyword("default"),
		hierarchyVar,
	)
	mf.AddMethod("first", FnFunc1(func(any) any { return "first" }))

	generation := mf.Generation()
	if !mf.IsGeneration(generation) {
		t.Fatal("fresh multimethod generation was not stable")
	}
	if exactGeneration, exact := mf.ExactGeneration(); !exact ||
		exactGeneration != generation {
		t.Fatalf("exact generation = (%d, %v), want (%d, true)",
			exactGeneration, exact, generation)
	}
	if got := testing.AllocsPerRun(1_000, func() {
		if !mf.IsGeneration(generation) {
			panic("generation changed")
		}
	}); got != 0 {
		t.Fatalf("stable generation check allocated %v objects, want 0", got)
	}
	mf.AddMethod("second", FnFunc1(func(any) any { return "second" }))
	if mf.IsGeneration(generation) {
		t.Fatal("adding a method did not invalidate the generation")
	}

	generation = mf.Generation()
	if !mf.IsGeneration(generation) {
		t.Fatal("updated multimethod generation was not stable")
	}
	hierarchyVar.BindRoot(NewMap(
		NewKeyword("parents"),
		NewMap("child", NewSet("parent")),
	))
	if mf.IsGeneration(generation) {
		t.Fatal("changing the hierarchy did not invalidate the generation")
	}
	if mf.Generation() == generation {
		t.Fatal("hierarchy invalidation did not advance the generation")
	}
	if _, exact := mf.ExactGeneration(); exact {
		t.Fatal("nonempty hierarchy was reported as exact")
	}
}
