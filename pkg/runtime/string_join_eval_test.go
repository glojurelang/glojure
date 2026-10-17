//go:build !glj_aot_runtime

package runtime

import (
	"testing"
)

func TestNativeStringJoinValues(t *testing.T) {
	ReadEval(`(require 'clojure.string)`)
	for _, test := range []struct{ code, want string }{
		{`(clojure.string/join ["a" nil 1 :key "λ" \a])`, `a1:keyλa`},
		{`(clojure.string/join "-" ["a" nil 1 :key "λ" \a])`, `a--1-:key-λ-a`},
		{`(clojure.string/join nil ["a" "b"])`, `ab`},
		{`(clojure.string/join :sep [])`, ``},
		{`(clojure.string/join "," nil)`, ``},
	} {
		if got := ReadEval(test.code); got != test.want {
			t.Fatalf("%s = %s, want %s", test.code, got, test.want)
		}
	}
}
