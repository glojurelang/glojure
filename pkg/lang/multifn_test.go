package lang

import (
	"reflect"
	"testing"
)

type testKVReducible struct{}

func (*testKVReducible) KVReduce(f IFn, init any) any {
	return f.Invoke(init, NewKeyword("value"), int64(42))
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

func TestKVReduceAutoRegisteredMethods(t *testing.T) {
	oldIsA := varIsA.Deref()
	defer varIsA.BindRoot(oldIsA)
	oldParents := VarParents.Deref()
	defer VarParents.BindRoot(oldParents)

	hierarchy := NewMap()
	hierarchyVar := NewVar(NSCore, NewSymbol("test-kv-reduce-hierarchy"))
	hierarchyVar.BindRoot(hierarchy)
	varIsA.BindRoot(FnFunc3(func(_ any, child, parent any) any {
		if child == parent {
			return true
		}
		childType, childOK := child.(reflect.Type)
		parentType, parentOK := parent.(reflect.Type)
		return childOK && parentOK && childType.AssignableTo(parentType)
	}))
	VarParents.BindRoot(FnFunc2(func(_, _ any) any { return nil }))

	mf := NewMultiFn(
		"kv-reduce",
		FnFunc(func(args ...any) any { return TypeOf(args[0]) }),
		NewKeyword("default"),
		hierarchyVar,
	)
	reducer := FnFunc3(func(acc, _, value any) any {
		return acc.(int64) + value.(int64)
	})

	if got := mf.Invoke(nil, reducer, int64(7)); got != int64(7) {
		t.Fatalf("nil kv-reduce = %v, want 7", got)
	}
	if got := mf.Invoke(&testKVReducible{}, reducer, int64(0)); got != int64(42) {
		t.Fatalf("native kv-reduce = %v, want 42", got)
	}
	if got := mf.Invoke(NewPersistentHashMap("one", int64(1), "two", int64(2)), reducer, int64(0)); got != int64(3) {
		t.Fatalf("fallback kv-reduce = %v, want 3", got)
	}

	methods := mf.GetMethodTable()
	if methods.Count() != 3 {
		t.Fatalf("registered kv-reduce method count = %d, want 3", methods.Count())
	}
	for entries := methods.Seq(); entries != nil; entries = entries.Next() {
		entry := entries.First().(IMapEntry)
		if !IsAutoRegisteredMethod("kv-reduce", entry.Key(), entry.Val()) {
			t.Fatalf("kv-reduce method for %v was not recognized as auto-registered", entry.Key())
		}
	}
}

func TestMultiFnDispatchesRecordsByConcreteIdentityAndInterfaces(t *testing.T) {
	oldIsA := varIsA.Deref()
	defer varIsA.BindRoot(oldIsA)
	oldParents := VarParents.Deref()
	defer VarParents.BindRoot(oldParents)

	hierarchy := NewMap()
	hierarchyVar := NewVar(NSCore, NewSymbol("test-record-hierarchy"))
	hierarchyVar.BindRoot(hierarchy)
	varIsA.BindRoot(FnFunc3(func(_, child, parent any) any { return child == parent }))
	VarParents.BindRoot(FnFunc2(func(_, _ any) any { return nil }))

	specificType := InternRecordType("test", "SpecificDispatchRecord", "value")
	otherType := InternRecordType("test", "OtherDispatchRecord", "value")
	mf := NewMultiFn("record-dispatch",
		FnFunc1(func(value any) any { return TypeOf(value) }),
		NewKeyword("default"), hierarchyVar)
	mf.AddMethod(reflect.TypeOf((*IRecord)(nil)).Elem(),
		FnFunc1(func(any) any { return "record" }))
	mf.AddMethod(specificType, FnFunc1(func(any) any { return "specific" }))

	if got := mf.Invoke(NewRecord(specificType, 1)); got != "specific" {
		t.Fatalf("specific record dispatch = %v, want specific", got)
	}
	if got := mf.Invoke(NewRecord(otherType, 1)); got != "record" {
		t.Fatalf("generic record dispatch = %v, want record", got)
	}
}
