package compiler

import (
	"testing"

	"github.com/glojurelang/glojure/pkg/ast"
	. "github.com/glojurelang/glojure/pkg/lang"
)

func TestCaseStar(t *testing.T) {
	// Helper to create a case* form
	makeCaseStar := func(expr interface{}, shift, mask int64, defaultForm interface{}, caseMap IPersistentMap, switchType, testType Keyword, skipCheck interface{}) interface{} {
		args := []interface{}{
			NewSymbol("case*"),
			expr,
			shift,
			mask,
			defaultForm,
			caseMap,
			switchType,
			testType,
		}
		if skipCheck != nil {
			args = append(args, skipCheck)
		}
		return NewList(args...)
	}
	
	tests := []struct {
		name     string
		form     interface{}
		expected map[string]interface{} // Expected properties of the parsed case*
	}{
		{
			name: "integer case - direct lookup",
			form: makeCaseStar(
				NewSymbol("x"),
				0, 0,
				KWDefault,
				NewPersistentArrayMapAsIfByAssoc([]interface{}{
					int64(1), NewVector(int64(1), KWOne),
					int64(2), NewVector(int64(2), KWTwo),
					int64(3), NewVector(int64(3), KWThree),
				}),
				KWCompact,
				KWInt,
				nil,
			),
			expected: map[string]interface{}{
				"testType":   KWInt,
				"switchType": KWCompact,
				"shift":      int64(0),
				"mask":       int64(0),
				"numEntries": 3,
				"hasDefault": true,
			},
		},
		{
			name: "collision case - false and nil",
			form: makeCaseStar(
				NewSymbol("x"),
				0, 0,
				KWDefault,
				NewPersistentArrayMapAsIfByAssoc([]interface{}{
					int64(2654435769),
					NewVector(
						int64(0),
						// For testing, just use a simple keyword result
						// In reality this would be a condp expression
						KWCollisionResult,
					),
				}),
				KWCompact,
				KWHashEquiv,
				NewSet(int64(0)),
			),
			expected: map[string]interface{}{
				"testType":      KWHashEquiv,
				"switchType":    KWCompact,
				"shift":         int64(0),
				"mask":          int64(0),
				"numEntries":    1,
				"hasDefault":    true,
				"hasCollision":  true,
				"skipCheckSize": 1,
			},
		},
	}

	analyzer := &Analyzer{
		FindNamespace: func(sym *Symbol) *Namespace {
			return nil
		},
		Macroexpand1: func(form interface{}) (interface{}, error) {
			return form, nil // No macroexpansion for tests
		},
		IsVar: func(v interface{}) bool {
			return false
		},
		CreateVar: func(sym *Symbol, env Env) (interface{}, error) {
			return nil, nil
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a binding node for "x"
			xBinding := &ast.Node{
				Op: ast.OpBinding,
				Sub: &ast.BindingNode{
					Name: NewSymbol("x"),
				},
			}
			
			// Analyze the case* form
			// Add "x" as a local binding in the environment
			env := Env(NewPersistentArrayMapAsIfByAssoc([]interface{}{
				KWLocals, NewPersistentArrayMapAsIfByAssoc([]interface{}{
					NewSymbol("x"), xBinding,
				}),
			}))
			node, err := analyzer.parseCaseStar(tt.form, env)
			if err != nil {
				t.Fatalf("Failed to parse case*: %v", err)
			}
			
			// Verify the AST structure
			caseNode, ok := node.Sub.(*ast.CaseNode)
			if !ok {
				t.Fatalf("Expected CaseNode, got %T", node.Sub)
			}
			
			// Check expected properties
			if expected, ok := tt.expected["testType"]; ok {
				if caseNode.TestType != expected {
					t.Errorf("TestType: expected %v, got %v", expected, caseNode.TestType)
				}
			}
			
			if expected, ok := tt.expected["switchType"]; ok {
				if caseNode.SwitchType != expected {
					t.Errorf("SwitchType: expected %v, got %v", expected, caseNode.SwitchType)
				}
			}
			
			if expected, ok := tt.expected["shift"]; ok {
				if caseNode.Shift != expected.(int64) {
					t.Errorf("Shift: expected %v, got %v", expected, caseNode.Shift)
				}
			}
			
			if expected, ok := tt.expected["mask"]; ok {
				if caseNode.Mask != expected.(int64) {
					t.Errorf("Mask: expected %v, got %v", expected, caseNode.Mask)
				}
			}
			
			if expected, ok := tt.expected["numEntries"]; ok {
				if len(caseNode.Entries) != expected.(int) {
					t.Errorf("Number of entries: expected %v, got %v", expected, len(caseNode.Entries))
				}
			}
			
			if expected, ok := tt.expected["hasDefault"]; ok {
				hasDefault := caseNode.Default != nil
				if hasDefault != expected.(bool) {
					t.Errorf("Has default: expected %v, got %v", expected, hasDefault)
				}
			}
			
			if expected, ok := tt.expected["hasCollision"]; ok && expected.(bool) {
				// Check that at least one entry has a collision
				hasCollision := false
				for _, entry := range caseNode.Entries {
					if entry.HasCollision {
						hasCollision = true
						break
					}
				}
				if !hasCollision {
					t.Error("Expected at least one collision entry")
				}
			}
			
			if expected, ok := tt.expected["skipCheckSize"]; ok {
				if len(caseNode.SkipCheck) != expected.(int) {
					t.Errorf("SkipCheck size: expected %v, got %v", expected, len(caseNode.SkipCheck))
				}
			}
		})
	}
}

func TestCaseStarEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		caseExpr string
		testVal  interface{}
		expected interface{}
	}{
		{
			name:     "integer match",
			caseExpr: `(case 2 1 :one 2 :two 3 :three)`,
			testVal:  nil, // embedded in expression
			expected: KWTwo,
		},
		{
			name:     "integer no match with default",
			caseExpr: `(case 5 1 :one 2 :two :default)`,
			testVal:  nil,
			expected: KWDefault,
		},
		{
			name:     "keyword match",
			caseExpr: `(case :b :a :result-a :b :result-b)`,
			testVal:  nil,
			expected: KWResultB,
		},
		{
			name:     "string match",
			caseExpr: `(case "foo" "foo" :found "bar" :not-found)`,
			testVal:  nil,
			expected: KWFound,
		},
		{
			name:     "false vs nil - false case",
			caseExpr: `(case false false :false-result nil :nil-result :default)`,
			testVal:  nil,
			expected: KWFalseResult,
		},
		{
			name:     "false vs nil - nil case",
			caseExpr: `(case nil false :false-result nil :nil-result :default)`,
			testVal:  nil,
			expected: KWNilResult,
		},
		{
			name:     "false vs nil - default case",
			caseExpr: `(case true false :false-result nil :nil-result :default)`,
			testVal:  nil,
			expected: KWDefault,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will be implemented after we fix the analyzer and evaluator
			// For now, just document what we expect
			t.Skip("Evaluation tests will be implemented after analyzer fixes")
		})
	}
}

// Helper function to read a string into a form
func readString(s string) interface{} {
	// For testing, we'll manually construct the forms
	// This is a simplified version - real implementation would use the reader
	
	// For now, skip the actual parsing and return a placeholder
	// The tests will need to be updated to use actual constructed forms
	return NewList(NewSymbol("case*"))
}

// Test keywords used in tests
var (
	KWOne             = NewKeyword("one")
	KWTwo             = NewKeyword("two")
	KWThree           = NewKeyword("three")
	KWFalse           = NewKeyword("false")
	KWNilValue        = NewKeyword("nil")
	KWResultB         = NewKeyword("result-b")
	KWFound           = NewKeyword("found")
	KWFalseResult     = NewKeyword("false-result")
	KWNilResult       = NewKeyword("nil-result")
	KWCollisionResult = NewKeyword("collision-result")
)