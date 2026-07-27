package ast

import (
	"testing"
)

func TestTransformRewritesChildrenBeforeParents(t *testing.T) {
	one := &Node{Op: OpConst, Sub: &ConstNode{Value: int64(1)}}
	two := &Node{Op: OpConst, Sub: &ConstNode{Value: int64(2)}}
	root := &Node{
		Op: OpVector,
		Sub: &VectorNode{Items: []*Node{
			one,
			{Op: OpVector, Sub: &VectorNode{Items: []*Node{two}}},
		}},
	}

	var visited []NodeOp
	transformed, err := Transform(root, func(node *Node) (*Node, error) {
		visited = append(visited, node.Op)
		if node.Op != OpConst {
			return node, nil
		}
		value := node.Sub.(*ConstNode).Value.(int64)
		replacement := *node
		replacement.Sub = &ConstNode{Value: value + 10}
		return &replacement, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	got := transformed.Sub.(*VectorNode).Items
	if value := got[0].Sub.(*ConstNode).Value; value != int64(11) {
		t.Fatalf("first transformed value = %v, want 11", value)
	}
	nested := got[1].Sub.(*VectorNode).Items[0]
	if value := nested.Sub.(*ConstNode).Value; value != int64(12) {
		t.Fatalf("nested transformed value = %v, want 12", value)
	}
	want := []NodeOp{OpConst, OpConst, OpVector, OpVector}
	if len(visited) != len(want) {
		t.Fatalf("visit order = %v, want %v", visited, want)
	}
	for i := range want {
		if visited[i] != want[i] {
			t.Fatalf("visit order = %v, want %v", visited, want)
		}
	}
}

func TestTransformVisitsCaseBranches(t *testing.T) {
	constNode := func(value int64) *Node {
		return &Node{Op: OpConst, Sub: &ConstNode{Value: value}}
	}
	root := &Node{
		Op: OpCase,
		Sub: &CaseNode{
			Test:    constNode(1),
			Default: constNode(2),
			Entries: []CaseEntry{{
				TestConstant: constNode(3),
				ResultExpr:   constNode(4),
			}},
		},
	}
	count := 0
	if _, err := Transform(root, func(node *Node) (*Node, error) {
		if node.Op == OpConst {
			count++
		}
		return node, nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("visited %d case constants, want 4", count)
	}
}
