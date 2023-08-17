package core

import (
	"os"

	"github.com/glojurelang/glojure/pkg/lang"
)

var (
	ns = lang.FindOrCreateNamespace(lang.NewSymbol("glojure.core"))
)

func init() {
	ns.ResetMeta(lang.NewPersistentArrayMap(
		lang.NewKeyword("doc"), "Fundamental library of the Clojure language",
	))

	{
		vr := ns.Intern(lang.NewSymbol("normalize-slurp-opts"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-open"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-meta"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("peek"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("merge-hash-collisions"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("butlast"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(277),
			lang.NewKeyword("column"), int(10),
			lang.NewKeyword("end-line"), int(281),
			lang.NewKeyword("end-column"), int(27),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("find"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("="))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rand-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("fnext"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(117),
			lang.NewKeyword("column"), int(8),
			lang.NewKeyword("end-line"), int(117),
			lang.NewKeyword("end-column"), int(47),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("error-handler"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("zero?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("use"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("amap"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-unalias"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("def-aset"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunked-seq?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("some-fn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("parse-double"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-data-reader-file"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("short"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("disj"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunk-first"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("NaN?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("partitionv-all"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("error-mode"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bigdec"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("fits-table?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("transient"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("StackTraceElement->vec"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("conj"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(82),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(89),
			lang.NewKeyword("end-column"), int(67),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("remove-watch"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ensure"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("+'"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("await"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nary-inline"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("array-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("long"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("filter"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sorted-set-by"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-resolve"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("prependss"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("add-watch"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("conj!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("re-matches"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("read-line"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-bindings*"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("memfn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("inst?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*e"))
		_ = vr
	}
	{
		vr := ns.Intern(lang.NewSymbol("tap>"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("set?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("simple-keyword?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dotimes"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*default-data-reader-fn*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("*in*"))
		vr.BindRoot(os.Stdin)
	}
	{
		vr := ns.Intern(lang.NewSymbol("disj!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-object"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("all-ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("not"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("biginteger"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("uuid?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol(".."))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("not-every?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(2560),
			lang.NewKeyword("column"), int(6),
			lang.NewKeyword("end-line"), int(2565),
			lang.NewKeyword("end-column"), int(49),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("system-newline"))
		vr.BindRoot("\n")
	}
	{
		vr := ns.Intern(lang.NewSymbol("remove"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("maybe-min-hash"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rem"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-ctor"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ex-info"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-boolean"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("binding-conveyor-fn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("get-thread-bindings"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("re-groups"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("await-for"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("mix-collection-hash"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("distinct"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pr-on"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("seq-to-map-for-destructuring"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("take-while"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("restart-agent"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("println-str"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("when-some"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("shutdown-agents"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("uri?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pop"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rsubseq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("io!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*data-readers*"))
		vr.BindRoot(lang.NewPersistentArrayMap())
	}
	{
		vr := ns.Intern(lang.NewSymbol("clear-agent-errors"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("re-find"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-subtract"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("is-annotation?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("assoc"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(190),
			lang.NewKeyword("column"), int(2),
			lang.NewKeyword("end-line"), int(199),
			lang.NewKeyword("end-column"), int(15),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*3"))
		_ = vr
	}
	{
		vr := ns.Intern(lang.NewSymbol("coll?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("printf"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("tapset"))
		vr.BindRoot(lang.NewAtom(lang.NewPersistentHashSet()))
	}
	{
		vr := ns.Intern(lang.NewSymbol("name"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("iteration"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("count"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol(">0?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reduced?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-multiply-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sync"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("keyword"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("number?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-throwable"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("read-string"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("identity"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("split-at"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("first"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(53),
			lang.NewKeyword("column"), int(8),
			lang.NewKeyword("end-line"), int(53),
			lang.NewKeyword("end-column"), int(86),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("qualified-symbol?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("var?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("if-some"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("hash-unordered-coll"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-data-readers"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rest"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(71),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(71),
			lang.NewKeyword("end-column"), int(77),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("PrintWriter-on"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("map-indexed"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("mapcat"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sigs"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(226),
			lang.NewKeyword("column"), int(2),
			lang.NewKeyword("end-line"), int(259),
			lang.NewKeyword("end-column"), int(43),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunk-buffer"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("contains?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("thread-bound?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("struct-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("distinct?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("keep-indexed"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*print-dup*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("reduce1"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ensure-reduced"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("data-reader-urls"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("repeatedly"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("find-ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*print-meta*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("set-error-handler!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("add-classpath"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-unmap"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("even?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("booleans"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-negate-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-prefix-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("compile"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unquote"))
		_ = vr
	}
	{
		vr := ns.Intern(lang.NewSymbol("ref-set"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("identical?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("type"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("prefer-method"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sorted-set"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*out*"))
		vr.BindRoot(os.Stdout)
	}
	{
		vr := ns.Intern(lang.NewSymbol("get-in"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nfirst"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(110),
			lang.NewKeyword("column"), int(9),
			lang.NewKeyword("end-line"), int(110),
			lang.NewKeyword("end-column"), int(49),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("var-get"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rand-nth"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("class"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("seqable?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("future?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("denominator"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("when-let"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("completing"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("resolve"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("set-agent-send-off-executor!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nil?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*loading-verbosely*"))
		vr.BindRoot(false)
	}
	{
		vr := ns.Intern(lang.NewSymbol("vector?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(179),
			lang.NewKeyword("column"), int(10),
			lang.NewKeyword("end-line"), int(179),
			lang.NewKeyword("end-column"), int(106),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("doto"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*loaded-libs*"))
		vr.BindRoot(lang.NewRef(lang.NewPersistentHashSet(
			lang.WithMeta(lang.NewSymbol("glojure.core.protocols"), lang.NewPersistentArrayMap(
				lang.NewKeyword("file"), "glojure/protocols.glj",
				lang.NewKeyword("line"), int(9),
				lang.NewKeyword("column"), int(5),
				lang.NewKeyword("end-line"), int(9),
				lang.NewKeyword("end-column"), int(26),
			)),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("intern"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("prep-ints"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("persistent!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-simple"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bound-fn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("volatile!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("if-let"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("send-via"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("->>"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-char"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-or"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rational?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bigint"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-and-not"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("refer-glojure"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unreduced"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("str"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("shorts"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("delay"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("tagged-literal?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pop-thread-bindings"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unquote-splicing"))
		_ = vr
	}
	{
		vr := ns.Intern(lang.NewSymbol("*unchecked-math*"))
		vr.BindRoot(false)
	}
	{
		vr := ns.Intern(lang.NewSymbol("ref-max-history"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("as->"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-float"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("select-keys"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("longs"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("realized?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("float?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("interpose"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("get-validator"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("flush"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-all"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("byte-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sorted?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunk-rest"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("int?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("spit"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-add"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("float-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reduced"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("doubles"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bases"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("get-method"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("cycle"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("iterate"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("check-cyclic-dependency"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("create-struct"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("string?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(165),
			lang.NewKeyword("column"), int(10),
			lang.NewKeyword("end-line"), int(165),
			lang.NewKeyword("end-column"), int(58),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("strip-ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("defstruct"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reset-meta!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reset!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("add-doc-and-meta"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("eval"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-byte"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("swap-vals!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("comment"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("next"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(62),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(62),
			lang.NewKeyword("end-column"), int(77),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("quot"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-redefs-fn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("inc'"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("gensym"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rseq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("emit-extend-type"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("class?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chars"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-bindings"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("hash-ordered-coll"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("import"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("requiring-resolve"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("empty"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("vals"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-initialized"))
		vr.BindRoot(true)
	}
	{
		vr := ns.Intern(lang.NewSymbol("fn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("complement"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("remove-tap"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("prn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("parse-long"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("map-entry?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("cond"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reduce-kv"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("supers"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("update-vals"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("vreset!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("deref-as-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("cat"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ex-data"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("last"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(267),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(270),
			lang.NewKeyword("end-column"), int(21),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("newline"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("not="))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("char"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("read+string"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("root-resource"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("vector"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("gen-class"))
		_ = vr
	}
	{
		vr := ns.Intern(lang.NewSymbol("lazy-cat"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("->"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("Throwable->map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unsigned-bit-shift-right"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("extend-type"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("take"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*read-eval*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("trampoline"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("val"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("split-with"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("for"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("derive"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("future"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("cond->>"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pmap"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("defonce"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dorun"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("max-switch-table-size"))
		vr.BindRoot(int64(8192))
	}
	{
		vr := ns.Intern(lang.NewSymbol("assert"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("assoc!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("tagged-literal"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("assert-valid-fdecl"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pop!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("into-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("set-validator!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("char-name-string"))
		vr.BindRoot(lang.NewPersistentArrayMap(
			lang.NewChar('\n'), "newline",
			lang.NewChar('\t'), "tab",
			lang.NewChar(' '), "space",
			lang.NewChar('\b'), "backspace",
			lang.NewChar('\f'), "formfeed",
			lang.NewChar('\r'), "return",
		))
	}
	{
		vr := ns.Intern(lang.NewSymbol("char-escape-string"))
		vr.BindRoot(lang.NewPersistentArrayMap(
			lang.NewChar('\n'), "\\n",
			lang.NewChar('\t'), "\\t",
			lang.NewChar('\r'), "\\r",
			lang.NewChar('"'), "\\\"",
			lang.NewChar('\\'), "\\\\",
			lang.NewChar('\f'), "\\f",
			lang.NewChar('\b'), "\\b",
		))
	}
	{
		vr := ns.Intern(lang.NewSymbol("byte"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol(">1?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ints"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*print-length*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("reduce"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("qualified-ident?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("double?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("agent-error"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("is-runtime-annotation?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("float"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("alter-meta!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-name"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("find-keyword"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("time"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("partial"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*1"))
		_ = vr
	}
	{
		vr := ns.Intern(lang.NewSymbol("fnil"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-sequential"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunk-cons"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("setup-reference"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-negate"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("char?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(158),
			lang.NewKeyword("column"), int(8),
			lang.NewKeyword("end-line"), int(158),
			lang.NewKeyword("end-column"), int(54),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("fn?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("true?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ref-min-history"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("decimal?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("create-ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("symbol"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-shift-left"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("random-sample"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("subvec"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ffirst"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(103),
			lang.NewKeyword("column"), int(9),
			lang.NewKeyword("end-line"), int(103),
			lang.NewKeyword("end-column"), int(50),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("read"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("partition-by"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("num"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("check-valid-options"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dec"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("file-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-char"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("every?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunk-append"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("remove-ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("boolean"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("filter-key"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("deref"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reversible?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("partitionv"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("alter"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("require"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("case-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("hash-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("zipmap"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("cond->"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-out-str"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-loading-context"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("eduction"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ex-message"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-libs"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("replicate"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("prefers"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("-"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-meta"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(217),
			lang.NewKeyword("column"), int(12),
			lang.NewKeyword("end-line"), int(218),
			lang.NewKeyword("end-column"), int(32),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-byte"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("flatten"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("future-call"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pr"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("re-matcher"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("take-last"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sort-by"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("root-directory"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*pending-paths*"))
		vr.BindRoot(lang.WithMeta(lang.NewList(), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(5847),
			lang.NewKeyword("column"), int(19),
			lang.NewKeyword("end-line"), int(5847),
			lang.NewKeyword("end-column"), int(20),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*print-readably*"))
		vr.BindRoot(true)
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-test"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-add-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("declare"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("prn-str"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("send-off"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("swap!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ancestors"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("hash-set"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reset-vals!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("interleave"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-shift-right"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("symbol?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunk"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("remove-all-methods"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-refers"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("subseq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("protocol?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("await1"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-inc-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("to-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), "[Ljava.lang.Object;",
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("range"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("qualified-keyword?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("agent"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("release-pending-sends"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*print-namespace-maps*"))
		vr.BindRoot(false)
	}
	{
		vr := ns.Intern(lang.NewSymbol("double-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bound?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-inc"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("struct"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("libspec?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("boolean?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-str"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("filterv"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("min"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("lazy-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nthrest"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*command-line-args*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("descendants"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("mapv"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("doseq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("abs"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aclone"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("serialized-require"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-publics"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("alias"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("set-agent-send-executor!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("get"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("mk-bound-fn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("second"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(96),
			lang.NewKeyword("column"), int(9),
			lang.NewKeyword("end-line"), int(96),
			lang.NewKeyword("end-column"), int(49),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("delay?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*2"))
		_ = vr
	}
	{
		vr := ns.Intern(lang.NewSymbol("*warn-on-reflection*"))
		vr.BindRoot(false)
	}
	{
		vr := ns.Intern(lang.NewSymbol("boolean-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	ns.InternWithValue(lang.NewSymbol("list"), lang.NewList, true)
	{
		vr := ns.Intern(lang.NewSymbol("-'"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("replace"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("deref-future"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*agent*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("object-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("find-var"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-and"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ident?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("any?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("merge-with"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("if-not"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("extend-protocol"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-not"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("subs"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("alength"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dedupe"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bounded-count"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("neg-int?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("when-first"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("comp"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("namespace"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("format"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("vary-meta"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nth"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("prep-hashes"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("min-key"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("future-cancelled?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("long-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-remainder-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("to-array-2d"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), "[[Ljava.lang.Object;",
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-redefs"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("vec"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-aliases"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-long"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("=="))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*assert*"))
		vr.BindRoot(false)
	}
	{
		vr := ns.Intern(lang.NewSymbol("*verbose-defrecords*"))
		vr.BindRoot(false)
	}
	{
		vr := ns.Intern(lang.NewSymbol("simple-symbol?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("short-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dissoc!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ref"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("areduce"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("+"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("assoc-in"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("seque"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-file"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-dec"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("volatile?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("parse-uuid"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*ns*"))
		vr.BindRoot(lang.FindOrCreateNamespace(lang.NewSymbol("glojure.core")))
	}
	{
		vr := ns.Intern(lang.NewSymbol("resultset-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aget"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sequence"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("drop-while"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("chunk-next"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("list?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("global-hierarchy"))
		vr.BindRoot(lang.NewPersistentArrayMap(
			lang.NewKeyword("parents"), lang.NewPersistentArrayMap(),
			lang.NewKeyword("descendants"), lang.NewPersistentArrayMap(),
			lang.NewKeyword("ancestors"), lang.NewPersistentArrayMap(),
		))
	}
	{
		vr := ns.Intern(lang.NewSymbol("int-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("emit-extend-protocol"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-double"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("promise"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("associative?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("push-thread-bindings"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("comparator"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("partition-all"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*print-level*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("merge"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("take-nth"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("false?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sort"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("max-key"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("some"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("make-hierarchy"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("assert-args"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bytes?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("when-not"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("seq?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(151),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(151),
			lang.NewKeyword("end-column"), int(87),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("underive"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("vswap!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("set-error-mode!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("char-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("memoize"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("when"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ratio?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("slurp"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ifn?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("drop-last"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-subtract-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-short"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-clear"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("some->>"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("update-in"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("not-any?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(2560),
			lang.NewKeyword("column"), int(6),
			lang.NewKeyword("end-line"), int(2565),
			lang.NewKeyword("end-column"), int(49),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reader-conditional?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("xml-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("group-by"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("numerator"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("defn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(292),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(334),
			lang.NewKeyword("end-column"), int(58),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pos-int?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("special-symbol?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-interns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pos?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("repeat"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bound-fn*"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*'"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-multiply"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("<="))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("lift-ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nnext"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(124),
			lang.NewKeyword("column"), int(8),
			lang.NewKeyword("end-line"), int(124),
			lang.NewKeyword("end-column"), int(46),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ex-cause"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("counted?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-xor"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("re-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("remove-method"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("parse-boolean"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("max"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*compiler-options*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol(">"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-float"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("defmethod"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("elide-top-frames"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sorted-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reductions"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reverse"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ref-history-count"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*file*"))
		vr.BindRoot("NO_SOURCE_FILE")
	}
	{
		vr := ns.Intern(lang.NewSymbol("make-array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("re-pattern"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("into"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("list*"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-lib"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pvalues"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("macroexpand-1"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("methods"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*compile-path*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("keep"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("parse-impls"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("or"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("deliver"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("some?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-one"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("atom"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*err*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("enumeration-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("while"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("meta"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(207),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(209),
			lang.NewKeyword("end-column"), int(21),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dec'"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("into1"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("println"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("drop"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("the-ns"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("descriptor"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("process-annotation"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("/"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("binding"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("double"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*compile-files*"))
		vr.BindRoot(false)
	}
	{
		vr := ns.Intern(lang.NewSymbol("defmacro"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(452),
			lang.NewKeyword("column"), int(11),
			lang.NewKeyword("end-line"), int(488),
			lang.NewKeyword("end-column"), int(40),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("cons"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(27),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(27),
			lang.NewKeyword("end-column"), int(89),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("splitv-at"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("future-cancel"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("iterator-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("max-mask-bits"))
		vr.BindRoot(int64(13))
	}
	{
		vr := ns.Intern(lang.NewSymbol("var-set"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("update-keys"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("extend"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sorted-map-by"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("throw-if"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("hash"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("run!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("line-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("mod"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("add-annotations"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("sequential?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("case"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("instance?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(144),
			lang.NewKeyword("column"), int(12),
			lang.NewKeyword("end-line"), int(144),
			lang.NewKeyword("end-column"), int(85),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("isa?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("keys"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nat-int?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("destructure"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("loaded-libs"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-reader"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("defmulti"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("preserving-reduced"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-map"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("compare"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("add-tap"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("spread"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("future-done?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("inst-ms"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("add-annotation"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("accessor"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("agent-errors"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("emit-hinted-impl"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rand"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("shift-mask"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("integer?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("maybe-destructured"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("cast"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-method"))
		vr.BindRoot(lang.NewSymbol("todo"))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dissoc"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("map?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(172),
			lang.NewKeyword("column"), int(7),
			lang.NewKeyword("end-line"), int(172),
			lang.NewKeyword("end-column"), int(97),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("*flush-on-newline*"))
		vr.BindRoot(nil)
	}
	{
		vr := ns.Intern(lang.NewSymbol("defprotocol"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-dup"))
		vr.BindRoot(lang.NewSymbol("todo"))
	}
	{
		vr := ns.Intern(lang.NewSymbol("force"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-local-vars"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("constantly"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("nthnext"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("floats"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("neg?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("alter-var-root"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("file"), "glojure/core.glj",
			lang.NewKeyword("line"), int(137),
			lang.NewKeyword("column"), int(6),
			lang.NewKeyword("end-line"), int(137),
			lang.NewKeyword("end-column"), int(126),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("some->"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bytes"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("key"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("parsing-err"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("definline"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("frequencies"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("load-string"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-in-str"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("simple-ident?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("print-tagged-object"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("empty?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("indexed?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("aset-short"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-divide-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("loop"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("tree-seq"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("odd?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-double"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("partition"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("defn-"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-set"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("halt-when"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("reader-conditional"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("dosync"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("send"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("refer"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol(">="))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-dec-int"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("array"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("let"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("not-empty"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("and"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("shuffle"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("every-pred"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("compare-and-set!"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("with-precision"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("transduce"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("parents"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("random-uuid"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("juxt"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("unchecked-long"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("bit-flip"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("infinite?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pcalls"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("locking"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("update"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("letfn"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("commute"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("keyword?"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("-protocols"))
		vr.BindRoot(lang.NewAtom(lang.NewPersistentArrayMap(
			lang.WithMeta(lang.NewSymbol("CollReduce"), lang.NewPersistentArrayMap(
				lang.NewKeyword("file"), "glojure/protocols.glj",
				lang.NewKeyword("line"), int(13),
				lang.NewKeyword("column"), int(14),
				lang.NewKeyword("end-line"), int(13),
				lang.NewKeyword("end-column"), int(23),
			)), lang.NewAtom(lang.NewPersistentArrayMap(
				lang.NewKeyword("multis"), lang.NewPersistentArrayMap(
					lang.NewKeyword("coll-reduce"), lang.NewSymbol("todo"),
				),
				lang.NewKeyword("on-interface"), true,
				lang.NewKeyword("sigs"), lang.NewSymbol("todo"),
			)),
			lang.WithMeta(lang.NewSymbol("InternalReduce"), lang.NewPersistentArrayMap(
				lang.NewKeyword("file"), "glojure/protocols.glj",
				lang.NewKeyword("line"), int(19),
				lang.NewKeyword("column"), int(14),
				lang.NewKeyword("end-line"), int(19),
				lang.NewKeyword("end-column"), int(27),
			)), lang.NewAtom(lang.NewPersistentArrayMap(
				lang.NewKeyword("multis"), lang.NewPersistentArrayMap(
					lang.NewKeyword("internal-reduce"), lang.NewSymbol("todo"),
				),
				lang.NewKeyword("on-interface"), true,
				lang.NewKeyword("sigs"), lang.NewSymbol("todo"),
			)),
			lang.WithMeta(lang.NewSymbol("IKVReduce"), lang.NewPersistentArrayMap(
				lang.NewKeyword("file"), "glojure/protocols.glj",
				lang.NewKeyword("line"), int(176),
				lang.NewKeyword("column"), int(14),
				lang.NewKeyword("end-line"), int(176),
				lang.NewKeyword("end-column"), int(22),
			)), lang.NewAtom(lang.NewPersistentArrayMap(
				lang.NewKeyword("multis"), lang.NewPersistentArrayMap(
					lang.NewKeyword("kv-reduce"), lang.NewSymbol("todo"),
				),
				lang.NewKeyword("on-interface"), true,
				lang.NewKeyword("sigs"), lang.NewSymbol("todo"),
			)),
			lang.WithMeta(lang.NewSymbol("Datafiable"), lang.NewPersistentArrayMap(
				lang.NewKeyword("file"), "glojure/protocols.glj",
				lang.NewKeyword("line"), int(183),
				lang.NewKeyword("column"), int(14),
				lang.NewKeyword("end-line"), int(183),
				lang.NewKeyword("end-column"), int(23),
			)), lang.NewAtom(lang.NewPersistentArrayMap(
				lang.NewKeyword("multis"), lang.NewPersistentArrayMap(
					lang.NewKeyword("datafy"), lang.NewSymbol("todo"),
				),
				lang.NewKeyword("on-interface"), true,
				lang.NewKeyword("sigs"), lang.NewSymbol("todo"),
			)),
			lang.WithMeta(lang.NewSymbol("Navigable"), lang.NewPersistentArrayMap(
				lang.NewKeyword("file"), "glojure/protocols.glj",
				lang.NewKeyword("line"), int(195),
				lang.NewKeyword("column"), int(14),
				lang.NewKeyword("end-line"), int(195),
				lang.NewKeyword("end-column"), int(22),
			)), lang.NewAtom(lang.NewPersistentArrayMap(
				lang.NewKeyword("multis"), lang.NewPersistentArrayMap(
					lang.NewKeyword("nav"), lang.NewSymbol("todo"),
				),
				lang.NewKeyword("on-interface"), true,
				lang.NewKeyword("sigs"), lang.NewSymbol("todo"),
			)),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("macroexpand"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("concat"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("doall"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("pr-str"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), lang.NewSymbol("todo"),
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("condp"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("data-reader-var"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("ns-imports"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("rationalize"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("apply"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("inc"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("test"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("<"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
	{
		vr := ns.Intern(lang.NewSymbol("set"))
		vr.BindRoot(lang.WithMeta(lang.NewSymbol("todo"), lang.NewPersistentArrayMap(
			lang.NewKeyword("rettag"), nil,
		)))
	}
}
