//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"sort"
	"strings"
)

func (g *Generator) aotRecordHasField(keyword string) bool {
	for _, record := range g.aotRecordTypes {
		for _, field := range record.fieldNames {
			if field == keyword {
				return true
			}
		}
	}
	return false
}

func (g *Generator) aotRecordHasFields(keywords []string) bool {
	for _, record := range g.aotRecordTypes {
		matched := true
		for _, keyword := range keywords {
			found := false
			for _, field := range record.fieldNames {
				if field == keyword {
					found = true
					break
				}
			}
			if !found {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func (g *Generator) allocKeywordLookupHelper(keyword string) string {
	if helper := g.keywordLookupHelpers[keyword]; helper != nil {
		return helper.name
	}
	helper := &aotKeywordLookupHelper{
		name:    fmt.Sprintf("aotKeywordLookup%d", len(g.keywordLookupHelpers)),
		keyword: keyword,
	}
	g.keywordLookupHelpers[keyword] = helper
	return helper.name
}

func (g *Generator) allocKeywordAssocHelper(keywords []string) string {
	key := strings.Join(keywords, "\x00")
	if helper := g.keywordAssocHelpers[key]; helper != nil {
		return helper.name
	}
	helper := &aotKeywordAssocHelper{
		name:     fmt.Sprintf("aotKeywordAssoc%d", len(g.keywordAssocHelpers)),
		keywords: append([]string(nil), keywords...),
	}
	g.keywordAssocHelpers[key] = helper
	return helper.name
}

func (g *Generator) generateAOTKeywordHelpers() {
	lookups := make([]*aotKeywordLookupHelper, 0, len(g.keywordLookupHelpers))
	for _, helper := range g.keywordLookupHelpers {
		lookups = append(lookups, helper)
	}
	sort.Slice(lookups, func(i, j int) bool {
		return lookups[i].name < lookups[j].name
	})
	for _, helper := range lookups {
		keyVar := helper.name + "Key"
		fmt.Fprintf(&g.aotDeclarations,
			"\nvar %s = lang.NewKeyword(%q)\n", keyVar, helper.keyword)
		fmt.Fprintf(&g.aotDeclarations,
			"func %s(value, fallback any) any {\n"+
				"\tswitch value := value.(type) {\n",
			helper.name)
		for _, record := range g.sortedAOTRecordTypes() {
			for field, name := range record.fieldNames {
				if name == helper.keyword {
					fmt.Fprintf(&g.aotDeclarations,
						"\tcase *%s: return value.f%d\n",
						record.typeName, field)
					break
				}
			}
		}
		fmt.Fprintf(&g.aotDeclarations,
			"\tdefault: return %s.Invoke2(value, fallback)\n"+
				"\t}\n}\n",
			keyVar)
	}

	assocs := make([]*aotKeywordAssocHelper, 0, len(g.keywordAssocHelpers))
	for _, helper := range g.keywordAssocHelpers {
		assocs = append(assocs, helper)
	}
	sort.Slice(assocs, func(i, j int) bool {
		return assocs[i].name < assocs[j].name
	})
	for _, helper := range assocs {
		g.generateAOTKeywordAssocHelper(helper)
	}
}

func (g *Generator) generateAOTKeywordAssocHelper(
	helper *aotKeywordAssocHelper,
) {
	keyVars := make([]string, len(helper.keywords))
	params := []string{"value any"}
	for i, keyword := range helper.keywords {
		keyVars[i] = fmt.Sprintf("%sKey%d", helper.name, i)
		params = append(params, fmt.Sprintf("v%d any", i))
		fmt.Fprintf(&g.aotDeclarations,
			"\nvar %s = lang.NewKeyword(%q)\n", keyVars[i], keyword)
	}
	fmt.Fprintf(&g.aotDeclarations,
		"func %s(%s) any {\n\tswitch value := value.(type) {\n",
		helper.name, strings.Join(params, ", "))
	for _, record := range g.sortedAOTRecordTypes() {
		indices := make([]int, len(helper.keywords))
		matched := true
		for i, keyword := range helper.keywords {
			indices[i] = -1
			for field, name := range record.fieldNames {
				if name == keyword {
					indices[i] = field
					break
				}
			}
			if indices[i] < 0 {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		fmt.Fprintf(&g.aotDeclarations,
			"\tcase *%s:\n\t\tresult := *value\n", record.typeName)
		for i, field := range indices {
			fmt.Fprintf(&g.aotDeclarations,
				"\t\tresult.f%d = v%d\n", field, i)
		}
		fmt.Fprintln(&g.aotDeclarations,
			"\t\tresult.hash = 0\n\t\tresult.hasheq = 0\n\t\treturn &result")
	}
	fmt.Fprintln(&g.aotDeclarations, "\tdefault:")
	fmt.Fprintln(&g.aotDeclarations, "\t\tvar result any = value")
	for i := range helper.keywords {
		fmt.Fprintf(&g.aotDeclarations,
			"\t\tresult = lang.Assoc(result, %s, v%d)\n", keyVars[i], i)
	}
	fmt.Fprintln(&g.aotDeclarations, "\t\treturn result\n\t}\n}")
}
