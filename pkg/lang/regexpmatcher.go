package lang

import (
	"io"
	"regexp"
)

type (
	// RegexpMatcher is a matcher that matches a string against a
	// regular expression. It's a wrapper around standard library
	// functions. We implement this to simplify a translation of clojure
	// regexp core functions to go.
	RegexpMatcher struct {
		re *regexp.Regexp
		s  string

		lastMatch       []int
		lastMatchOffset int
	}
)

// NewRegexpMatcher creates a new RegexpMatcher.
func NewRegexpMatcher(re *regexp.Regexp, s string) *RegexpMatcher {
	return &RegexpMatcher{re: re, s: s}
}

// ResolveFieldOrMethod keeps interop with the runtime-owned matcher on direct
// fixed-arity calls. Matchers are short-lived, so resolving their bound Go
// methods through reflection would otherwise allocate a method value and a
// reflection adapter for every matcher.
func (m *RegexpMatcher) ResolveFieldOrMethod(
	name string,
) (interface{}, bool) {
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
		name = string(name[0]-'a'+'A') + name[1:]
	}
	switch name {
	case "Find":
		return FnFunc0(func() any { return m.Find() }), true
	case "GroupCount":
		return FnFunc0(func() any { return m.GroupCount() }), true
	case "Group":
		return FnFunc0(func() any { return m.Group() }), true
	case "GroupInt":
		return FnFunc1(func(group any) any {
			return m.GroupInt(MustAsInt(group))
		}), true
	case "Matches":
		return FnFunc0(func() any { return m.Matches() }), true
	case "AppendReplacement":
		return FnFunc2(func(writer, replacement any) any {
			return m.AppendReplacement(writer.(io.Writer), replacement.(string))
		}), true
	case "AppendTail":
		return FnFunc1(func(writer any) any {
			m.AppendTail(writer.(io.Writer))
			return nil
		}), true
	default:
		return nil, false
	}
}

// Find attempts to find the next subsequence of the input sequence
// that matches the pattern.
func (m *RegexpMatcher) Find() bool {
	var nextStart int
	if len(m.lastMatch) > 0 {
		if m.lastMatchOffset+m.lastMatch[1] == len(m.s) {
			return false
		}
		if m.lastMatch[0] == m.lastMatch[1] {
			nextStart = m.lastMatchOffset + m.lastMatch[1] + 1
		} else {
			nextStart = m.lastMatchOffset + m.lastMatch[1]
		}
	}

	match := m.re.FindStringSubmatchIndex(m.s[nextStart:])
	if match == nil {
		return false
	}
	m.lastMatch = match

	m.lastMatchOffset = nextStart
	return true
}

// GroupCount returns the number of capturing groups in this matcher's
// pattern.
func (m *RegexpMatcher) GroupCount() int {
	return m.re.NumSubexp()
}

// Group returns the input subsequence matched by the previous match.
func (m *RegexpMatcher) Group() string {
	if len(m.lastMatch) == 0 {
		return ""
	}
	return m.GroupInt(0).(string)
}

// GroupInt returns the input subsequence captured by the given group
// during the previous match operation.
func (m *RegexpMatcher) GroupInt(group int) any {
	if len(m.lastMatch) == 0 {
		panic(NewIndexOutOfBoundsError())
	}
	if group < 0 || group >= len(m.lastMatch)/2 {
		panic(NewIndexOutOfBoundsError())
	}
	start, end := m.lastMatch[2*group], m.lastMatch[2*group+1]
	if start == -1 || end == -1 {
		return nil
	}
	return m.s[m.lastMatchOffset+start : m.lastMatchOffset+end]
}

// Matches attempts to match the entire region against the pattern.
func (m *RegexpMatcher) Matches() bool {
	match := m.re.FindStringSubmatchIndex(m.s)
	if match == nil || match[0] != 0 || match[1] != len(m.s) {
		return false
	}
	m.lastMatch = match
	m.lastMatchOffset = 0
	return true
}

// AppendReplacement implements a non-terminal append-and-replace step.
func (m *RegexpMatcher) AppendReplacement(sb io.Writer, replacement string) *RegexpMatcher {
	if len(m.lastMatch) == 0 {
		return m
	}
	io.WriteString(sb, m.s[m.lastMatchOffset:m.lastMatchOffset+m.lastMatch[0]])
	io.WriteString(sb, replacement)
	return m
}

// AppendTail implements a terminal append-and-replace step.
func (m *RegexpMatcher) AppendTail(sb io.Writer) {
	if len(m.lastMatch) == 0 {
		io.WriteString(sb, m.s)
		return
	}
	io.WriteString(sb, m.s[m.lastMatchOffset+m.lastMatch[1]:])
}
