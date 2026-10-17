package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// Binding one leading argument needs one function object rather than a
// closure for every arity plus a multi-arity dispatch table.
type partialOne struct {
	fn, first any
	apply     *lang.Var
}

func (*partialOne) IsFnValue() {}

func (*partialOne) Meta() lang.IPersistentMap { return nil }

func (p *partialOne) WithMeta(meta lang.IPersistentMap) any {
	return lang.NewMetaFn(p, meta)
}

func (p *partialOne) Invoke0() any { return lang.Apply1(p.fn, p.first) }

func (p *partialOne) Invoke1(a any) any { return lang.Apply2(p.fn, p.first, a) }

func (p *partialOne) Invoke2(a, b any) any {
	return lang.Apply3(p.fn, p.first, a, b)
}

func (p *partialOne) Invoke3(a, b, c any) any {
	return lang.Apply4(p.fn, p.first, a, b, c)
}

func (p *partialOne) Invoke(args ...any) any {
	switch len(args) {
	case 0:
		return p.Invoke0()
	case 1:
		return p.Invoke1(args[0])
	case 2:
		return p.Invoke2(args[0], args[1])
	case 3:
		return p.Invoke3(args[0], args[1], args[2])
	default:
		return lang.Apply(p.apply.Get(), []any{
			p.fn, p.first, args[0], args[1], args[2], lang.NewSliceSeq(args[3:]),
		})
	}
}

func (p *partialOne) ApplyTo(args lang.ISeq) any {
	// Pull only the fixed prefix, preserving lazy variadic argument tails.
	var prefix [3]any
	for i := range prefix {
		if args == nil {
			return p.Invoke(prefix[:i]...)
		}
		prefix[i] = args.First()
		args = args.Next()
	}
	if args == nil {
		return p.Invoke3(prefix[0], prefix[1], prefix[2])
	}
	return lang.Apply(p.apply.Get(), []any{
		p.fn, p.first, prefix[0], prefix[1], prefix[2], args,
	})
}
