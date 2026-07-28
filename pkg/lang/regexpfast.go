package lang

import (
	"regexp"
	"regexp/syntax"
	"strings"
	"sync"
)

// fixedASCIIClass is a compact set of ASCII bytes accepted at one position in
// a fixed-width regular expression.
type fixedASCIIClass [2]uint64

func (class *fixedASCIIClass) add(value byte) {
	class[value>>6] |= uint64(1) << (value & 63)
}

func (class fixedASCIIClass) matches(value byte) bool {
	return value < 0x80 &&
		class[value>>6]&(uint64(1)<<(value&63)) != 0
}

func (class fixedASCIIClass) intersects(other fixedASCIIClass) bool {
	for index := range class {
		if class[index]&other[index] != 0 {
			return true
		}
	}
	return false
}

// fixedASCIIRegexp is a conservative execution plan for non-empty regular
// expressions composed only of ASCII literals, character classes,
// concatenation, capture, and alternation.
type fixedASCIIRegexp struct {
	alternatives [][]fixedASCIIClass
}

const fixedASCIIRegexpPlanLimit = 256

var fixedASCIIRegexpPlans = struct {
	sync.Mutex
	entries map[string]*fixedASCIIRegexp
	order   []string
}{
	entries: make(map[string]*fixedASCIIRegexp),
}

// fixedASCIIRegexpPlan returns a plan only when leftmost-first and
// leftmost-longest matching select the same byte range. This keeps the plan
// valid even if callers use CompilePOSIX or call Regexp.Longest after caching.
func fixedASCIIRegexpPlan(expression *regexp.Regexp) *fixedASCIIRegexp {
	key := expression.String()
	fixedASCIIRegexpPlans.Lock()
	defer fixedASCIIRegexpPlans.Unlock()
	if plan, ok := fixedASCIIRegexpPlans.entries[key]; ok {
		return plan
	}

	parsed, err := syntax.Parse(key, syntax.Perl)
	if err != nil {
		cacheFixedASCIIRegexpPlan(key, nil)
		return nil
	}
	alternatives, ok := expandFixedASCIIRegexp(parsed.Simplify())
	if !ok || len(alternatives) == 0 || len(alternatives) > 64 {
		cacheFixedASCIIRegexpPlan(key, nil)
		return nil
	}
	for _, alternative := range alternatives {
		if len(alternative) == 0 || len(alternative) > 128 {
			cacheFixedASCIIRegexpPlan(key, nil)
			return nil
		}
	}
	for left := range alternatives {
		for right := left + 1; right < len(alternatives); right++ {
			if len(alternatives[left]) != len(alternatives[right]) &&
				alternatives[left][0].intersects(alternatives[right][0]) {
				cacheFixedASCIIRegexpPlan(key, nil)
				return nil
			}
		}
	}

	plan := &fixedASCIIRegexp{alternatives: alternatives}
	cacheFixedASCIIRegexpPlan(key, plan)
	return plan
}

// cacheFixedASCIIRegexpPlan must be called with fixedASCIIRegexpPlans locked.
func cacheFixedASCIIRegexpPlan(key string, plan *fixedASCIIRegexp) {
	if len(fixedASCIIRegexpPlans.order) == fixedASCIIRegexpPlanLimit {
		delete(
			fixedASCIIRegexpPlans.entries,
			fixedASCIIRegexpPlans.order[0],
		)
		copy(
			fixedASCIIRegexpPlans.order,
			fixedASCIIRegexpPlans.order[1:],
		)
		fixedASCIIRegexpPlans.order =
			fixedASCIIRegexpPlans.order[:fixedASCIIRegexpPlanLimit-1]
	}
	fixedASCIIRegexpPlans.entries[key] = plan
	fixedASCIIRegexpPlans.order = append(fixedASCIIRegexpPlans.order, key)
}

func expandFixedASCIIRegexp(
	expression *syntax.Regexp,
) ([][]fixedASCIIClass, bool) {
	switch expression.Op {
	case syntax.OpEmptyMatch:
		return [][]fixedASCIIClass{{}}, true
	case syntax.OpCapture:
		return expandFixedASCIIRegexp(expression.Sub[0])
	case syntax.OpLiteral:
		if expression.Flags&syntax.FoldCase != 0 {
			return nil, false
		}
		result := make([]fixedASCIIClass, len(expression.Rune))
		for index, value := range expression.Rune {
			if value < 0 || value > 0x7f {
				return nil, false
			}
			result[index].add(byte(value))
		}
		return [][]fixedASCIIClass{result}, true
	case syntax.OpCharClass:
		var class fixedASCIIClass
		for index := 0; index < len(expression.Rune); index += 2 {
			low, high := expression.Rune[index], expression.Rune[index+1]
			if low < 0 || high > 0x7f {
				return nil, false
			}
			for value := low; value <= high; value++ {
				class.add(byte(value))
			}
		}
		return [][]fixedASCIIClass{{class}}, true
	case syntax.OpAlternate:
		var result [][]fixedASCIIClass
		for _, subexpression := range expression.Sub {
			alternatives, ok := expandFixedASCIIRegexp(subexpression)
			if !ok || len(result)+len(alternatives) > 64 {
				return nil, false
			}
			result = append(result, alternatives...)
		}
		return result, true
	case syntax.OpConcat:
		result := [][]fixedASCIIClass{{}}
		for _, subexpression := range expression.Sub {
			parts, ok := expandFixedASCIIRegexp(subexpression)
			if !ok || len(parts) == 0 || len(result)*len(parts) > 64 {
				return nil, false
			}
			combined := make(
				[][]fixedASCIIClass,
				0,
				len(result)*len(parts),
			)
			for _, prefix := range result {
				for _, suffix := range parts {
					value := make(
						[]fixedASCIIClass,
						0,
						len(prefix)+len(suffix),
					)
					value = append(value, prefix...)
					value = append(value, suffix...)
					combined = append(combined, value)
				}
			}
			result = combined
		}
		return result, true
	default:
		return nil, false
	}
}

func (plan *fixedASCIIRegexp) find(
	input string,
	start int,
) (matchStart, matchEnd int, ok bool) {
	for position := start; position < len(input); position++ {
		for _, alternative := range plan.alternatives {
			if position+len(alternative) > len(input) {
				continue
			}
			matched := true
			for offset, class := range alternative {
				if !class.matches(input[position+offset]) {
					matched = false
					break
				}
			}
			if matched {
				return position, position + len(alternative), true
			}
		}
	}
	return 0, 0, false
}

func (plan *fixedASCIIRegexp) count(input string, start int) int {
	count := 0
	for start <= len(input) {
		_, end, ok := plan.find(input, start)
		if !ok {
			return count
		}
		count++
		start = end
	}
	return count
}

// ReplaceRegexpAllString uses a fixed-width ASCII plan when replacement is
// literal, and otherwise preserves regexp.Regexp's full replacement
// semantics.
func ReplaceRegexpAllString(
	expression *regexp.Regexp,
	input string,
	replacement string,
) string {
	plan := fixedASCIIRegexpPlan(expression)
	if plan == nil || strings.Contains(replacement, "$") {
		return expression.ReplaceAllString(input, replacement)
	}

	var builder strings.Builder
	builder.Grow(len(input))
	written := 0
	search := 0
	for {
		start, end, ok := plan.find(input, search)
		if !ok {
			if written == 0 {
				return input
			}
			builder.WriteString(input[written:])
			return builder.String()
		}
		builder.WriteString(input[written:start])
		builder.WriteString(replacement)
		written = end
		search = end
	}
}
