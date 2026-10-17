package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// treeWalker shares one recursive visitor across an entire tree. Common
// collections need neither per-node partial functions nor lazy map pipelines.
type treeWalker struct {
	fn          any
	walk        *lang.Var
	walkVersion *lang.VarRootVersion
	identity    *lang.Var
	pre         bool
}

func installNativeWalkFunctions() {
	ns := lang.FindNamespace(lang.NewSymbol("clojure.walk"))
	if ns == nil {
		return
	}
	walk := ns.FindInternedVar(lang.NewSymbol("walk"))
	if walk == nil {
		return
	}
	version := walk.RootVersion()
	identity := lang.FindNamespace(lang.NewSymbol("clojure.core")).
		FindInternedVar(lang.NewSymbol("identity"))
	for _, name := range []string{"prewalk", "postwalk"} {
		if vr := ns.FindInternedVar(lang.NewSymbol(name)); vr != nil {
			pre := name == "prewalk"
			vr.BindRoot(lang.FnFunc2(func(fn, form any) any {
				visitor := &treeWalker{fn: fn, walk: walk, walkVersion: version,
					identity: identity, pre: pre}
				return visitor.Invoke1(form)
			}))
		}
	}
}

func (*treeWalker) IsFnValue() {}

func (*treeWalker) Meta() lang.IPersistentMap { return nil }

func (p *treeWalker) WithMeta(meta lang.IPersistentMap) any {
	return lang.NewMetaFn(p, meta)
}

func (p *treeWalker) Invoke(args ...any) any {
	if len(args) != 1 {
		panic(lang.NewIllegalArgumentError("walk visitor expects one argument"))
	}
	return p.Invoke1(args[0])
}

func (p *treeWalker) ApplyTo(args lang.ISeq) any {
	return p.Invoke(lang.ToSlice(args)...)
}

func (p *treeWalker) Invoke1(form any) any {
	outer := p.fn
	if p.pre {
		outer = p.identity.Get()
		form = lang.Apply1(p.fn, form)
	}
	if p.walk.RootVersion() != p.walkVersion {
		return lang.Apply3(p.walk.Get(), p, outer, form)
	}
	var walked any
	switch value := form.(type) {
	case *lang.Vector:
		values := make([]any, value.Count())
		for i := range values {
			values[i] = p.Invoke1(value.Nth(i))
		}
		vector := lang.NewVector(values...)
		if meta := value.Meta(); meta != nil {
			vector = vector.WithMeta(meta).(*lang.Vector)
		}
		walked = vector
	case *lang.Map:
		var result lang.IPersistentCollection = value.Empty()
		walked = value.KVReduce(lang.FnFunc3(func(acc, key, val any) any {
			entry := p.Invoke1(lang.NewMapEntry(key, val))
			return acc.(lang.IPersistentCollection).Cons(entry)
		}), result)
	case *lang.MapEntry:
		key := p.Invoke1(value.Key())
		val := p.Invoke1(value.Val())
		walked = lang.NewMapEntry(key, val)
	case nil, bool, string, int, int64, float64, lang.Keyword, lang.Char, *lang.Symbol:
		walked = form
	default:
		// Lists, seqs, records and specialized collections retain walk's
		// original reconstruction and metadata behavior.
		return lang.Apply3(p.walk.Get(), p, outer, form)
	}
	return lang.Apply1(outer, walked)
}
