package lang

import "strings"

func ConcatStrings(strs ...string) string {
	b := strings.Builder{}
	for _, str := range strs {
		b.WriteString(str)
	}
	return b.String()
}
