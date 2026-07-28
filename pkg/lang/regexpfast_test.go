package lang

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestFixedASCIIRegexpPlanMatchesStandardRegexp(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
	}{
		{`agggtaaa|tttaccct`, "xxagggtaaayytttaccctzz"},
		{`[cgt]gggtaaa|tttaccc[acg]`, "cgggtaaa-tttaccca"},
		{`a[act]ggtaaa|tttacc[agt]t`, "aaggtaaa tttaccatt"},
		{`aND|caN|Ha[DS]|WaS`, "aNDcaNHaDHaSWaS"},
		{`a[NSt]|BY`, "aNtaStaStBY"},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			expression := regexp.MustCompile(test.pattern)
			plan := fixedASCIIRegexpPlan(expression)
			if plan == nil {
				t.Fatalf("fixed ASCII pattern %q was not planned", test.pattern)
			}

			var got [][2]int
			for start := 0; start <= len(test.input); {
				matchStart, matchEnd, ok := plan.find(test.input, start)
				if !ok {
					break
				}
				got = append(got, [2]int{matchStart, matchEnd})
				start = matchEnd
			}
			standard := expression.FindAllStringIndex(test.input, -1)
			want := make([][2]int, len(standard))
			for index, match := range standard {
				want[index] = [2]int{match[0], match[1]}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("fixed matches = %v, want %v", got, want)
			}
		})
	}
}

func TestFixedASCIIRegexpPlanRejectsUnsafeShapes(t *testing.T) {
	for _, pattern := range []string{
		`a+`,
		`a|aa`,
		`(?i:abc)`,
		`café`,
		`[^a]`,
		`a*`,
		`^abc`,
	} {
		t.Run(pattern, func(t *testing.T) {
			if plan := fixedASCIIRegexpPlan(regexp.MustCompile(pattern)); plan != nil {
				t.Fatalf("unsafe pattern %q produced a fixed plan", pattern)
			}
		})
	}
}

func TestReplaceRegexpAllStringUsesEquivalentFixedPlan(t *testing.T) {
	tests := []struct {
		pattern     string
		input       string
		replacement string
	}{
		{`tHa[Nt]`, "tHaN-tHat-x", "<4>"},
		{`aND|caN|Ha[DS]|WaS`, "aNDcaNHaDHaSWaS", "<3>"},
		{`a[NSt]|BY`, "aNtaStaStBY", "<2>"},
		{`(ab|cd)`, "xxabyycdzz", "_"},
		{`a+`, "caaab", "_"},
		{`(ab)`, "xxabyy", "$1$1"},
	}

	for _, test := range tests {
		t.Run(test.pattern+"/"+test.replacement, func(t *testing.T) {
			expression := regexp.MustCompile(test.pattern)
			want := expression.ReplaceAllString(test.input, test.replacement)
			if got := ReplaceRegexpAllString(
				expression,
				test.input,
				test.replacement,
			); got != want {
				t.Fatalf("replacement = %q, want %q", got, want)
			}
		})
	}
}

func TestFixedASCIIRegexpPlanRemainsValidAfterLongest(t *testing.T) {
	sameWidth := regexp.MustCompile(`ab|ac`)
	plan := fixedASCIIRegexpPlan(sameWidth)
	if plan == nil {
		t.Fatal("same-width alternation was not planned")
	}
	sameWidth.Longest()
	start, end, ok := plan.find("xac", 0)
	if !ok || start != 1 || end != 3 {
		t.Fatalf("longest fixed match = (%d, %d, %v), want (1, 3, true)",
			start, end, ok)
	}

	varyingWidth := regexp.MustCompile(`a|aa`)
	if plan := fixedASCIIRegexpPlan(varyingWidth); plan != nil {
		t.Fatal("ambiguous varying-width pattern was planned")
	}
}

func FuzzFixedASCIIRegexpPlanMatchesStandard(f *testing.F) {
	for _, seed := range []struct {
		pattern string
		input   string
	}{
		{`ab|cd`, "xxabyycdzz"},
		{`[a-c]x|[d-f]yz`, "éaxfyz"},
		{`a[0-9]z`, "a1z-a9z"},
		{`a|aa`, "aaaa"},
		{`[^a]`, "ba"},
	} {
		f.Add(seed.pattern, seed.input)
	}

	f.Fuzz(func(t *testing.T, pattern, input string) {
		expression, err := regexp.Compile(pattern)
		if err != nil {
			return
		}
		plan := fixedASCIIRegexpPlan(expression)
		if plan == nil {
			return
		}
		for search := 0; search <= len(input); search++ {
			standard := expression.FindStringIndex(input[search:])
			start, end, ok := plan.find(input, search)
			if standard == nil {
				if ok {
					t.Fatalf(
						"plan found [%d %d] from %d; regexp found none",
						start, end, search,
					)
				}
				continue
			}
			wantStart := search + standard[0]
			wantEnd := search + standard[1]
			if !ok || start != wantStart || end != wantEnd {
				t.Fatalf(
					"plan found [%d %d] (%v) from %d; want [%d %d]",
					start, end, ok, search, wantStart, wantEnd,
				)
			}
		}
	})
}

func FuzzReplaceRegexpAllStringMatchesStandard(f *testing.F) {
	f.Add(`ab|cd`, "xxabyycdzz", "_")
	f.Add(`a[NSt]|BY`, "aNtaStaStBY", "<2>")
	f.Add(`(ab)`, "xxabyy", "$1$1")
	f.Add(`a+`, "caaab", "_")

	f.Fuzz(func(t *testing.T, pattern, input, replacement string) {
		expression, err := regexp.Compile(pattern)
		if err != nil || strings.Contains(replacement, "$") {
			return
		}
		if fixedASCIIRegexpPlan(expression) == nil {
			return
		}
		want := expression.ReplaceAllString(input, replacement)
		if got := ReplaceRegexpAllString(
			expression,
			input,
			replacement,
		); got != want {
			t.Fatalf("fixed replacement = %q, want %q", got, want)
		}
	})
}
