package lang

import (
	"regexp"
	"strings"
	"testing"
)

func TestRegexpMatcher(t *testing.T) {
	re := regexp.MustCompile(`\d+`)
	elements := []string{"123", "456", "789"}
	str := strings.Join(elements, " ")
	matcher := NewRegexpMatcher(re, str)

	if matcher.GroupCount() != 0 {
		t.Errorf("Expected 1 group, got %d", matcher.GroupCount())
	}

	if matcher.Matches() {
		t.Errorf("Expected string %q not to fully match regexp %v", str, re)
	}
	var result []string
	for matcher.Find() {
		result = append(result, matcher.Group())
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 matches, got %v", len(result))
	}
	for i, match := range result {
		if match != elements[i] {
			t.Errorf("Expected match %q, got %q", elements[i], match)
		}
	}
}

func TestRegexpMatcherGroups(t *testing.T) {
	re := regexp.MustCompile(`(\d+)-(\w+)`)
	elements := []string{"123-abc", "456-def", "789-ghi"}
	str := strings.Join(elements, " ")
	matcher := NewRegexpMatcher(re, str)

	if matcher.GroupCount() != 2 {
		t.Errorf("Expected 2 groups, got %d", matcher.GroupCount())
	}

	if matcher.Matches() {
		t.Errorf("Expected string %q not to fully match regexp %v", str, re)
	}

	i := 0
	for matcher.Find() {
		if matcher.Group() != elements[i] {
			t.Errorf("Expected match %q, got %q", elements[i], matcher.Group())
		}
		if matcher.GroupInt(1) != elements[i][0:3] {
			t.Errorf("Expected submatch %q, got %q", elements[i][0:3], matcher.GroupInt(1))
		}
		if matcher.GroupInt(2) != elements[i][4:7] {
			t.Errorf("Expected submatch %q, got %q", elements[i][4:7], matcher.GroupInt(2))
		}
		i++
	}
}

func TestRegexpMatcherReplace(t *testing.T) {
	re := regexp.MustCompile(`foo`)
	str := "foo bar foo bar foo bar"
	matcher := NewRegexpMatcher(re, str)

	sb := &strings.Builder{}
	for matcher.Find() {
		matcher.AppendReplacement(sb, "baz")
	}
	matcher.AppendTail(sb)

	if sb.String() != "baz bar baz bar baz bar" {
		t.Errorf("Expected string %q, got %q", "baz bar baz bar baz bar", sb.String())
	}
}
