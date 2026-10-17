package runtime

import "github.com/glojurelang/glojure/pkg/lang"

func nativeVec(coll any) any {
	if _, vector := coll.(lang.IPersistentVector); vector {
		if object, ok := coll.(lang.IObj); ok {
			return object.WithMeta(nil)
		}
	}
	return lang.CreateLazilyPersistentVector(coll)
}

func installNativeVectorMetadata(core *lang.Namespace) {
	if vr := core.FindInternedVar(lang.NewSymbol("into")); vr != nil {
		original := vr.Get()
		installFixedArityCoreFunction(core, "into", 2, lang.FnFunc2(func(to, from any) any {
			if result, ok := to.(*lang.Vector); ok {
				if source, ok := from.(*lang.Vector); ok && source.Count() > 0 && source.Count() <= 4 {
					for i := 0; i < source.Count(); i++ {
						result = result.Cons(source.Nth(i)).(*lang.Vector)
					}
					return result
				}
			}
			return lang.Apply2(original, to, from)
		}))
	}

	if vr := core.FindInternedVar(lang.NewSymbol("vec")); vr != nil {
		vr.BindRoot(lang.FnFunc1(nativeVec))
	}
	if vr := core.FindInternedVar(lang.NewSymbol("with-meta")); vr != nil {
		original := vr.Get()
		vr.BindRoot(lang.FnFunc2(func(value, metadata any) any {
			object, objectOK := value.(lang.IObj)
			meta, metaOK := metadata.(lang.IPersistentMap)
			if lang.IsNil(metadata) {
				metadata, meta, metaOK = nil, nil, true
			}
			if objectOK && (metadata == nil || metaOK) {
				return object.WithMeta(meta)
			}
			return lang.Apply2(original, value, metadata)
		}))
	}
}
