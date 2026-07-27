package lang

import "strings"

func ConcatStrings(strs ...string) string {
	b := strings.Builder{}
	for _, str := range strs {
		b.WriteString(str)
	}
	return b.String()
}

// ReverseString reverses Unicode code points. Glojure strings use byte
// collection semantics, while text operations such as clojure.string/reverse
// explicitly operate on characters.
func ReverseString(s string) string {
	runes := []rune(s)
	for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
		runes[left], runes[right] = runes[right], runes[left]
	}
	return string(runes)
}
