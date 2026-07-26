package lang

import "testing"

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
