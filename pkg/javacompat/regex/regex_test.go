package regex

import "testing"

func TestReplaceAll(t *testing.T) {
	pattern := Compile(`\x1b\[.*?m`)
	if !IsPattern(pattern) {
		t.Fatal("compiled value is not recognized as a Pattern")
	}
	if got := ReplaceAll(pattern, "a\x1b[31mb", ""); got != "ab" {
		t.Fatalf("ReplaceAll = %q, want ab", got)
	}
}

func TestCompileJavaEscape(t *testing.T) {
	pattern := Compile(`\e\[.*?m`)
	matcher := pattern.Matcher("\x1b[31m")
	if !matcher.Find() {
		t.Fatal("translated Java escape pattern did not match")
	}
}
