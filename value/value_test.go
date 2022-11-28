package value_test

import (
	"strings"
	"testing"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/value"
)

func TestBind(t *testing.T) {
	type testCase struct {
		name     string
		pattern  string
		value    string
		expected string
	}

	testCases := []testCase{
		// sequential destructuring
		{
			name:     "simple",
			pattern:  "[a b]",
			value:    "(1 2)",
			expected: "[a 1 b 2]",
		},
		{
			name:     "nested",
			pattern:  "[a [b c]]",
			value:    "(1 [2 3])",
			expected: "[a 1 b 2 c 3]",
		},
		{
			name:     "ignore extras",
			pattern:  "[a [b c]]",
			value:    "(1 [2 3 4] 5 6)",
			expected: "[a 1 b 2 c 3]",
		},
		{
			name:     "rest",
			pattern:  "[a b & rest]",
			value:    "(1 2 3 4 5)",
			expected: "[a 1 b 2 rest (3 4 5)]",
		},
		{
			name:     "rest destructured",
			pattern:  "[a b & [c d & rest]]",
			value:    "(1 2 3 4 5 6)",
			expected: "[a 1 b 2 c 3 d 4 rest (5 6)]",
		},
	}

	read := func(t *testing.T, s string) interface{} {
		val, err := reader.New(strings.NewReader(s)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(val) != 1 {
			t.Fatalf("expected 1 value, got %d", len(val))
		}
		return val[0]
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pattern := read(t, tc.pattern).(*value.Vector)
			val := read(t, tc.value)
			expected := read(t, tc.expected)

			bindings, err := value.Bind(pattern, val)
			if err != nil {
				t.Fatal(err)
			}
			bindingsVector := value.NewVector(bindings)
			if !bindingsVector.Equal(expected) {
				t.Fatalf("expected %v, got %v", expected, bindingsVector)
			}
		})
	}
}
