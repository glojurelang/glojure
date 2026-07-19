package lang

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type Namespace struct {
	name *Symbol

	mappingsMtx        sync.RWMutex
	mappings           map[string]namespaceMapping
	mappingsShared     bool
	referenceSnapshots []namespaceReferenceSnapshot
	mappingsSnapshot   IPersistentMap

	aliases atomic.Value

	meta IPersistentMap
}

type namespaceMapping struct {
	sym *Symbol
	val interface{}
}

type namespaceReferenceSnapshot struct {
	source   *Namespace
	mappings map[string]namespaceMapping
	excluded map[string]struct{}
}

// NamespaceReference describes a Var imported from another namespace.
type NamespaceReference struct {
	Alias  *Symbol
	Source *Symbol
}

var (
	SymbolCoreNamespace = NewSymbol("clojure.core")

	namespaces = map[string]*Namespace{}
	nsMtx      sync.RWMutex
)

func AllNamespaces() ISeq {
	nsMtx.RLock()
	defer nsMtx.RUnlock()
	ns := make([]*Namespace, 0, len(namespaces))
	for _, n := range namespaces {
		ns = append(ns, n)
	}
	return Seq(ns)
}

func FindNamespace(sym *Symbol) *Namespace {
	nsMtx.RLock()
	defer nsMtx.RUnlock()
	return namespaces[sym.String()]
}

func FindOrCreateNamespace(sym *Symbol) *Namespace {
	ns := FindNamespace(sym)
	if ns != nil {
		return ns
	}
	nsMtx.Lock()
	defer nsMtx.Unlock()
	ns = namespaces[sym.String()]
	if ns != nil {
		return ns
	}
	ns = NewNamespace(sym)
	namespaces[sym.String()] = ns
	return ns
}

func RemoveNamespace(sym *Symbol) {
	if sym.String() == "clojure.core" {
		panic(errors.New("cannot remove clojure.core namespace"))
	}

	nsMtx.Lock()
	defer nsMtx.Unlock()
	delete(namespaces, sym.String())
}

func NamespaceFor(inns *Namespace, sym *Symbol) *Namespace {
	//note, presumes non-nil sym.ns
	// first check against currentNS' aliases...
	nsSym := NewSymbol(sym.Namespace())
	ns := inns.LookupAlias(nsSym)
	if ns != nil {
		return ns
	}

	return FindNamespace(nsSym)
}

func NewNamespace(name *Symbol) *Namespace {
	ns := &Namespace{
		name:     name,
		mappings: make(map[string]namespaceMapping),
	}

	seedHostClassImports(ns.mappings)
	ns.aliases.Store(NewBox(emptyMap))

	return ns
}

// seedHostClassImports adds entries for every host class registered in
// pkgmap. Mirrors real Clojure's auto-import of
// java.lang.* (and other packages we publish) so (ns-imports *ns*)
// returns a populated map.
func seedHostClassImports(m map[string]namespaceMapping) {
	for name, typ := range pkgmap.HostClassTypes() {
		sym := NewSymbol(name)
		m[sym.String()] = namespaceMapping{sym: sym, val: typ}
	}
}

func (ns *Namespace) String() string {
	return ns.Name().String()
}

func (ns *Namespace) Name() *Symbol {
	return ns.name
}

func (ns *Namespace) Mappings() IPersistentMap {
	ns.mappingsMtx.Lock()
	defer ns.mappingsMtx.Unlock()

	if ns.mappingsSnapshot != nil {
		return ns.mappingsSnapshot
	}
	visible := make(map[string]namespaceMapping, len(ns.mappings))
	for _, snapshot := range ns.referenceSnapshots {
		for key, mapping := range snapshot.mappings {
			if _, excluded := snapshot.excluded[key]; excluded {
				continue
			}
			vr, ok := mapping.val.(*Var)
			if !ok || vr.Namespace() != snapshot.source ||
				vr.Symbol().String() != key {
				continue
			}
			visible[key] = mapping
		}
	}
	for _, mapping := range ns.mappings {
		visible[mapping.sym.String()] = mapping
	}
	kvs := make([]interface{}, 0, 2*len(visible))
	for _, mapping := range visible {
		kvs = append(kvs, mapping.sym, mapping.val)
	}
	ns.mappingsSnapshot = NewPersistentHashMap(kvs...)
	return ns.mappingsSnapshot
}

func (ns *Namespace) aliasesBox() *Box {
	return ns.aliases.Load().(*Box)
}

func (ns *Namespace) Aliases() IPersistentMap {
	return ns.aliasesBox().val.(IPersistentMap)
}

func (ns *Namespace) isInternedMapping(sym *Symbol, v interface{}) bool {
	vr, ok := v.(*Var)
	return ok && vr.Namespace() == ns && Equals(vr.Symbol(), sym)
}

// Intern creates a new Var in this namespace with the given name.
func (ns *Namespace) Intern(sym *Symbol) *Var {
	if sym.Namespace() != "" {
		panic(fmt.Errorf("can't intern qualified name: %s", sym))
	}
	ns.mappingsMtx.Lock()
	defer ns.mappingsMtx.Unlock()

	key := sym.String()
	mapping, exists := ns.visibleMappingLocked(key)
	if !exists {
		v := NewVar(ns, sym)
		ns.ensureMappingsMutableLocked()
		ns.mappings[key] = namespaceMapping{sym: sym, val: v}
		ns.mappingsSnapshot = nil
		return v
	}
	o := mapping.val
	if ns.isInternedMapping(sym, o) {
		return o.(*Var)
	}
	v := NewVar(ns, sym)
	if ns.checkReplacement(sym, o, v) {
		ns.ensureMappingsMutableLocked()
		ns.mappings[key] = namespaceMapping{sym: sym, val: v}
		ns.mappingsSnapshot = nil
		return v
	}

	return o.(*Var)
}

func (ns *Namespace) checkReplacement(sym *Symbol, old, neu interface{}) bool {
	/*
		 This method checks if a namespace's mapping is applicable and warns on problematic cases.
		 It will return a boolean indicating if a mapping is replaceable.
		 The semantics of what constitutes a legal replacement mapping is summarized as follows:

		| classification | in namespace ns        | newval = anything other than ns/name | newval = ns/name                    |
		|----------------+------------------------+--------------------------------------+-------------------------------------|
		| native mapping | name -> ns/name        | no replace, warn-if newval not-core  | no replace, warn-if newval not-core |
		| alias mapping  | name -> other/whatever | warn + replace                       | warn + replace                      |
	*/

	errOut := GlobalEnv.Stderr()

	if _, ok := old.(*Var); ok {
		var nns *Namespace
		if neuVar, ok := neu.(*Var); ok {
			nns = neuVar.Namespace()
		}
		if ns.isInternedMapping(sym, old) {
			if nns != FindNamespace(SymbolCoreNamespace) {
				fmt.Fprintf(errOut, "REJECTED: attempt to replace interned var %s with %s in %s, you must ns-unmap first\n", old, neu, ns.name)
			}
			return false
		}
	}

	fmt.Fprintf(errOut, "WARNING: %s already refers to %s in namespace: %s, being replaced by: %s\n", sym, old, ns.name, neu)
	return true
}

func (ns *Namespace) InternWithValue(sym *Symbol, value interface{}, replaceRoot bool) *Var {
	v := ns.Intern(sym)
	if !v.HasRoot() || replaceRoot {
		v.BindRoot(value)
	}
	return v
}

// Unmap removes the mapping for the symbol from the namespace.
func (ns *Namespace) Unmap(sym *Symbol) {
	if sym.Namespace() != "" {
		panic(NewIllegalArgumentError("Can't unintern namespace-qualified symbol"))
	}
	mb := ns.mappingsBox()
	for mb.val.(IPersistentMap).ContainsKey(sym) {
		newMap := mb.val.(IPersistentMap).Without(sym)
		ns.mappings.CompareAndSwap(mb, NewBox(newMap))
		mb = ns.mappingsBox()
	}
}

func (ns *Namespace) GetMapping(sym *Symbol) interface{} {
	ns.mappingsMtx.RLock()
	defer ns.mappingsMtx.RUnlock()
	mapping, _ := ns.visibleMappingLocked(sym.String())
	return mapping.val
}

func (ns *Namespace) FindInternedVar(sym *Symbol) *Var {
	ns.mappingsMtx.RLock()
	defer ns.mappingsMtx.RUnlock()

	v := ns.mappings[sym.String()].val
	if v == nil {
		return nil
	}
	vr, ok := v.(*Var)
	if !ok {
		return nil
	}
	if vr.Namespace() != ns {
		return nil
	}
	return vr
}

func (ns *Namespace) LookupAlias(sym *Symbol) *Namespace {
	m := ns.Aliases()
	v := m.ValAt(sym)
	if v == nil {
		return nil
	}
	return v.(*Namespace)
}

func (ns *Namespace) AddAlias(alias *Symbol, ns2 *Namespace) {
	if alias == nil || ns2 == nil {
		panic(fmt.Errorf("add-alias: expecting symbol (%v) + namespace (%v)", alias, ns2))
	}
	ab := ns.aliasesBox()
	for !ab.val.(IPersistentMap).ContainsKey(alias) {
		newAliases := ab.val.(IPersistentMap).Assoc(alias, ns2)
		ns.aliases.CompareAndSwap(ab, NewBox(newAliases))
		ab = ns.aliasesBox()
	}
	if v := ab.val.(IPersistentMap).ValAt(alias); v != ns2 {
		panic(fmt.Errorf("add-alias: alias %s already refers to %s", alias, v))
	}
}

// Import references an export from a Go package.
func (ns *Namespace) Import(export string, v interface{}) interface{} {
	_, name := pkgmap.SplitExport(export)
	ns.reference(NewSymbol(name), v)
	return v
}

// Refer adds a reference to an existing Var, possibly in another
// namespace, to this namespace.
func (ns *Namespace) Refer(sym *Symbol, v *Var) *Var {
	return ns.reference(sym, v).(*Var)
}

// ReferAll adds a batch of references from src. Source mappings are collected
// under one read lock before the target namespace is changed.
func (ns *Namespace) ReferAll(src *Namespace, refs []NamespaceReference) {
	vars := make([]*Var, len(refs))
	src.mappingsMtx.RLock()
	for i, ref := range refs {
		if v, ok := src.mappings[ref.Source.String()].val.(*Var); ok {
			vars[i] = v
		}
	}
	src.mappingsMtx.RUnlock()

	for i, v := range vars {
		if v != nil {
			ns.Refer(refs[i].Alias, v)
		}
	}
}

// ReferAllSnapshot installs a shared, immutable view of src's interned Vars,
// excluding the supplied unqualified names. Generated AOT namespaces use this
// for broad refers such as clojure.core: lookups remain lazy, while a later
// mutation of src uses copy-on-write and cannot change the captured view.
func (ns *Namespace) ReferAllSnapshot(src *Namespace, excluded []string) {
	mappings := src.shareMappings()
	excludedSet := make(map[string]struct{}, len(excluded))
	for _, name := range excluded {
		excludedSet[name] = struct{}{}
	}

	ns.mappingsMtx.Lock()
	defer ns.mappingsMtx.Unlock()
	snapshot := namespaceReferenceSnapshot{
		source:   src,
		mappings: mappings,
		excluded: excludedSet,
	}
	for i := range ns.referenceSnapshots {
		if ns.referenceSnapshots[i].source == src {
			ns.referenceSnapshots[i] = snapshot
			ns.mappingsSnapshot = nil
			return
		}
	}
	ns.referenceSnapshots = append(ns.referenceSnapshots, snapshot)
	ns.mappingsSnapshot = nil
}

func (ns *Namespace) reference(sym *Symbol, v interface{}) interface{} {
	if sym.Namespace() != "" {
		panic(fmt.Errorf("can't intern qualified name: %s", sym))
	}
	if v == nil {
		panic(fmt.Errorf("can't refer to nil (%s)", sym))
	}

	ns.mappingsMtx.Lock()
	defer ns.mappingsMtx.Unlock()

	key := sym.String()
	mapping, exists := ns.visibleMappingLocked(key)
	if !exists {
		ns.ensureMappingsMutableLocked()
		ns.mappings[key] = namespaceMapping{sym: sym, val: v}
		ns.mappingsSnapshot = nil
		return v
	}
	o := mapping.val
	if ns.isInternedMapping(sym, o) {
		return o.(*Var)
	}

	// NB: in Go, some types are not comparable.
	oCmp := reflect.TypeOf(o).Comparable()
	vCmp := reflect.TypeOf(v).Comparable()
	if oCmp && vCmp {
		if o == v {
			return o
		}
	} else if oCmp == vCmp {
		// TODO: what to do here? for now, assume equal
		return o
	}

	if ns.checkReplacement(sym, o, v) {
		ns.ensureMappingsMutableLocked()
		ns.mappings[key] = namespaceMapping{sym: sym, val: v}
		ns.mappingsSnapshot = nil
		return v
	}

	return o
}

func (ns *Namespace) visibleMappingLocked(key string) (namespaceMapping, bool) {
	if mapping, exists := ns.mappings[key]; exists {
		return mapping, true
	}
	for i := len(ns.referenceSnapshots) - 1; i >= 0; i-- {
		snapshot := ns.referenceSnapshots[i]
		if _, excluded := snapshot.excluded[key]; excluded {
			continue
		}
		mapping, exists := snapshot.mappings[key]
		if !exists {
			continue
		}
		vr, ok := mapping.val.(*Var)
		if ok && vr.Namespace() == snapshot.source &&
			vr.Symbol().String() == key {
			return mapping, true
		}
	}
	return namespaceMapping{}, false
}

func (ns *Namespace) ensureMappingsMutableLocked() {
	if !ns.mappingsShared {
		return
	}
	mappings := make(map[string]namespaceMapping, len(ns.mappings))
	for key, mapping := range ns.mappings {
		mappings[key] = mapping
	}
	ns.mappings = mappings
	ns.mappingsShared = false
}

func (ns *Namespace) shareMappings() map[string]namespaceMapping {
	ns.mappingsMtx.Lock()
	defer ns.mappingsMtx.Unlock()
	if len(ns.referenceSnapshots) == 0 {
		ns.mappingsShared = true
		return ns.mappings
	}

	visible := make(map[string]namespaceMapping, len(ns.mappings))
	for _, snapshot := range ns.referenceSnapshots {
		for key, mapping := range snapshot.mappings {
			if _, excluded := snapshot.excluded[key]; excluded {
				continue
			}
			vr, ok := mapping.val.(*Var)
			if ok && vr.Namespace() == snapshot.source &&
				vr.Symbol().String() == key {
				visible[key] = mapping
			}
		}
	}
	for key, mapping := range ns.mappings {
		visible[key] = mapping
	}
	return visible
}

func (ns *Namespace) Meta() IPersistentMap {
	return ns.meta
}

func (ns *Namespace) AlterMeta(alter IFn, args ISeq) IPersistentMap {
	meta := alter.ApplyTo(NewCons(ns.Meta(), args)).(IPersistentMap)
	ns.ResetMeta(meta)
	return meta
}

func (ns *Namespace) ResetMeta(meta IPersistentMap) IPersistentMap {
	ns.meta = meta
	return meta
}
