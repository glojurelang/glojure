package runtime

import "github.com/glojurelang/glojure/pkg/lang"

func directLinkEnabled() bool {
	compilerOptions := lang.NSCore.FindInternedVar(
		lang.NewSymbol("*compiler-options*"),
	)
	if compilerOptions == nil || !compilerOptions.IsBound() {
		return true
	}
	options := compilerOptions.Get()
	missing := &struct{}{}
	value := lang.GetDefault(
		options,
		lang.KWDirectLinking,
		missing,
	)
	if value == missing {
		return true
	}
	return RT.BooleanCast(value)
}
