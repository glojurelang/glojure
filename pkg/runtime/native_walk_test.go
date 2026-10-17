//go:build !glj_no_aot_stdlib

package runtime_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/glj"
	"github.com/glojurelang/glojure/pkg/lang"
	_ "github.com/glojurelang/glojure/pkg/stdlib/clojure/walk"
)

func referenceTreeWalk(fn, form any, pre bool) any {
	outer := fn
	if pre {
		outer = glj.Var("clojure.core", "identity").(*lang.Var).Get()
		form = lang.Apply1(fn, form)
	}
	inner := lang.FnFunc1(func(child any) any {
		return referenceTreeWalk(fn, child, pre)
	})
	return lang.Apply3(glj.Var("clojure.walk", "walk").(*lang.Var).Get(), inner, outer, form)
}

func walkTrace(value any) string {
	meta := any(nil)
	if object, ok := value.(lang.IMeta); ok {
		meta = object.Meta()
	}
	_, entry := value.(lang.IMapEntry)
	_, seq := value.(lang.ISeq)
	return fmt.Sprintf("%v/%v %s meta=%s", entry, seq,
		lang.PrintString(value), lang.PrintString(meta))
}

func TestNativeTreeWalkMatchesOriginalTraversal(t *testing.T) {
	glj.Var("clojure.core", "require").Invoke(lang.NewSymbol("clojure.walk"))
	for _, pre := range []bool{false, true} {
		name := "postwalk"
		if pre {
			name = "prewalk"
		}
		for _, code := range []string{
			`[1 {:a [2 3]} (4 5) #{6 7} nil true "text" :key symbol]`,
			`{[1 2] {:a 3} :b [4 {:c 5}]}`,
			`^{:tag :root} [^{:tag :child} [1 2] ^{:tag :list} (3 4)]`,
			`^{:tag :root} {:a ^{:tag :child} [1 2]}`,
		} {
			t.Run(name+"/"+code, func(t *testing.T) {
				var referenceEvents, nativeEvents []string
				transform := func(events *[]string) lang.FnFunc1 {
					return func(value any) any {
						*events = append(*events, walkTrace(value))
						if n, ok := value.(int64); ok {
							return n + 10
						}
						return value
					}
				}
				want := referenceTreeWalk(transform(&referenceEvents), glj.Read(code), pre)
				got := glj.Var("clojure.walk", name).Invoke(
					transform(&nativeEvents), glj.Read(code))
				if !lang.Equals(got, want) || !reflect.DeepEqual(nativeEvents, referenceEvents) {
					t.Fatalf("%s mismatch:\ngot %v\nwant %v\nevents %v\nwant %v",
						name, got, want, nativeEvents, referenceEvents)
				}
			})
		}
	}
}

func TestNativePrewalkVisitsReplacementChildren(t *testing.T) {
	glj.Var("clojure.core", "require").Invoke(lang.NewSymbol("clojure.walk"))
	fn := lang.FnFunc1(func(value any) any {
		if value == lang.NewKeyword("expand") {
			return lang.NewVector(int64(1), int64(2))
		}
		if n, ok := value.(int64); ok {
			return n + 10
		}
		return value
	})
	got := glj.Var("clojure.walk", "prewalk").Invoke(fn, lang.NewKeyword("expand"))
	if !lang.Equals(got, lang.NewVector(int64(11), int64(12))) {
		t.Fatalf("prewalk replacement = %v", got)
	}
}

func TestNativeTreeWalkHonorsWalkRedefinition(t *testing.T) {
	glj.Var("clojure.core", "require").Invoke(lang.NewSymbol("clojure.walk"))
	walk := glj.Var("clojure.walk", "walk").(*lang.Var)
	before := walk.Get()
	defer walk.BindRoot(before)
	walk.BindRoot(lang.FnFunc3(func(inner, outer, form any) any { return "redefined" }))
	identity := lang.FnFunc1(func(value any) any { return value })
	for _, name := range []string{"prewalk", "postwalk"} {
		if got := glj.Var("clojure.walk", name).Invoke(identity, lang.NewVector(1)); got != "redefined" {
			t.Fatalf("%s ignored walk redefinition: %v", name, got)
		}
	}
}
