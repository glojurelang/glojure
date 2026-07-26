//go:build !glj_aot_runtime

package runtime

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"go/format"
	"go/token"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	goruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

// TODO
// - handle namespace requires/uses/etc.
// - handle let bindings that are shared across multiple vars
// - test repeated let bindings of the same name, where previous bindings are shadowed

// varScope represents a variable allocation scope
type varScope struct {
	nextNum    int
	names      map[string]string // maps Clojure names to Go variable names
	localAtoms map[string]bool   // local atoms proven not to escape
}

// recurContext represents the context for a loop/recur form
type recurContext struct {
	loopID   *lang.Symbol // The loop ID to match recur with its loop
	bindings []string     // Go variable names for loop bindings (in order)
	useGoto  bool         // Whether to use Go's "goto" for recur
}

// liftedKey is a composite key for deduplicating lifted values
type liftedKey struct {
	isPointer bool
	pointer   uintptr // For reference types
	value     any     // For primitive types (used in equality check)
}

// liftedValue represents a value that has been lifted to package scope
type liftedValue struct {
	value   any
	varName string
}

type varInfo struct {
	ns  string
	sym string
}

type namedVar struct {
	name *lang.Symbol
	vr   *lang.Var
}

type aotReferredVar struct {
	symName string
	srcNS   string
	srcSym  string
}

type valueInit struct {
	name string              // Name of the variable or var being initialized
	buf  bytes.Buffer        // Buffer holding the initialization code
	deps map[string]struct{} // Set of var/value names this value depends on
}

type aotSpecializationTarget struct {
	vr              *lang.Var
	fn              *Fn
	arity           int
	arityDispatch   bool
	directLinked    bool
	directArities   [21]bool
	directFnVar     string
	directArityVars [21]string
	int64FnVar      string
	int64Analysis   *int64AOTAnalysis
	float64FnVar    string
	float64Analysis *float64AOTAnalysis
	rootVersionVar  string
}

type aotExternalCallTarget struct {
	vr             *lang.Var
	arity          int
	fnVar          string
	intrinsic      string
	directLinked   bool
	defaultVar     string
	rootVersionVar string
}

type aotExternalCallKey struct {
	vr        *lang.Var
	arity     int
	intrinsic string
}

type aotKeywordLookupHelper struct {
	name    string
	keyword string
}

type aotKeywordAssocHelper struct {
	name     string
	keywords []string
}

type aotRecordType struct {
	index        int
	descriptor   *lang.RecordType
	descriptorGo string
	typeName     string
	constructor  string
	mapFactory   string
	fieldNames   []string
}

type aotRecordCallTarget struct {
	record  *aotRecordType
	fromMap bool
}

// Generator handles the conversion of AST nodes to Go code
type Generator struct {
	originalWriter io.Writer

	currentWriter    io.Writer
	currentValueInit *valueInit // current value initialization being generated

	varScopes  []varScope     // stack of variable scopes
	recurStack []recurContext // stack of recur contexts for nested loops

	imports                map[string]string  // set of imported packages with their aliases
	varVariables           map[varInfo]string // map of vars to their Go variable names
	symbolVariables        map[string]string  // set of all generated symbols to minimize allocations
	kwVariables            map[string]string  // set of all generated keywords to minimize allocations
	keywordMapConstructors map[string]string  // co-allocating constructors for those layouts
	keywordLookupHelpers   map[string]*aotKeywordLookupHelper
	keywordAssocHelpers    map[string]*aotKeywordAssocHelper
	aotRecordTypes         map[*lang.RecordType]*aotRecordType

	valueInits []*valueInit // map of value initializations

	aotDeclarations        bytes.Buffer
	aotCallTargets         map[*lang.Var]*aotSpecializationTarget
	aotExternalCallTargets map[aotExternalCallKey]*aotExternalCallTarget
	aotNamespace           *lang.Namespace
	directLink             bool

	// Fields for handling closures
	liftedValues  map[liftedKey]*liftedValue // Dedupe by composite key
	liftedCounter int                        // Counter for closed0, closed1...
	currentFnEnv  lang.Environment           // Current function's captured env

	// specializationTarget is non-nil only while generating the root function
	// value for a Var. Nested function literals retain the generic code path.
	specializationTarget *aotSpecializationTarget
}

var (
	omittedVars = map[string]bool{
		// initialized by the runtime
		"#'clojure.core/*in*":            true,
		"#'clojure.core/*out*":           true,
		"#'clojure.core/*compile-files*": true,
		"#'clojure.core/load-file":       true,
		"#'clojure.core/add-load-path":   true,
		"#'clojure.core/shuffle":         true,
		"#'clojure.core/promise":         true,
	}

	runtimeStateInitializers = map[string]string{
		// Loaded namespaces are process-local state. Serializing the compiler
		// process's set makes a fresh AOT process skip namespaces it has not
		// actually loaded.
		"#'clojure.core/*loaded-libs*": "lang.NewRef(lang.NewSet())",
	}
)

// NewGenerator creates a new code generator
func NewGenerator(w io.Writer) *Generator {
	return newGenerator(w, true)
}

func newGenerator(w io.Writer, directLink bool) *Generator {
	return &Generator{
		originalWriter:         w,
		currentWriter:          w,
		varScopes:              []varScope{{nextNum: 0, names: make(map[string]string)}},
		recurStack:             []recurContext{},
		imports:                make(map[string]string),
		varVariables:           make(map[varInfo]string),
		symbolVariables:        make(map[string]string),
		kwVariables:            make(map[string]string),
		keywordMapConstructors: make(map[string]string),
		keywordLookupHelpers:   make(map[string]*aotKeywordLookupHelper),
		keywordAssocHelpers:    make(map[string]*aotKeywordAssocHelper),
		aotRecordTypes:         make(map[*lang.RecordType]*aotRecordType),
		liftedValues:           make(map[liftedKey]*liftedValue),
		liftedCounter:          0,
		aotCallTargets:         make(map[*lang.Var]*aotSpecializationTarget),
		aotExternalCallTargets: make(map[aotExternalCallKey]*aotExternalCallTarget),
		directLink:             directLink,
	}
}

func runtimeStateInitializer(vr *lang.Var) (string, bool) {
	initializer, ok := runtimeStateInitializers[vr.String()]
	return initializer, ok
}

// Generate takes a namespace and generates Go code that populates the same namespace
func (g *Generator) Generate(ns *lang.Namespace) error {
	g.aotNamespace = ns

	// add lang import
	g.addImport("github.com/glojurelang/glojure/pkg/lang")
	g.addImport("github.com/glojurelang/glojure/pkg/runtime")
	g.addImport("fmt")     // for error formatting
	g.addImport("reflect") // for reflect.TypeOf

	var nsBuf bytes.Buffer
	g.currentWriter = &nsBuf

	g.writef("// reference fmt to avoid unused import error\n")
	g.writef("_ = fmt.Printf\n")
	g.writef("// reference reflect to avoid unused import error\n")
	g.writef("_ = reflect.TypeOf\n")

	g.writef("  ns := lang.FindOrCreateNamespace(%s)\n", g.allocSymVar(ns.Name().String()))
	g.writef("  _ = ns\n")

	// 1. Iterate through ns.Mappings()
	// 2. Generate Go code for each var (this discovers lifted values)
	mappings := ns.Mappings()

	// Collect exact referred vars from compile-time mappings.
	var referredVars []aotReferredVar

	for seq := mappings.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First()
		name, ok := lang.First(entry).(*lang.Symbol)
		if !ok {
			continue
		}
		second, _ := lang.Nth(entry, 1)
		vr, ok := second.(*lang.Var)
		if !ok {
			continue
		}
		// Non-interned = referred from another namespace
		if !(vr.Namespace() == ns && lang.Equals(vr.Symbol(), name)) {
			referredVars = append(referredVars, aotReferredVar{
				symName: name.String(),
				srcNS:   vr.Namespace().Name().String(),
				srcSym:  vr.Symbol().String(),
			})
		}
	}

	// Emit one batch per source namespace so generated loaders do not build a
	// complete persistent mapping snapshot or lock the source for every Var.
	referredByNamespace := make(map[string][]aotReferredVar)
	var referredNamespaces []string
	for _, rv := range referredVars {
		if _, ok := referredByNamespace[rv.srcNS]; !ok {
			referredNamespaces = append(referredNamespaces, rv.srcNS)
		}
		referredByNamespace[rv.srcNS] = append(referredByNamespace[rv.srcNS], rv)
	}
	sort.Strings(referredNamespaces)
	for _, srcNS := range referredNamespaces {
		refs := referredByNamespace[srcNS]
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].symName == refs[j].symName {
				return refs[i].srcSym < refs[j].srcSym
			}
			return refs[i].symName < refs[j].symName
		})
		snapshot, exclusions := snapshotAOTReferences(srcNS, refs)
		explicit := refs
		if snapshot {
			explicit = make([]aotReferredVar, 0, len(refs))
			for _, ref := range refs {
				if ref.symName != ref.srcSym {
					explicit = append(explicit, ref)
				}
			}
		}

		srcNSSym := g.allocSymVar(srcNS)
		g.writef("{ // refer vars from %s\n", srcNS)
		g.writef("  srcNS := lang.FindOrCreateNamespace(%s)\n", srcNSSym)
		if snapshot {
			g.writef("  ns.ReferAllSnapshot(srcNS, []string{\n")
			for _, exclusion := range exclusions {
				g.writef("    %q,\n", exclusion)
			}
			g.writef("  })\n")
		}
		if len(explicit) != 0 {
			g.writef("  ns.ReferAll(srcNS, []lang.NamespaceReference{\n")
			for _, rv := range explicit {
				symSym := g.allocSymVar(rv.symName)
				srcSymSym := g.allocSymVar(rv.srcSym)
				g.writef("    {Alias: %s, Source: %s},\n", symSym, srcSymSym)
			}
			g.writef("  })\n")
		}
		g.writef("}\n")
	}

	// Generate alias setup
	aliases := ns.Aliases()
	for seq := aliases.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First()
		aliasSym := lang.First(entry).(*lang.Symbol)
		targetNS, _ := lang.Nth(entry, 1)
		g.writef("ns.AddAlias(%s, lang.FindOrCreateNamespace(%s))\n",
			g.allocSymVar(aliasSym.String()),
			g.allocSymVar(targetNS.(*lang.Namespace).Name().String()))
	}

	var internedVars []namedVar

	for seq := mappings.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First()
		name, ok := lang.First(entry).(*lang.Symbol)
		if !ok {
			panic(fmt.Sprintf("expected symbol, got %T", entry))
		}
		second, _ := lang.Nth(entry, 1)
		vr, ok := second.(*lang.Var)
		if !ok {
			continue // skip non-var mappings
			// TODO: handle non-var mappings like direct references to functions or values
			// panic(fmt.Sprintf("can't codegen %v: expected var, got %T (%v)", name, second, second))
		}

		if !(vr.Namespace() == ns && lang.Equals(vr.Symbol(), name)) {
			continue // Skip non-interned mappings
		}

		internedVars = append(internedVars, namedVar{name: name, vr: vr})
	}
	// Sort internedVars by name for deterministic output
	sort.Slice(internedVars, func(i, j int) bool {
		return internedVars[i].name.String() < internedVars[j].name.String()
	})
	g.prepareAOTRecordTypes(internedVars)
	g.prepareAOTCallTargets(internedVars)
	for _, nv := range internedVars {
		if isRuntimeOwnedVar(nv.vr) {
			// Skip runtime-owned vars
			continue
		}

		if err := g.generateVar("ns", nv.name, nv.vr); err != nil {
			return fmt.Errorf("failed to generate code for var %s: %w", nv.name, err)
		}
	}

	////////////////////////////////////////////////////////////////////////////////
	// Generate lifted values at the beginning of init() if any
	if len(g.liftedValues) > 0 {
		generated := make(map[*liftedValue]bool)
		for {
			// Generating a lifted closure can discover more captured values.
			// Drain them all, sorting each batch for deterministic output.
			var sortedLifted []*liftedValue
			for _, lifted := range g.liftedValues {
				if !generated[lifted] {
					sortedLifted = append(sortedLifted, lifted)
				}
			}
			if len(sortedLifted) == 0 {
				break
			}
			sort.Slice(sortedLifted, func(i, j int) bool {
				return sortedLifted[i].varName < sortedLifted[j].varName
			})

			for _, lifted := range sortedLifted {
				generated[lifted] = true
				g.startNewValueInit(lifted.varName)
				g.pushVarScope()
				g.writef("{\n")
				valueCode := g.generateValue(lifted.value)
				// Declare the lifted variable with the final value
				g.writef("%s = %s\n", lifted.varName, valueCode)
				g.writef("}\n")
				g.popVarScope()
			}
		}

		// Declare every captured value before any initializer. Nested closures
		// can introduce forward references and cycles while they are generated.
		var names []string
		for _, lifted := range g.liftedValues {
			names = append(names, lifted.varName)
		}
		sort.Strings(names)
		for _, name := range names {
			nsBuf.WriteString(fmt.Sprintf("var %s any\n", name))
		}
	}

	////////////////////////////////////////////////////////////////////////////////

	// Now construct the complete init function
	var initBuf bytes.Buffer
	{
		// Reproduce the behavior of root-resource function
		rootResourceName := nsToPath(ns.Name().String())
		initBuf.WriteString(`func init() {
runtime.RegisterNSLoader(` + fmt.Sprintf("%q", rootResourceName) + `, LoadNS)
}

`)
	}
	initBuf.WriteString(`func checkDerefVar (v *lang.Var) any {
  if v.IsMacro() {
	  panic(lang.NewIllegalArgumentError(fmt.Sprintf("can't take value of macro: %v", v)))
  }
  return v.Get()
}

`)
	initBuf.WriteString(`func checkArity(args []any, expected int) {
  if len(args) != expected {
		panic(lang.NewIllegalArgumentError("wrong number of arguments (" + fmt.Sprint(len(args)) + ")"))
  }
}

`)
	initBuf.WriteString(`func checkArityGTE(args []any, min int) {
  if len(args) < min {
		panic(lang.NewIllegalArgumentError("wrong number of arguments (" + fmt.Sprint(len(args)) + ")"))
  }
}

`)
	initBuf.WriteString(fmt.Sprintf("// LoadNS initializes the namespace %q\n", ns.Name().String()))
	initBuf.WriteString("func LoadNS() {\n")

	//////////////////////////
	// Symbols
	var symbolNames []string
	for sym := range g.symbolVariables {
		symbolNames = append(symbolNames, sym)
	}
	sort.Strings(symbolNames) // Sort for deterministic output
	for _, sym := range symbolNames {
		varName := g.symbolVariables[sym]
		initBuf.WriteString(fmt.Sprintf("%s := lang.NewSymbolUnchecked(%q)\n", varName, sym))
	}

	//////////////////////////
	// Keywords
	var kwNames []string
	for kw := range g.kwVariables {
		kwNames = append(kwNames, kw)
	}
	sort.Strings(kwNames) // Sort for deterministic output
	for _, kw := range kwNames {
		varName := g.kwVariables[kw]
		initBuf.WriteString(fmt.Sprintf("%s := lang.NewKeyword(%q)\n", varName, kw))
	}

	//////////////////////////
	// Vars initialization
	var varNames []string
	var inverseVarMap = make(map[string]varInfo)
	for vi, varName := range g.varVariables {
		varNames = append(varNames, varName)
		inverseVarMap[varName] = vi
	}
	sort.Strings(varNames) // Sort for deterministic output
	for _, varName := range varNames {
		vi := inverseVarMap[varName]
		initBuf.WriteString(fmt.Sprintf("// var %s/%s\n", vi.ns, vi.sym))
		// NB: the variables will already have been allocated
		initBuf.WriteString(fmt.Sprintf("%s := lang.InternVarName(%s, %s)\n", varName, g.allocSymVar(vi.ns), g.allocSymVar(vi.sym)))
	}

	/////////////////////////////
	// Roots of statically resolved calls to other namespaces
	externalTargets := make([]*aotExternalCallTarget, 0, len(g.aotExternalCallTargets))
	for _, target := range g.aotExternalCallTargets {
		externalTargets = append(externalTargets, target)
	}
	sort.Slice(externalTargets, func(i, j int) bool {
		return externalTargets[i].fnVar < externalTargets[j].fnVar
	})
	for _, target := range externalTargets {
		if target.intrinsic != "" && target.directLinked {
			continue
		}
		varName := g.allocVarVar(
			target.vr.Namespace().Name().String(),
			target.vr.Symbol().String(),
		)
		if target.intrinsic != "" {
			initBuf.WriteString(fmt.Sprintf(
				"%s := runtime.IsDefaultCoreVar(%s)\n%s := %s.RootVersion()\n",
				target.defaultVar,
				varName,
				target.rootVersionVar,
				varName,
			))
			continue
		}
		adapter := "Cache"
		if target.directLinked {
			adapter = "Link"
		}
		initBuf.WriteString(fmt.Sprintf(
			"%s := aot%sFn%d(%s)\n",
			target.fnVar, adapter, target.arity, varName,
		))
	}

	/////////////////////////////
	// Var and closed-over value inits

	// NS boilerplate
	initBuf.Write(nsBuf.Bytes())

	{
		sort.Slice(g.valueInits, func(i, j int) bool {
			return g.valueInits[i].name < g.valueInits[j].name
		})

		dependents := make(map[string][]*valueInit)

		for _, vi := range g.valueInits {
			for dep := range vi.deps {
				if dep == vi.name {
					continue // skip self-dependency
				}
				dependents[dep] = append(dependents[dep], vi)
			}
		}
		// // print dependencies for debugging
		// for _, vi := range g.valueInits {
		// 	fmt.Printf("# %s\n", vi.name)
		// 	for dep := range vi.deps {
		// 		fmt.Printf("  -> %s\n", dep)
		// 	}
		// 	fmt.Println()
		// }

		// Simple dependency resolution: repeatedly emit value inits that have no remaining deps
		emitted := make(map[string]bool)
		for len(emitted) < len(g.valueInits) {
			progress := false
			for _, vi := range g.valueInits {
				if emitted[vi.name] {
					continue // already emitted
				}
				// Check if all dependencies have been emitted
				allDepsEmitted := true
				for dep := range vi.deps {
					if !emitted[dep] {
						allDepsEmitted = false
						break
					}
				}
				if allDepsEmitted {
					// Emit this value init
					initBuf.WriteString(vi.buf.String())
					emitted[vi.name] = true
					progress = true
					// Remove this from dependents
					for _, depVi := range dependents[vi.name] {
						delete(depVi.deps, vi.name)
					}
				}
			}
			if !progress {
				// Circular dependency detected; break the cycle by emitting one of the remaining inits
				for _, vi := range g.valueInits {
					if !emitted[vi.name] {
						initBuf.WriteString(vi.buf.String())
						emitted[vi.name] = true
						break
					}
				}
			}
		}
	}

	// Closing brace for LoadNS
	initBuf.WriteString("}\n")
	g.generateAOTKeywordHelpers()
	g.generateAOTExternalAdapters()

	////////////////////////////////////////////////////////////////////////////////

	// Prepare the final source
	sourceBytes := []byte(g.header(mungePackageName(getLastNSPart(ns.Name().String())))) // File header with package and imports
	sourceBytes = append(sourceBytes, g.aotDeclarations.Bytes()...)                      // Package-level AOT call caches
	sourceBytes = append(sourceBytes, initBuf.Bytes()...)                                // The complete init function

	// Format the generated code
	formatted, err := format.Source(sourceBytes)
	if err != nil {
		// If formatting fails, write the unformatted code with the error
		g.originalWriter.Write(sourceBytes)
		return fmt.Errorf("formatting failed: %w\n", err)
	}

	// Write formatted code to the original writer
	_, err = g.originalWriter.Write(formatted)
	return err
}

func snapshotAOTReferences(
	sourceName string,
	refs []aotReferredVar,
) (bool, []string) {
	source := lang.FindNamespace(lang.NewSymbol(sourceName))
	if source == nil {
		return false, nil
	}
	direct := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.symName == ref.srcSym {
			direct[ref.symName] = struct{}{}
		}
	}
	if len(direct) == 0 {
		return false, nil
	}

	var exclusions []string
	for seq := source.Mappings().Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First()
		name, ok := lang.First(entry).(*lang.Symbol)
		if !ok {
			continue
		}
		value, _ := lang.Nth(entry, 1)
		vr, ok := value.(*lang.Var)
		if !ok || vr.Namespace() != source ||
			vr.Symbol().String() != name.String() {
			continue
		}
		if _, included := direct[name.String()]; !included {
			exclusions = append(exclusions, name.String())
		}
	}
	if len(exclusions) >= len(direct) {
		return false, nil
	}
	sort.Strings(exclusions)
	return true, exclusions
}

////////////////////////////////////////////////////////////////////////////////

// generateVar generates Go code for a single Var
func (g *Generator) generateVar(nsVariableName string, name *lang.Symbol, vr *lang.Var) error {
	if omittedVars[vr.String()] {
		// Skip omitted vars like *in* and *out*, which are initialized by the runtime
		return nil
	}

	// Generate code for the var
	varVar := g.allocVarVar(vr.Namespace().Name().String(), name.String())
	g.startNewValueInit(varVar)

	g.pushVarScope()
	defer g.popVarScope()
	defer func() { g.specializationTarget = nil }()

	g.writef("// %s\n", name.String())
	g.writef("{\n")
	defer g.writef("}\n")

	meta := vr.Meta()
	varSym := g.allocateTempVar()
	var isDynamic bool
	g.writef("%s := %s\n", varSym, g.allocSymVar(name.String()))
	if !lang.IsNil(meta) && RT.BooleanCast(lang.Get(meta, lang.KWDynamic)) {
		isDynamic = true
	}

	// check if the var has a value
	if initializer, ok := runtimeStateInitializer(vr); ok {
		g.writef("%s = %s.InternWithValue(%s, %s, true)\n", varVar, nsVariableName, varSym, initializer)
	} else if vr.IsBound() {
		// we call Get() on a new goroutine to ensure we get the root value in the case
		// of dynamic vars
		valChan := make(chan any)
		go func() {
			valChan <- vr.Get()
		}()
		v := <-valChan
		if target := g.aotCallTargets[vr]; target != nil {
			g.specializationTarget = target
		}
		valueExpr := g.generateValue(v)
		if target := g.specializationTarget; target != nil {
			g.writef("%s = %s\n", target.directFnVar, valueExpr)
		}
		g.writef("%s = %s.InternWithValue(%s, %s, true)\n", varVar, nsVariableName, varSym, valueExpr)
		if target := g.specializationTarget; target != nil && target.rootVersionVar != "" {
			g.writef("%s = %s.RootVersion()\n", target.rootVersionVar, varVar)
		}
	} else {
		g.writef("%s = %s.Intern(%s)\n", varVar, nsVariableName, varSym)
	}

	// Set metadata on the var if the symbol has metadata
	if meta != nil {
		g.writef("%s.SetMetaLazy(func() lang.IPersistentMap {\n", varVar)
		g.pushVarScope()
		metaVariable := g.generateValue(meta)
		g.writef("\treturn %s\n", metaVariable)
		g.popVarScope()
		g.writef("})\n")
	}
	if isDynamic {
		g.writef("%s.SetDynamic()\n", varVar)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Value Generation

// returns the variable name or constant expression for the value
func (g *Generator) generateValue(value any) string {
	switch v := value.(type) {
	case *lang.RecordType:
		return g.allocAOTRecordType(v).descriptorGo
	case *lang.RecordConstructor:
		return g.generateRecordConstructorValue(v)
	case *lang.Class:
		return g.generateClassValue(v)
	case reflect.Type:
		return g.generateTypeValue(v)
	case *lang.Atom:
		return g.generateAtomValue(v)
	case *lang.Ref:
		return g.generateRefValue(v)
	case *lang.Var:
		// Generate a reference to a Var
		ns := v.Namespace()
		sym := v.Symbol()
		return fmt.Sprintf("lang.FindOrCreateNamespace(%s).FindInternedVar(%s)", g.allocSymVar(ns.Name().String()), g.allocSymVar(sym.String()))
	case *lang.Namespace:
		return fmt.Sprintf("lang.FindOrCreateNamespace(%s)", g.allocSymVar(v.Name().String()))
	case *lang.NumberMethods:
		// Numbers is a stateless, package-level host-method receiver.
		return "lang.Numbers"
	case *lang.LockingTransactor:
		return "lang.LockingTransaction"
	case *http.Client:
		if v == http.DefaultClient {
			return g.addImportWithAlias("net/http") + ".DefaultClient"
		}
		panic("cannot generate a non-default HTTP client")
	case *os.File:
		alias := g.addImportWithAlias("os")
		switch v {
		case os.Stdin:
			return alias + ".Stdin"
		case os.Stdout:
			return alias + ".Stdout"
		case os.Stderr:
			return alias + ".Stderr"
		default:
			panic("cannot generate a non-standard file handle")
		}
	case *RTMethods:
		// RT is the package-level host-method receiver used by core forms.
		return "runtime.RT"
	case *evalCompiler:
		return "runtime.Compiler"
	case *Fn:
		return g.generateFn(v)
	case lang.FnFunc:
		return g.generateFnFunc(v)
	case lang.IPersistentMap:
		return g.generateMapValue(v)
	case lang.IPersistentVector:
		return g.generateVectorValue(v)
	case lang.IPersistentSet:
		return g.generateSetValue(v)
	case *lang.MultiFn:
		return g.generateMultiFn(v)
	case *lang.Volatile:
		return fmt.Sprintf("lang.NewVolatile(%s)", g.generateValue(v.Deref()))
	case *lang.Delay:
		fn := v.PendingFn()
		if fn == nil {
			panic("cannot generate an already-realized delay")
		}
		return fmt.Sprintf("lang.NewDelay(%s)", g.generateValue(fn))
	case lang.Keyword:
		if ns := v.Namespace(); ns != nil {
			return g.allocKWVar(fmt.Sprintf("%s/%s", ns, v.Name()))
		} else {
			return g.allocKWVar(v.Name())
		}
	case *lang.Symbol:
		return g.allocSymVar(v.String())
	case lang.Char:
		return fmt.Sprintf("lang.NewChar(%#v)", rune(v))
	case string:
		// just return the string as a Go string literal
		return fmt.Sprintf("%#v", v)
	case int:
		return fmt.Sprintf("int(%d)", v)
	case int64:
		return fmt.Sprintf("int64(%d)", v)
	case float64:
		return fmt.Sprintf("float64(%s)", g.generateFloatLiteral(v, 64))
	case float32:
		return fmt.Sprintf("float32(%s)", g.generateFloatLiteral(float64(v), 32))
	case time.Duration:
		alias := g.addImportWithAlias("time")
		return fmt.Sprintf("%s.Duration(%d)", alias, int64(v))
	case *regexp.Regexp:
		return fmt.Sprintf("%s.MustCompile(%#v)", g.addImportWithAlias("regexp"), v.String())
	case *lang.BigDecimal:
		return g.generateBigDecimalValue(v)
	case *lang.BigInt:
		return generateBigIntValue(v)
	case *lang.Ratio:
		return generateRatioValue(v)
	case bool:
		// return the boolean as a Go boolean literal
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "nil"
	default:
		if lang.IsSeq(v) {
			var vals []string
			for seq := lang.Seq(v); seq != nil; seq = seq.Next() {
				first := seq.First()
				vals = append(vals, g.generateValue(first))
			}
			return fmt.Sprintf("lang.NewList(%s)", strings.Join(vals, ", "))
		}

		if fname, ok := getWellKnownFunctionName(v); ok {
			if fname == "math.IsNaN" {
				return g.addImportWithAlias("math") + ".IsNaN"
			}
			return fname
		}

		rv := reflect.ValueOf(v)
		if scalar, ok := g.generateNamedScalarValue(rv); ok {
			return scalar
		}

		if rv.IsValid() && rv.Kind() == reflect.Func {
			if fn := goruntime.FuncForPC(rv.Pointer()); fn != nil {
				const langPrefix = "github.com/glojurelang/glojure/pkg/lang."
				if name := strings.TrimPrefix(fn.Name(), langPrefix); name != fn.Name() && token.IsIdentifier(name) {
					return "lang." + name
				}
				if name := strings.TrimPrefix(fn.Name(), "math."); name != fn.Name() && token.IsIdentifier(name) {
					return g.addImportWithAlias("math") + "." + name
				}
				if pkgName, name, ok := strings.Cut(fn.Name(), "."); ok &&
					token.IsIdentifier(name) &&
					map[string]bool{
						"errors":  true,
						"fmt":     true,
						"sort":    true,
						"strings": true,
					}[pkgName] {
					return g.addImportWithAlias(pkgName) + "." + name
				}
				if dot := strings.LastIndexByte(fn.Name(), '.'); dot > 0 {
					pkgPath, name := fn.Name()[:dot], fn.Name()[dot+1:]
					pkgBase := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:]
					if !strings.ContainsRune(pkgBase, '.') && token.IsIdentifier(name) {
						return g.addImportWithAlias(pkgPath) + "." + name
					}
				}
				panic(fmt.Sprintf("unsupported function value %T (%s)", v, fn.Name()))
			}
		}
		panic(fmt.Sprintf("unsupported value type %T: %v", v, v))
	}
}

// generateNamedScalarValue emits constants whose Go type has a name, such as
// fs.FileMode or uuid.UUID. Host symbols resolve to their exact Go values
// during analysis, so AOT generation must preserve both the value and its
// named type.
func (g *Generator) generateNamedScalarValue(v reflect.Value) (string, bool) {
	if !v.IsValid() || v.Type().Name() == "" {
		return "", false
	}

	typeName := g.getTypeString(v.Type())
	switch v.Kind() {
	case reflect.Bool:
		return fmt.Sprintf("%s(%t)", typeName, v.Bool()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%s(%d)", typeName, v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%s(%d)", typeName, v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%s(%s)", typeName, g.generateFloatLiteral(v.Float(), v.Type().Bits())), true
	case reflect.Complex64, reflect.Complex128:
		bits := v.Type().Bits() / 2
		value := v.Complex()
		return fmt.Sprintf(
			"%s(complex(%s, %s))",
			typeName,
			g.generateFloatLiteral(real(value), bits),
			g.generateFloatLiteral(imag(value), bits),
		), true
	case reflect.String:
		return fmt.Sprintf("%s(%q)", typeName, v.String()), true
	case reflect.Array:
		elements := make([]string, v.Len())
		for i := range elements {
			element, ok := g.generateScalarLiteral(v.Index(i))
			if !ok {
				return "", false
			}
			elements[i] = element
		}
		return fmt.Sprintf("%s{%s}", typeName, strings.Join(elements, ", ")), true
	default:
		return "", false
	}
}

func (g *Generator) generateScalarLiteral(v reflect.Value) (string, bool) {
	switch v.Kind() {
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return g.generateFloatLiteral(v.Float(), v.Type().Bits()), true
	case reflect.Complex64, reflect.Complex128:
		bits := v.Type().Bits() / 2
		value := v.Complex()
		return fmt.Sprintf(
			"complex(%s, %s)",
			g.generateFloatLiteral(real(value), bits),
			g.generateFloatLiteral(imag(value), bits),
		), true
	case reflect.String:
		return fmt.Sprintf("%q", v.String()), true
	default:
		return "", false
	}
}

// generateFloatLiteral uses ordinary Go literals when they preserve the value.
// Go has no literal syntax for infinities, NaNs, or negative zero, so emit those
// values from their exact IEEE-754 bit pattern.
func (g *Generator) generateFloatLiteral(value float64, bits int) string {
	if !math.IsNaN(value) && !math.IsInf(value, 0) && (value != 0 || !math.Signbit(value)) {
		return fmt.Sprintf("%g", value)
	}

	alias := g.addImportWithAlias("math")
	if bits == 32 {
		return fmt.Sprintf("%s.Float32frombits(0x%08x)", alias, math.Float32bits(float32(value)))
	}
	return fmt.Sprintf("%s.Float64frombits(0x%016x)", alias, math.Float64bits(value))
}

// generateClassValue wraps the embedded reflect.Type in a fresh
// lang.Class so the AOT-compiled binary preserves the JVM-style class
// identity (and its FQ Java name) for symbols seeded by the host-class
// import path (Math, Integer, java.lang.Integer, ...).
func (g *Generator) generateClassValue(c *lang.Class) string {
	typeExpr := g.generateTypeValue(c.Type)
	resultId := g.allocateTempVar()
	g.writef("%s := lang.NewClass(%s, %#v)\n", resultId, typeExpr, c.JavaName)
	return resultId
}

func (g *Generator) generateTypeValue(t reflect.Type) string {
	resultId := g.allocateTempVar()

	// Generate the appropriate zero value expression based on the type
	// TODO: review this LLM slop
	zeroValueExpr := g.generateZeroValueExpr(t)

	// For named types (structs, interfaces), use the (*T)(nil).Elem() pattern
	// For other types, use the zero value directly
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Interface {
		g.writef("%s := reflect.TypeOf((*%s)(nil)).Elem()\n", resultId, zeroValueExpr)
	} else {
		g.writef("%s := reflect.TypeOf(%s)\n", resultId, zeroValueExpr)
	}

	return resultId
}

// generateZeroValueExpr generates a Go expression that creates a zero value
// of the given type, handling package imports as needed
func (g *Generator) generateZeroValueExpr(t reflect.Type) string {
	// TODO: review this LLM slop. for numeric types, return the type
	// cast of 0 with the (possibly aliased) type name
	switch {
	case t == reflect.TypeOf(lang.NewChar('a')):
		return "lang.NewChar(0)"
	case t == reflect.TypeOf(time.Duration(0)):
		alias := g.addImportWithAlias("time")
		return fmt.Sprintf("%s.Duration(0)", alias)
	}

	switch t.Kind() {
	case reflect.Bool:
		return "false"
	case reflect.Int:
		return "int(0)"
	case reflect.Int8:
		return "int8(0)"
	case reflect.Int16:
		return "int16(0)"
	case reflect.Int32:
		return "int32(0)"
	case reflect.Int64:
		return "int64(0)"
	case reflect.Uint:
		return "uint(0)"
	case reflect.Uint8:
		return "uint8(0)"
	case reflect.Uint16:
		return "uint16(0)"
	case reflect.Uint32:
		return "uint32(0)"
	case reflect.Uint64:
		return "uint64(0)"
	case reflect.Uintptr:
		return "uintptr(0)"
	case reflect.Float32:
		return "float32(0)"
	case reflect.Float64:
		return "float64(0)"
	case reflect.Complex64:
		return "complex64(0)"
	case reflect.Complex128:
		return "complex128(0)"
	case reflect.String:
		return `""`
	case reflect.Array:
		elemExpr := g.getTypeString(t.Elem())
		return fmt.Sprintf("[%d]%s{}", t.Len(), elemExpr)
	case reflect.Slice:
		elemType := g.getTypeString(t.Elem())
		return fmt.Sprintf("[]%s(nil)", elemType)
	case reflect.Map:
		keyType := g.getTypeString(t.Key())
		elemType := g.getTypeString(t.Elem())
		return fmt.Sprintf("map[%s]%s(nil)", keyType, elemType)
	case reflect.Chan:
		elemType := g.getTypeString(t.Elem())
		switch t.ChanDir() {
		case reflect.RecvDir:
			return fmt.Sprintf("(<-chan %s)(nil)", elemType)
		case reflect.SendDir:
			return fmt.Sprintf("(chan<- %s)(nil)", elemType)
		default:
			return fmt.Sprintf("(chan %s)(nil)", elemType)
		}
	case reflect.Func:
		return g.getTypeString(t) + "(nil)"
	case reflect.Interface:
		// For interfaces, return the type string for use with (*T)(nil).Elem()
		return g.getTypeString(t)
	case reflect.Ptr:
		elemType := g.getTypeString(t.Elem())
		return fmt.Sprintf("(*%s)(nil)", elemType)
	case reflect.Struct:
		// For structs, return the type string for use with (*T)(nil).Elem()
		return g.getTypeString(t)
	default:
		// Fallback: try to use the type string directly
		return g.getTypeString(t) + "{}"
	}
}

// getTypeString returns a string representation of the type suitable for use
// in Go code, adding package imports as necessary
func (g *Generator) getTypeString(t reflect.Type) string {
	// Handle unnamed types
	if t.Name() == "" {
		switch t.Kind() {
		case reflect.Slice:
			return "[]" + g.getTypeString(t.Elem())
		case reflect.Array:
			return fmt.Sprintf("[%d]%s", t.Len(), g.getTypeString(t.Elem()))
		case reflect.Map:
			return fmt.Sprintf("map[%s]%s", g.getTypeString(t.Key()), g.getTypeString(t.Elem()))
		case reflect.Ptr:
			fmt.Printf("Pointer to %s\n", t.Elem().String())
			fmt.Println("returning", "*"+g.getTypeString(t.Elem()))
			return "*" + g.getTypeString(t.Elem())
		case reflect.Chan:
			switch t.ChanDir() {
			case reflect.RecvDir:
				return "<-chan " + g.getTypeString(t.Elem())
			case reflect.SendDir:
				return "chan<- " + g.getTypeString(t.Elem())
			default:
				return "chan " + g.getTypeString(t.Elem())
			}
		case reflect.Interface:
			return "any"
		default:
			// For basic types like int, string, etc.
			// Note: We can't use t.String() directly here because it might
			// return "package.Type" format which is not what we want
			return t.Kind().String()
		}
	}

	// Handle named types
	pkgPath := t.PkgPath()
	if pkgPath == "" {
		// Built-in type or type from current package
		// For built-in types, Name() might be empty, so use String() as fallback
		if t.Name() != "" {
			return t.Name()
		}
		return t.String()
	}

	// Import the package and get an alias
	alias := g.addImportWithAlias(pkgPath)
	return alias + "." + t.Name()
}

func (g *Generator) generateAtomValue(atom *lang.Atom) string {
	// Allocate a variable to hold the atom
	atomVar := g.allocateTempVar()

	// Generate the initial value
	initialValue := g.generateValue(atom.Deref())

	var metaVar string
	if meta := atom.Meta(); meta != nil {
		metaVar = g.generateValue(meta)
	}

	if metaVar == "" {
		g.writef("%s := lang.NewAtom(%s)\n", atomVar, initialValue)
	} else {
		g.writef("%s := lang.NewAtomWithMeta(%s, %s)\n", atomVar, initialValue, metaVar)
	}

	return atomVar
}

func (g *Generator) generateRefValue(ref *lang.Ref) string {
	refVar := g.allocateTempVar()
	initialValue := g.generateValue(ref.Deref())
	g.writef("%s := lang.NewRef(%s)\n", refVar, initialValue)
	return refVar
}

// generateMapValue generates Go code for a Clojure map
func (g *Generator) generateMapValue(m lang.IPersistentMap) string {
	var buf bytes.Buffer
	if m.Count()*2 > lang.PersistentArrayMapInlineKeyValueCount {
		buf.WriteString("lang.NewMapUniqueKeys(")
	} else {
		buf.WriteString("lang.NewMap(")
	}

	// Iterate through the map entries
	for seq := m.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First()
		key := lang.First(entry)
		value, _ := lang.Nth(entry, 1)
		keyVar := g.generateValue(key)
		valueVar := g.generateValue(value)
		buf.WriteString(keyVar + ", " + valueVar + ", ")
	}

	// Remove trailing comma and space
	if m.Count() > 0 {
		buf.Truncate(buf.Len() - 2)
	}

	buf.WriteString(")")
	return buf.String()
}

// generateVectorValue generates Go code for a Clojure vector
func (g *Generator) generateVectorValue(v lang.IPersistentVector) string {
	var buf bytes.Buffer
	buf.WriteString("lang.NewVector(")

	// Iterate through the vector elements
	for i := 0; i < v.Count(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		element := v.Nth(i)
		elementVar := g.generateValue(element)
		buf.WriteString(elementVar)
	}

	buf.WriteString(")")
	return buf.String()
}

func (g *Generator) generateBigDecimalValue(bd *lang.BigDecimal) string {
	bigFloat := bd.ToBigFloat()
	blob, err := bigFloat.GobEncode()
	if err != nil {
		panic(fmt.Sprintf("failed to encode big.Float: %v", err))
	}
	// nice compact hex literal
	hexBlob := hex.EncodeToString(blob)

	resultId := g.allocateTempVar()

	hexAlias := g.addImportWithAlias("encoding/hex")
	bigAlias := g.addImportWithAlias("math/big")

	g.writef(`%s := lang.NewBigDecimalFromBigFloat((func() *%s.Float {
  var z %s.Float
  b, _ := %s.DecodeString("%s")
  if err := z.GobDecode(b); err != nil { panic(err) }
  return &z
})())
`, resultId, bigAlias, bigAlias, hexAlias, hexBlob)

	return resultId
}

func generateBigIntValue(value *lang.BigInt) string {
	return fmt.Sprintf(
		`func() *lang.BigInt { value, err := lang.NewBigInt(%q); if err != nil { panic(err) }; return value }()`,
		value.String(),
	)
}

func generateRatioValue(value *lang.Ratio) string {
	numerator := lang.NewBigIntFromGoBigInt(value.Numerator())
	denominator := lang.NewBigIntFromGoBigInt(value.Denominator())
	return fmt.Sprintf(
		"lang.NewRatioBigInt(%s, %s)",
		generateBigIntValue(numerator),
		generateBigIntValue(denominator),
	)
}

// generateSetValue generates Go code for a Clojure set
func (g *Generator) generateSetValue(s lang.IPersistentSet) string {
	var buf bytes.Buffer
	buf.WriteString("lang.NewSet(")

	idx := 0

	// Iterate through the set elements
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		if idx > 0 {
			buf.WriteString(", ")
		}
		idx++
		element := seq.First()
		elementVar := g.generateValue(element)
		buf.WriteString(elementVar)
	}

	buf.WriteString(")")

	return buf.String()
}

func (g *Generator) generateMultiFn(mf *lang.MultiFn) string {
	// Allocate a variable for the MultiFn
	mfVar := g.allocateTempVar()

	// Generate the dispatch function
	dispatchFnVar := g.generateValue(mf.GetDispatchFn())

	// Generate the default dispatch value
	defaultValVar := g.generateValue(mf.GetDefaultDispatchVal())

	// Generate the hierarchy reference
	hierarchyVar := g.generateValue(mf.GetHierarchy())

	// Create the MultiFn
	g.writef("// MultiFn %s\n", mf.GetName())
	g.writef("%s := lang.NewMultiFn(%#v, %s, %s, %s)\n",
		mfVar, mf.GetName(), dispatchFnVar, defaultValVar, hierarchyVar)

	// Add all methods from the method table. Skip entries that
	// lang.NewMultiFn already seeds via registerWellKnownMethods: their
	// method values are opaque Go FnFuncs (un-generatable), and the
	// compiled binary re-seeds them on construction.
	methodTable := mf.GetMethodTable()
	if methodTable != nil && methodTable.Count() > 0 {
		for seq := lang.Seq(methodTable); seq != nil; seq = seq.Next() {
			entry := seq.First().(lang.IMapEntry)
			dispatchVal := entry.Key()
			method := entry.Val()

			if lang.IsAutoRegisteredMethod(mf.GetName(), dispatchVal, method) {
				continue
			}

			dispatchValVar := g.generateValue(dispatchVal)
			methodVar := g.generateValue(method)

			g.writef("%s.AddMethod(%s, %s)\n", mfVar, dispatchValVar, methodVar)
		}
	}

	// Add all preferences from the prefer table
	preferTable := mf.PreferTable()
	if preferTable != nil && preferTable.Count() > 0 {
		for seq := lang.Seq(preferTable); seq != nil; seq = seq.Next() {
			entry := seq.First().(lang.IMapEntry)
			dispatchValX := entry.Key()
			prefs := entry.Val()

			// Iterate through the set of preferred values
			for prefSeq := lang.Seq(prefs); prefSeq != nil; prefSeq = prefSeq.Next() {
				dispatchValY := prefSeq.First()

				dispatchValXVar := g.generateValue(dispatchValX)
				dispatchValYVar := g.generateValue(dispatchValY)

				g.writef("%s.PreferMethod(%s, %s)\n", mfVar, dispatchValXVar, dispatchValYVar)
			}
		}
	}

	return mfVar
}

func (g *Generator) generateFnFunc(fn lang.FnFunc) string {
	name := "unknown"
	if runtimeFn := goruntime.FuncForPC(reflect.ValueOf(fn).Pointer()); runtimeFn != nil {
		name = runtimeFn.Name()
	}
	panic(fmt.Sprintf("cannot generate opaque go function value %s", name))
}

func (g *Generator) generateFn(fn *Fn) string {
	// Save and restore current environment
	prevEnv := g.currentFnEnv
	// Runtime function values carry their captured environment. Functions
	// constructed directly from nested AST nodes do not; in that case they
	// inherit the environment of the function currently being generated.
	if env := fn.GetEnvironment(); env != nil {
		g.currentFnEnv = env
	}
	defer func() { g.currentFnEnv = prevEnv }()

	astNode := fn.ASTNode()
	fnNode := astNode.Sub.(*ast.FnNode)

	// Determine if we can use a fixed-arity FnFuncN (0-20 args, single method,
	// non-variadic). fixedArity == -1 means fall back to FnFunc.
	fixedArity := -1
	if len(fnNode.Methods) == 1 && !fnNode.IsVariadic {
		mn := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		if mn.FixedArity <= 20 {
			fixedArity = mn.FixedArity
		}
	}
	arityDispatch := len(fnNode.Methods) > 1 || fnNode.IsVariadic
	smallFixed := true
	for _, method := range fnNode.Methods {
		mn := method.Sub.(*ast.FnMethodNode)
		if !mn.IsVariadic && mn.FixedArity > 4 {
			smallFixed = false
			break
		}
	}

	// Allocate a variable to return the function
	fnVar := g.allocateTempVar()

	// Declare with the appropriate type.
	// FnFuncN for supported non-variadic single-arity functions eliminates
	// []any heap allocation at call sites that use ApplyN.
	fnType := "lang.FnFunc"
	if fixedArity >= 0 {
		fnType = fmt.Sprintf("lang.FnFunc%d", fixedArity)
	} else if arityDispatch {
		fnType = "lang.ArityFn"
	}
	// declare it now to make sure it's in the scope of the caller
	// we may add a nested scope to declare the function in to keep a
	// scoped variable for the function itself, if the function is named
	g.writef("var %s %s\n", fnVar, fnType)

	// Push a new scope for the function definition
	g.pushVarScope()
	defer g.popVarScope()

	if fnNode.Local != nil {
		// If there's a local binding, use that name
		localNode := fnNode.Local.Sub.(*ast.BindingNode)
		if fnName := localNode.Name.Name(); fnName != "" {
			g.writef("{ // function %s\n", fnName)
			defer g.writef("}\n")

			namedFnVar := g.allocateLocal(fnName)
			g.writef("var %s %s\n", namedFnVar, fnType)
			defer func() {
				g.writeAssign(namedFnVar, fnVar)
				g.writeAssign("_", namedFnVar) // Prevent unused variable warning
			}()
		}
	}

	if fixedArity >= 0 {
		// Supported single arity: emit FnFuncN with direct named params.
		methodNode := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		paramNames := fixedParamNames(fixedArity)
		if !g.generateInt64SpecializedFixedFn(fn, fnVar, methodNode, paramNames) &&
			!g.generateFloat64SpecializedFixedFn(fn, fnVar, methodNode, paramNames) {
			sig := ""
			if fixedArity > 0 {
				sig = strings.Join(paramNames, ", ") + " any"
			}
			g.writef("%s = lang.FnFunc%d(func(%s) any {\n", fnVar, fixedArity, sig)
			g.generateFnMethodFixed(methodNode, paramNames)
			g.writef("})\n")
		}
	} else if arityDispatch {
		var fixedMethods [5]*ast.FnMethodNode
		var fixedOther []*ast.FnMethodNode
		var variadicMethod *ast.FnMethodNode
		for _, method := range fnNode.Methods {
			methodNode := method.Sub.(*ast.FnMethodNode)
			if methodNode.IsVariadic {
				variadicMethod = methodNode
			} else if methodNode.FixedArity < len(fixedMethods) {
				fixedMethods[methodNode.FixedArity] = methodNode
			} else {
				fixedOther = append(fixedOther, methodNode)
			}
		}

		target := g.specializationTarget
		if target != nil && target.fn == fn {
			for _, method := range fnNode.Methods {
				methodNode := method.Sub.(*ast.FnMethodNode)
				if methodNode.IsVariadic ||
					methodNode.FixedArity < 0 ||
					methodNode.FixedArity >= len(target.directArityVars) {
					continue
				}
				slot := target.directArityVars[methodNode.FixedArity]
				if slot == "" {
					continue
				}
				g.writef("%s = ", slot)
				g.generateFixedMethodFnValue(methodNode)
				g.writef("\n")
			}
		}

		if smallFixed {
			g.writef("%s = lang.NewArityFn(\n", fnVar)
			for arity, methodNode := range fixedMethods {
				if methodNode == nil {
					g.writef("nil,\n")
					continue
				}
				if target != nil && target.fn == fn {
					if slot := target.directArityVars[arity]; slot != "" {
						g.writef("%s,\n", slot)
						continue
					}
				}
				g.generateFixedMethodFn(methodNode)
			}
		} else {
			g.writef("%s = lang.NewArityFnMethods(\n", fnVar)
			g.writef("map[int]lang.IFn{\n")
			for arity, methodNode := range fixedMethods {
				if methodNode == nil {
					continue
				}
				g.writef("%d: ", arity)
				if target != nil && target.fn == fn {
					if slot := target.directArityVars[arity]; slot != "" {
						g.writef("%s,\n", slot)
						continue
					}
				}
				g.generateFixedMethodFn(methodNode)
			}
			for _, methodNode := range fixedOther {
				g.writef("%d: ", methodNode.FixedArity)
				if target != nil && target.fn == fn &&
					methodNode.FixedArity < len(target.directArityVars) {
					if slot := target.directArityVars[methodNode.FixedArity]; slot != "" {
						g.writef("%s,\n", slot)
						continue
					}
				}
				g.generateFixedMethodFn(methodNode)
			}
			g.writef("},\n")
		}
		if variadicMethod == nil {
			g.writef("nil,\n0,\n")
		} else {
			g.writef("lang.NewVariadicFn(%d, func(args []any, rest lang.ISeq) any {\n",
				variadicMethod.FixedArity)
			g.generateFnMethodSplit(variadicMethod, "args", "rest")
			g.writef("}),\n%d,\n", variadicMethod.FixedArity)
		}
		g.writef(")\n")
	} else if len(fnNode.Methods) == 1 && !fnNode.IsVariadic {
		// Single-arity 5+: emit FnFunc with args slice
		method := fnNode.Methods[0]
		methodNode := method.Sub.(*ast.FnMethodNode)

		g.writef("%s = lang.NewFnFunc(func(args ...any) any {\n", fnVar)

		// Check arity
		g.writef("checkArity(args, %d)\n", methodNode.FixedArity)

		// Generate method body
		g.generateFnMethod(methodNode, "args")

		g.writef("})\n")
	}

	// defn uses :rettag to communicate a return hint to the compiler. It is
	// not runtime function metadata, so do not serialize it into the AOT
	// value. Preserve any metadata explicitly attached to the function.
	if meta := runtimeFunctionMeta(fn.Meta()); meta != nil {
		metaVar := g.generateValue(meta)
		g.writeAssign(fnVar, fmt.Sprintf("%s.WithMeta(%s).(%s)", fnVar, metaVar, fnType))
	}

	// Return the function variable
	return fnVar
}

func (g *Generator) generateFixedMethodFn(methodNode *ast.FnMethodNode) {
	g.generateFixedMethodFnValue(methodNode)
	g.writef(",\n")
}

func (g *Generator) generateFixedMethodFnValue(methodNode *ast.FnMethodNode) {
	arity := methodNode.FixedArity
	if arity <= 20 {
		paramNames := fixedParamNames(arity)
		sig := ""
		if arity > 0 {
			sig = strings.Join(paramNames, ", ") + " any"
		}
		g.writef("lang.FnFunc%d(func(%s) any {\n", arity, sig)
		g.generateFnMethodFixed(methodNode, paramNames)
		g.writef("})")
		return
	}

	g.writef("lang.NewFnFunc(func(args ...any) any {\n")
	g.writef("checkArity(args, %d)\n", arity)
	g.generateFnMethod(methodNode, "args")
	g.writef("})")
}

func fixedParamNames(arity int) []string {
	names := make([]string, arity)
	for i := range names {
		names[i] = fmt.Sprintf("p%d", i)
	}
	return names
}

func runtimeFunctionMeta(meta lang.IPersistentMap) lang.IPersistentMap {
	if meta == nil {
		return nil
	}
	meta = meta.Without(lang.NewKeyword("rettag"))
	if meta.Count() == 0 {
		return nil
	}
	return meta
}

// generateFnMethod generates the body of a function method
func (g *Generator) generateFnMethod(methodNode *ast.FnMethodNode, argsVar string) {
	// Push a new scope for the method body
	g.pushVarScope()
	defer g.popVarScope()

	paramVars := make([]string, methodNode.FixedArity)

	// Bind parameters
	for i, param := range methodNode.Params {
		paramNode := param.Sub.(*ast.BindingNode)
		paramVar := g.allocateLocal(paramNode.Name.Name())

		if i < methodNode.FixedArity {
			// Regular parameter
			g.writef("%s := %s[%d]\n", paramVar, argsVar, i)
			g.writeAssign("_", paramVar) // Prevent unused variable warning
			paramVars[i] = paramVar
		} else {
			// Variadic parameter - collect rest args
			g.writef("restArgs := %s[%d:]\n", argsVar, methodNode.FixedArity)
			g.writef("var %s any\n", paramVar)
			g.writef("if len(restArgs) > 0 {\n")
			g.writef("  %s = lang.NewList(restArgs...)\n", paramVar)
			g.writef("}\n")
			g.writeAssign("_", paramVar) // Prevent unused variable warning
			paramVars = append(paramVars, paramVar)
		}
	}

	// Add a recur label
	if methodNode.LoopID != nil && nodeRecurs(methodNode.Body, methodNode.LoopID.Name()) {
		g.writef("recur_%s:\n", methodNode.LoopID.Name())

		g.pushRecurContext(methodNode.LoopID, paramVars, true)
		defer g.popRecurContext()
	}

	// Generate the body
	bodyVar := g.generateASTNode(methodNode.Body)
	if bodyVar != "" {
		g.writef("return %s\n", bodyVar)
	}
	// If bodyVar is empty (e.g., from throw), no return is generated
}

// generateFnMethodSplit generates a variadic method body with its fixed
// parameters in a slice and its rest parameter already represented as an ISeq.
func (g *Generator) generateFnMethodSplit(
	methodNode *ast.FnMethodNode,
	fixedVar string,
	restVar string,
) {
	g.pushVarScope()
	defer g.popVarScope()

	paramVars := make([]string, methodNode.FixedArity)

	for i, param := range methodNode.Params {
		paramNode := param.Sub.(*ast.BindingNode)
		paramVar := g.allocateLocal(paramNode.Name.Name())

		if i < methodNode.FixedArity {
			g.writef("%s := %s[%d]\n", paramVar, fixedVar, i)
			g.writeAssign("_", paramVar)
			paramVars[i] = paramVar
		} else {
			g.writef("var %s any = %s\n", paramVar, restVar)
			g.writeAssign("_", paramVar)
			paramVars = append(paramVars, paramVar)
		}
	}

	if methodNode.LoopID != nil && nodeRecurs(methodNode.Body, methodNode.LoopID.Name()) {
		g.writef("recur_%s:\n", methodNode.LoopID.Name())

		g.pushRecurContext(methodNode.LoopID, paramVars, true)
		defer g.popRecurContext()
	}

	bodyVar := g.generateASTNode(methodNode.Body)
	if bodyVar != "" {
		g.writef("return %s\n", bodyVar)
	}
}

// generateFnMethodFixed generates a function method body where parameters are
// bound directly from named Go function params instead of indexing an args slice.
// Used for FnFuncN (0-4 arity) functions to avoid []any allocation.
// paramVarNames contains the Go parameter variable names (e.g. ["p0", "p1"]).
func (g *Generator) generateFnMethodFixed(methodNode *ast.FnMethodNode, paramVarNames []string) {
	// Push a new scope for the method body
	g.pushVarScope()
	defer g.popVarScope()

	paramVars := make([]string, len(paramVarNames))

	// Bind parameters directly from named Go function params
	for i, param := range methodNode.Params {
		paramNode := param.Sub.(*ast.BindingNode)
		paramVar := g.allocateLocal(paramNode.Name.Name())
		g.writef("%s := %s\n", paramVar, paramVarNames[i])
		g.writeAssign("_", paramVar) // Prevent unused variable warning
		paramVars[i] = paramVar
	}

	// Add a recur label
	if methodNode.LoopID != nil && nodeRecurs(methodNode.Body, methodNode.LoopID.Name()) {
		g.writef("recur_%s:\n", methodNode.LoopID.Name())

		g.pushRecurContext(methodNode.LoopID, paramVars, true)
		defer g.popRecurContext()
	}

	// Generate the body
	bodyVar := g.generateASTNode(methodNode.Body)
	if bodyVar != "" {
		g.writef("return %s\n", bodyVar)
	}
	// If bodyVar is empty (e.g., from throw), no return is generated
}

////////////////////////////////////////////////////////////////////////////////
// AST Node Generation

// generateASTNode generates code for an AST node
func (g *Generator) generateASTNode(node *ast.Node) (res string) {
	switch node.Op {
	case ast.OpDef:
		return g.generateDef(node)
	case ast.OpLetFn:
		return g.generateLetFn(node)
	case ast.OpGo:
		return g.generateGo(node)
	case ast.OpSetBang:
		return g.generateSetBang(node)
	case ast.OpCase:
		return g.generateCase(node)
	case ast.OpTry:
		return g.generateTry(node)
	case ast.OpThrow:
		return g.generateThrow(node)
	case ast.OpConst:
		constNode := node.Sub.(*ast.ConstNode)
		if constNode.HostSymbol != nil {
			// Host classes are synthetic JVM-compatible values, not Go
			// package exports (for example java.lang.Math). Preserve their
			// class identity instead of attempting to import the Java name.
			switch constNode.Value.(type) {
			case *lang.Class, reflect.Type:
			default:
				return g.generateGoExportedName(constNode.HostSymbol.FullName())
			}
		}
		// A compiled Clojure regex is a constant object: repeated evaluation of
		// one literal returns the same object, while two equal literal
		// occurrences remain distinct. Lift by pointer identity to preserve
		// both halves of that contract.
		if _, ok := constNode.Value.(*regexp.Regexp); ok {
			return g.liftValue(constNode.Value)
		}
		return g.generateValue(constNode.Value)
	case ast.OpVector:
		return g.generateVector(node)
	case ast.OpMap:
		return g.generateMap(node)
	case ast.OpSet:
		return g.generateSet(node)
	case ast.OpLocal:
		localNode := node.Sub.(*ast.LocalNode)
		// Look up the variable in our scope
		return g.getLocal(localNode.Name.Name())
	case ast.OpDo:
		return g.generateDo(node)
	case ast.OpLet:
		return g.generateLet(node, false)
	case ast.OpLoop:
		return g.generateLet(node, true)
	case ast.OpIf:
		return g.generateIf(node)
	case ast.OpInvoke:
		return g.generateInvoke(node)
	case ast.OpKeywordLookup:
		lookup := node.Sub.(*ast.KeywordLookupNode)
		target := g.generateASTNode(lookup.Target)
		result := g.allocateTempVar()
		if g.aotRecordHasField(keywordName(lookup.Keyword)) {
			fallback := "nil"
			if lookup.Default != nil {
				fallback = g.generateASTNode(lookup.Default)
			}
			helper := g.allocKeywordLookupHelper(keywordName(lookup.Keyword))
			g.writef(
				"%s := %s(%s, %s)\n",
				result,
				helper,
				target,
				fallback,
			)
		} else {
			keyword := g.generateValue(lookup.Keyword)
			if lookup.Default == nil {
				g.writef("%s := %s.Invoke1(%s)\n", result, keyword, target)
			} else {
				fallback := g.generateASTNode(lookup.Default)
				g.writef(
					"%s := %s.Invoke2(%s, %s)\n",
					result,
					keyword,
					target,
					fallback,
				)
			}
		}
		return result
	case ast.OpAssoc:
		assoc := node.Sub.(*ast.AssocNode)
		target := g.generateASTNode(assoc.Target)
		staticNames, staticKeys := staticKeywordMapNames(keysFromAssoc(assoc))
		useRecordHelper := staticKeys && g.aotRecordHasFields(staticNames)
		keys := make([]string, len(assoc.Entries))
		values := make([]string, len(assoc.Entries))
		for i, entry := range assoc.Entries {
			if !useRecordHelper {
				keys[i] = g.generateASTNode(entry.Key)
			}
			values[i] = g.generateASTNode(entry.Val)
		}
		if useRecordHelper {
			helper := g.allocKeywordAssocHelper(staticNames)
			result := g.allocateTempVar()
			g.writef(
				"%s := %s(%s, %s)\n",
				result,
				helper,
				target,
				strings.Join(values, ", "),
			)
			return result
		}
		result := g.allocateTempVar()
		g.writef("var %s any = %s\n", result, target)
		for i := range keys {
			g.writef(
				"%s = lang.Assoc(%s, %s, %s)\n",
				result,
				result,
				keys[i],
				values[i],
			)
		}
		return result
	case ast.OpReplaceLast:
		replace := node.Sub.(*ast.ReplaceLastNode)
		collection := g.generateASTNode(replace.Collection)
		plan := g.allocateTempVar()
		g.writef("%s := runtime.PrepareReplaceLast(%s)\n", plan, collection)
		value := g.generateASTNode(replace.Value)
		result := g.allocateTempVar()
		g.writef("%s := %s.Finish(%s)\n", result, plan, value)
		return result
	case ast.OpVar:
		return g.generateVarDeref(node)
	case ast.OpRecur:
		return g.generateRecur(node)
	case ast.OpGoBuiltin:
		return g.generateGoBuiltin(node)
	case ast.OpWithMeta:
		return g.generateWithMeta(node)
	case ast.OpMaybeClass:
		return g.generateMaybeClass(node)
	case ast.OpQuote:
		return g.generateValue(node.Sub.(*ast.QuoteNode).Expr.Sub.(*ast.ConstNode).Value)
	case ast.OpFn:
		return g.generateFn(NewFn(node, nil))
	case ast.OpHostCall:
		return g.generateHostCall(node)
	case ast.OpHostInterop:
		return g.generateHostInterop(node)
	case ast.OpMaybeHostForm:
		return g.generateMaybeHostForm(node)
	case ast.OpTheVar:
		return g.generateTheVar(node)
	case ast.OpNew:
		return g.generateNew(node)
	default:
		panic(fmt.Sprintf("unsupported AST node type %T", node.Sub))
	}
}

func (g *Generator) generateDef(node *ast.Node) string {
	defNode := node.Sub.(*ast.DefNode)
	init := defNode.Init
	vr := defNode.Var
	meta := defNode.Meta

	vrVar := g.allocVarVar(vr.Namespace().Name().String(), vr.Symbol().String())
	if !lang.IsNil(meta) {
		metaVar := g.generateASTNode(meta)
		g.writef("%s.SetMeta(%s.(lang.IPersistentMap))\n", vrVar, metaVar)
		// SetDynamic if dynamic kw true in meta
		g.writef("if runtime.RT.BooleanCast(lang.Get(%s, lang.KWDynamic)) {\n", metaVar)
		g.writef("\t%s.SetDynamic()\n", vrVar)
		g.writef("}\n")
	}

	if lang.IsNil(init) {
		return vrVar // No initialization
	}
	initVar := g.generateASTNode(init)
	g.writef("%s.BindRoot(%s)\n", vrVar, initVar)

	return vrVar
}

func (g *Generator) generateGo(node *ast.Node) string {
	goNode := node.Sub.(*ast.GoNode)

	invokeNode := goNode.Invoke.Sub.(*ast.InvokeNode)
	fn := invokeNode.Fn
	args := invokeNode.Args

	// Generate the function expression
	fnExpr := g.generateASTNode(fn)

	// Generate the arguments
	var argExprs []string
	for _, arg := range args {
		argExprs = append(argExprs, g.generateASTNode(arg))
	}

	g.writef("go lang.Apply(%s, []any{%s})\n", fnExpr, strings.Join(argExprs, ", "))
	return "nil" // starting a goroutine returns nil
}

// generateVarDeref generates code for a Var dereference
func (g *Generator) generateVarDeref(node *ast.Node) string {
	varNode := node.Sub.(*ast.VarNode)

	varNamespace := varNode.Var.Namespace()
	varSymbol := varNode.Var.Symbol()

	// Look up the var variable
	varId := g.allocVarVar(varNamespace.Name().String(), varSymbol.String())
	// add as a dependency to the current value init if we're in one
	if g.currentValueInit != nil && varId != g.currentValueInit.name {
		g.currentValueInit.deps[varId] = struct{}{}
	}

	resultId := g.allocateTempVar()
	g.writef("%s := checkDerefVar(%s)\n", resultId, varId)

	return resultId
}

// generateInvoke generates code for an Invoke node
func (g *Generator) generateInvoke(node *ast.Node) string {
	invokeNode := node.Sub.(*ast.InvokeNode)
	if result, ok := g.generateLocalAtomInvoke(invokeNode); ok {
		return result
	}
	if plan := analyzeReducePipeline(invokeNode); plan != nil {
		return g.generateAOTReducePipeline(invokeNode, plan)
	}
	return g.generateInvokeDefault(invokeNode)
}

func (g *Generator) generateInvokeDefault(invokeNode *ast.InvokeNode) string {
	// A package-level Go function whose parameters are all `any` can be
	// emitted as a direct call. This preserves static return types such as
	// ISeq while avoiding reflection in lang.Apply.
	var fnExpr string
	directCall := false
	directKeywordCall := false
	recordTarget := g.aotRecordInvokeTarget(invokeNode)
	aotTarget := g.aotInvokeTarget(invokeNode)
	var externalTarget *aotExternalCallTarget
	if recordTarget == nil {
		externalTarget = g.aotExternalInvokeTarget(invokeNode)
	}
	if invokeNode.Fn.Op == ast.OpConst {
		value := invokeNode.Fn.Sub.(*ast.ConstNode).Value
		if _, ok := value.(lang.Keyword); ok &&
			(len(invokeNode.Args) == 1 || len(invokeNode.Args) == 2) {
			fnExpr = g.generateValue(value)
			directKeywordCall = true
		}
		if typ := reflect.TypeOf(value); typ != nil &&
			typ.Kind() == reflect.Func &&
			!typ.IsVariadic() &&
			typ.NumIn() == len(invokeNode.Args) &&
			typ.NumOut() == 1 {
			anyType := reflect.TypeOf((*any)(nil)).Elem()
			directCall = true
			for i := 0; i < typ.NumIn(); i++ {
				if typ.In(i) != anyType {
					directCall = false
					break
				}
			}
			if directCall {
				fnExpr = g.generateValue(value)
			}
		}
	}
	if !directCall && !directKeywordCall && recordTarget == nil &&
		aotTarget == nil && externalTarget == nil {
		// Generate the general function expression before its arguments.
		fnExpr = g.generateASTNode(invokeNode.Fn)
	}

	var aotFast, aotFallbackFn string
	if aotTarget != nil && !aotTarget.directLinked {
		varNode := invokeNode.Fn.Sub.(*ast.VarNode)
		varID := g.allocVarVar(
			varNode.Var.Namespace().Name().String(),
			varNode.Var.Symbol().String(),
		)
		if g.currentValueInit != nil && varID != g.currentValueInit.name {
			g.currentValueInit.deps[varID] = struct{}{}
		}
		aotFast = g.allocateTempVar()
		g.writef("%s := %s.RootVersion() == %s && !%s.IsMacro()\n",
			aotFast, varID, aotTarget.rootVersionVar, varID)
		aotFallbackFn = g.allocateTempVar()
		g.writef("var %s any\n", aotFallbackFn)
		g.writef("if !%s {\n", aotFast)
		g.writef("%s = checkDerefVar(%s)\n", aotFallbackFn, varID)
		g.writef("}\n")
	}

	// Generate the arguments
	var argExprs []string
	_, staticInstance := g.staticInstanceCall(
		invokeNode,
		[]string{"", "value"},
	)
	skipInstanceType := staticInstance &&
		((aotTarget != nil && aotTarget.directLinked) ||
			(externalTarget != nil &&
				externalTarget.directLinked &&
				externalTarget.intrinsic == "instance?"))
	for index, arg := range invokeNode.Args {
		if skipInstanceType && index == 0 {
			argExprs = append(argExprs, "")
			continue
		}
		argExprs = append(argExprs, g.generateASTNode(arg))
	}

	// Allocate a result variable for the invocation
	resultVar := g.allocateTempVar()
	if recordTarget != nil {
		factory := recordTarget.record.constructor
		if recordTarget.fromMap {
			factory = recordTarget.record.mapFactory
		}
		g.writef("%s := %s(%s)\n",
			resultVar,
			factory,
			strings.Join(argExprs, ", "),
		)
		return resultVar
	}
	if directCall {
		g.writef("%s := %s(%s)\n", resultVar, fnExpr, strings.Join(argExprs, ", "))
		return resultVar
	}
	if directKeywordCall {
		g.writef("%s := %s.Invoke%d(%s)\n",
			resultVar,
			fnExpr,
			len(argExprs),
			strings.Join(argExprs, ", "),
		)
		return resultVar
	}
	if aotTarget != nil {
		if aotTarget.directLinked {
			if instanceCall, ok := g.staticInstanceCall(invokeNode, argExprs); ok {
				g.writef("%s := %s\n", resultVar, instanceCall)
				return resultVar
			}
			if slot := aotTarget.directArityVars[len(argExprs)]; slot != "" {
				g.writef("%s := %s(%s)\n",
					resultVar,
					slot,
					strings.Join(argExprs, ", "),
				)
			} else if aotTarget.arityDispatch {
				g.writef("%s := %s.Invoke%d(%s)\n",
					resultVar,
					aotTarget.directFnVar,
					len(argExprs),
					strings.Join(argExprs, ", "),
				)
			} else {
				g.writef("%s := %s(%s)\n",
					resultVar, aotTarget.directFnVar, strings.Join(argExprs, ", "))
			}
			return resultVar
		}
		if instanceCall, ok := g.staticInstanceCall(invokeNode, argExprs); ok {
			g.writef("var %s any\n", resultVar)
			g.writef("if %s {\n", aotFast)
			g.writef("%s = %s\n", resultVar, instanceCall)
			g.writef("} else {\n")
			g.generateApply(resultVar, aotFallbackFn, argExprs, false)
			g.writef("}\n")
			return resultVar
		}
		g.writef("var %s any\n", resultVar)
		g.writef("if %s {\n", aotFast)
		if slot := aotTarget.directArityVars[len(argExprs)]; slot != "" {
			g.writef("%s = %s(%s)\n",
				resultVar,
				slot,
				strings.Join(argExprs, ", "),
			)
		} else if aotTarget.arityDispatch {
			g.writef("%s = %s.Invoke%d(%s)\n",
				resultVar,
				aotTarget.directFnVar,
				len(argExprs),
				strings.Join(argExprs, ", "),
			)
		} else {
			g.writef("%s = %s(%s)\n",
				resultVar, aotTarget.directFnVar, strings.Join(argExprs, ", "))
		}
		g.writef("} else {\n")
		g.generateApply(resultVar, aotFallbackFn, argExprs, false)
		g.writef("}\n")
		return resultVar
	}
	if externalTarget != nil {
		if externalTarget.intrinsic != "" {
			if externalTarget.directLinked {
				g.writef("%s := %s\n",
					resultVar,
					g.aotExternalIntrinsicCall(
						externalTarget.intrinsic,
						invokeNode,
						argExprs,
					),
				)
				return resultVar
			}
			varNode := invokeNode.Fn.Sub.(*ast.VarNode)
			varID := g.allocVarVar(
				varNode.Var.Namespace().Name().String(),
				varNode.Var.Symbol().String(),
			)
			g.writef("var %s any\n", resultVar)
			g.writef("if %s && %s.RootVersion() == %s {\n",
				externalTarget.defaultVar,
				varID,
				externalTarget.rootVersionVar,
			)
			g.writef("%s = %s\n",
				resultVar,
				g.aotExternalIntrinsicCall(
					externalTarget.intrinsic,
					invokeNode,
					argExprs,
				),
			)
			g.writef("} else {\n")
			fallback := g.allocateTempVar()
			g.writef("%s := checkDerefVar(%s)\n", fallback, varID)
			g.generateApply(resultVar, fallback, argExprs, false)
			g.writef("}\n")
			return resultVar
		}
		g.writef("%s := %s(%s)\n",
			resultVar,
			externalTarget.fnVar,
			strings.Join(argExprs, ", "),
		)
		return resultVar
	}

	// Emit the invocation using fixed-arity Apply for 0-4 args to avoid []any alloc.
	g.generateApply(resultVar, fnExpr, argExprs, true)
	return resultVar
}

func staticInstanceType(invoke *ast.InvokeNode) (reflect.Type, bool) {
	if len(invoke.Args) != 2 ||
		invoke.Fn.Op != ast.OpVar ||
		invoke.Args[0].Op != ast.OpConst {
		return nil, false
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr.Namespace().Name().String() != "clojure.core" ||
		vr.Symbol().String() != "instance?" {
		return nil, false
	}
	typ, ok := invoke.Args[0].Sub.(*ast.ConstNode).Value.(reflect.Type)
	return typ, ok && typ != nil
}

func (g *Generator) staticInstanceCall(
	invoke *ast.InvokeNode,
	args []string,
) (string, bool) {
	typ, ok := staticInstanceType(invoke)
	if !ok || len(args) != 2 {
		return "", false
	}
	typeExpr, ok := g.goTypeExpr(typ)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("lang.IsInstance[%s](%s)", typeExpr, args[1]), true
}

func (g *Generator) generateApply(
	resultVar string,
	fnExpr string,
	argExprs []string,
	declare bool,
) {
	operator := "="
	if declare {
		operator = ":="
	}
	arity := len(argExprs)
	if arity == 0 {
		g.writef("%s %s lang.Apply0(%s)\n", resultVar, operator, fnExpr)
		return
	}
	if arity <= 20 {
		g.writef("%s %s lang.Apply%d(%s, %s)\n",
			resultVar, operator, arity, fnExpr, strings.Join(argExprs, ", "))
		return
	}
	g.writef("%s %s lang.Apply(%s, []any{%s})\n",
		resultVar, operator, fnExpr, strings.Join(argExprs, ", "))
}

// generateDo generates code for a Do node
func (g *Generator) generateDo(node *ast.Node) string {
	doNode := node.Sub.(*ast.DoNode)

	// Emit all statements except the last to g.w
	for _, stmt := range doNode.Statements {
		if stmt == nil {
			continue
		}
		stmtResult := g.generateASTNode(stmt)
		g.writeAssign("_", stmtResult) // Prevent unused variable warning
	}

	// Return the final expression
	return g.generateASTNode(doNode.Ret)
}

// generateIf generates code for an If node
func (g *Generator) generateIf(node *ast.Node) string {
	ifNode := node.Sub.(*ast.IfNode)

	// Allocate result variable
	resultVar := g.allocateTempVar()

	// Emit the if statement to g.w
	g.writef("var %s any\n", resultVar)
	testExpr := g.generateTruthyTest(ifNode.Test)
	g.writef("if %s {\n", testExpr)
	thenExpr := g.generateASTNode(ifNode.Then)
	g.writeAssign(resultVar, thenExpr)
	g.writef("} else {\n")
	if ifNode.Else != nil {
		elsExpr := g.generateASTNode(ifNode.Else)
		g.writeAssign(resultVar, elsExpr)
	} else {
		g.writef("  %s = nil\n", resultVar)
	}
	g.writef("}\n")

	// Return the r-value
	return resultVar
}

func (g *Generator) generateTruthyTest(node *ast.Node) string {
	if node.Op != ast.OpInvoke {
		return fmt.Sprintf("lang.IsTruthy(%s)", g.generateASTNode(node))
	}
	invoke := node.Sub.(*ast.InvokeNode)
	target := g.aotExternalInvokeTarget(invoke)
	if target == nil || target.intrinsic != "seq" {
		return fmt.Sprintf("lang.IsTruthy(%s)", g.generateASTNode(node))
	}

	arg := g.generateASTNode(invoke.Args[0])
	if target.directLinked {
		result := g.allocateTempVar()
		g.writef("%s := lang.IsSeqTruthy(%s)\n", result, arg)
		return result
	}
	varNode := invoke.Fn.Sub.(*ast.VarNode)
	varID := g.allocVarVar(
		varNode.Var.Namespace().Name().String(),
		varNode.Var.Symbol().String(),
	)
	result := g.allocateTempVar()
	g.writef("var %s bool\n", result)
	g.writef("if %s && %s.RootVersion() == %s {\n",
		target.defaultVar,
		varID,
		target.rootVersionVar,
	)
	g.writef("%s = lang.IsSeqTruthy(%s)\n", result, arg)
	g.writef("} else {\n")
	fallback := g.allocateTempVar()
	g.writef("%s := checkDerefVar(%s)\n", fallback, varID)
	fallbackResult := g.allocateTempVar()
	g.writef("var %s any\n", fallbackResult)
	g.generateApply(fallbackResult, fallback, []string{arg}, false)
	g.writef("%s = lang.IsTruthy(%s)\n", result, fallbackResult)
	g.writef("}\n")
	return result
}

func (g *Generator) generateCase(node *ast.Node) string {
	caseNode := node.Sub.(*ast.CaseNode)

	testExpr := g.generateASTNode(caseNode.Test)
	resultVar := g.allocateTempVar()

	g.writef("// case\n")
	g.writef("var %s any\n", resultVar)
	// implement as if-else chain; evaluation of case clauses is order-dependent
	// case tests are evaluated lazily, so we need to generate them in the if conditions
	// moreover, the text expressions may produce multiple statements, so we need to generate them inline
	// therefore we can't use a switch statement or || operator
	// instead we generate a series of if-else statements
	// each test expression is compared to the testExpr using lang.Equals
	// if a test matches, we evaluate the corresponding body and assign to resultVar
	// Generate code based on test type
	testType := caseNode.TestType.(lang.Keyword)

	// Calculate the lookup key based on test type
	lookupVar := g.allocateTempVar()
	g.writef("var %s int64\n", lookupVar)

	switch testType {
	case lang.KWInt:
		// For integers, convert directly to int64 and apply shift/mask
		g.writef("switch v := %s.(type) {\n", testExpr)
		g.writef("case int64: %s = v\n", lookupVar)
		g.writef("case int: %s = int64(v)\n", lookupVar)
		g.writef("case int32: %s = int64(v)\n", lookupVar)
		g.writef("case int16: %s = int64(v)\n", lookupVar)
		g.writef("case int8: %s = int64(v)\n", lookupVar)
		g.writef("default: %s = -1 // won't match any case\n", lookupVar)
		g.writef("}\n")
		// Apply shift and mask if needed
		if caseNode.Mask != 0 {
			g.writef("%s = int64(uint32(%s >> %d) & uint32(%d))\n",
				lookupVar, lookupVar, caseNode.Shift, caseNode.Mask)
		}

	case lang.KWHashIdentity:
		// Use identity hash
		if caseNode.Mask == 0 {
			g.writef("%s = int64(lang.IdentityHash(%s))\n", lookupVar, testExpr)
		} else {
			g.writef("%s = int64(uint32(lang.IdentityHash(%s) >> %d) & uint32(%d))\n",
				lookupVar, testExpr, caseNode.Shift, caseNode.Mask)
		}

	case lang.KWHashEquiv:
		// Use hash
		if caseNode.Mask == 0 {
			g.writef("%s = int64(lang.Hash(%s))\n", lookupVar, testExpr)
		} else {
			g.writef("%s = int64(uint32(lang.Hash(%s) >> %d) & uint32(%d))\n",
				lookupVar, testExpr, caseNode.Shift, caseNode.Mask)
		}
	}

	// Generate switch statement for the entries
	first := true
	for i, entry := range caseNode.Entries {
		g.writef("// case entry %d (key=%d, collision=%v)\n", i, entry.Key, entry.HasCollision)

		if first {
			g.writef("if %s == %d {\n", lookupVar, entry.Key)
			first = false
		} else {
			g.writef("} else if %s == %d {\n", lookupVar, entry.Key)
		}

		if entry.HasCollision {
			// For collision cases, evaluate the condp expression
			condpExpr := g.generateASTNode(entry.ResultExpr)
			g.writeAssign(resultVar, condpExpr)
		} else if testType == lang.KWInt {
			// For integers with shift/mask, we need to verify the actual value
			// because multiple values can map to the same key
			if caseNode.Mask != 0 {
				// Need to check actual value matches
				expectedExpr := g.generateASTNode(entry.TestConstant)
				g.writef("if lang.Equals(%s, %s) {\n", testExpr, expectedExpr)
				resultExpr := g.generateASTNode(entry.ResultExpr)
				g.writeAssign(resultVar, resultExpr)
				g.writef("} else {\n")
				// Fall through to default
				if caseNode.Default != nil {
					defaultExpr := g.generateASTNode(caseNode.Default)
					g.writeAssign(resultVar, defaultExpr)
				} else {
					g.writef("panic(lang.NewIllegalArgumentError(fmt.Sprintf(\"No matching clause: %%v\", %s)))\n", testExpr)
				}
				g.writef("}\n")
			} else {
				// For integers without shift/mask, the key match is sufficient
				resultExpr := g.generateASTNode(entry.ResultExpr)
				g.writeAssign(resultVar, resultExpr)
			}
		} else {
			// For hash-based dispatch, verify the actual value matches
			expectedExpr := g.generateASTNode(entry.TestConstant)
			g.writef("if ")
			if testType == lang.KWHashIdentity {
				g.writef("%s == %s", testExpr, expectedExpr)
			} else {
				g.writef("lang.Equals(%s, %s)", testExpr, expectedExpr)
			}
			g.writef(" {\n")
			resultExpr := g.generateASTNode(entry.ResultExpr)
			g.writeAssign(resultVar, resultExpr)
			g.writef("} else {\n")
			// Fall through to default
			if caseNode.Default != nil {
				defaultExpr := g.generateASTNode(caseNode.Default)
				g.writeAssign(resultVar, defaultExpr)
			} else {
				g.writef("panic(lang.NewIllegalArgumentError(fmt.Sprintf(\"No matching clause: %%v\", %s)))\n", testExpr)
			}
			g.writef("}\n")
		}
	}
	if caseNode.Default != nil {
		g.writef("} else {\n")
		defaultExpr := g.generateASTNode(caseNode.Default)
		g.writeAssign(resultVar, defaultExpr)
		g.writef("}\n")
	} else {
		g.writef("} else {\n")
		g.writef("  panic(fmt.Sprintf(\"no matching case clause: %%v\", %s))\n", testExpr)
		g.writef("}\n")
	}

	return resultVar
}

// generateLet generates code for a Let node
func (g *Generator) generateLet(node *ast.Node, isLoop bool) string {
	letNode := node.Sub.(*ast.LetNode)

	// Push a new variable scope for the let bindings
	resultId := g.allocateTempVar()
	g.writef("var %s any\n", resultId)

	g.writef("{ // let\n")
	g.pushVarScope()
	defer func() {
		g.popVarScope()
		g.writef("} // end let\n")
	}()

	// Collect binding variable names for recur context if this is a loop
	var bindingVars []string
	if isLoop {
		bindingVars = make([]string, 0, len(letNode.Bindings))
	}

	// Emit bindings directly to g.w
	for bindingIndex, binding := range letNode.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		name := bindingNode.Name.Name()
		init := bindingNode.Init

		// Allocate a Go variable for the Clojure name
		g.writef("// let binding \"%s\"\n", name)

		// Generate initialization code
		var localAtomInit *ast.Node
		if !isLoop {
			localAtomInit = scalarReplaceableAtomInit(
				bindingNode,
				letNode.Bindings[bindingIndex+1:],
				letNode.Body,
			)
		}
		initCode := ""
		if localAtomInit != nil {
			initCode = g.generateASTNode(localAtomInit)
		} else {
			initCode = g.generateASTNode(init)
		}
		varName := g.allocateLocal(name)
		g.writef("var %s any = %s\n", varName, initCode)
		if localAtomInit != nil {
			g.markLocalAtom(name)
		}
		g.writeAssign("_", varName) // Prevent unused variable warning

		// Collect binding variables for loop
		if isLoop {
			bindingVars = append(bindingVars, varName)
		}
	}

	if isLoop {
		// Push recur context for this loop
		g.pushRecurContext(letNode.LoopID, bindingVars, false)
		defer g.popRecurContext()

		g.writef("for {\n")
	}

	// Return the body expression (r-value)
	result := g.generateASTNode(letNode.Body)
	if isLoop {
		g.writeAssign(resultId, result)
		g.writef("  break\n") // Break out of the loop after the body
		g.writef("}\n")
	} else {
		g.writeAssign(resultId, result)
	}
	return resultId
}

func (g *Generator) generateLetFn(node *ast.Node) string {
	letFnNode := node.Sub.(*ast.LetFnNode)

	resultId := g.allocateTempVar()
	g.writef("var %s any\n", resultId)

	// Push a new variable scope for the letfn bindings
	g.writef("{ // letfn\n")
	g.pushVarScope()
	defer func() {
		g.popVarScope()
		g.writef("} // end letfn\n")
	}()

	// Emit bindings directly to g.w
	for _, binding := range letFnNode.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		name := bindingNode.Name
		fn := bindingNode.Init

		// Allocate a Go variable for the Clojure name.
		// Use any so that FnFuncN assignments (from fixed-arity functions)
		// are accepted without type mismatch; ApplyN's type switch handles dispatch.
		g.writef("// letfn binding \"%s\"\n", name)
		varName := g.allocateLocal(name.Name())
		// declare the variable now to allow for recursion
		g.writef("var %s any\n", varName)
		fnVar := g.generateASTNode(fn)
		g.writeAssign(varName, fnVar)
		g.writeAssign("_", varName) // Prevent unused variable warning
	}

	// Return the body expression (r-value)
	result := g.generateASTNode(letFnNode.Body)
	g.writeAssign(resultId, result)
	return resultId
}

func (g *Generator) generateRecur(node *ast.Node) string {
	recurNode := node.Sub.(*ast.RecurNode)

	// Find the matching recur context
	var ctx *recurContext
	for i := len(g.recurStack) - 1; i >= 0; i-- {
		if lang.Equals(g.recurStack[i].loopID, recurNode.LoopID) {
			ctx = &g.recurStack[i]
			break
		}
	}

	if ctx == nil {
		panic(fmt.Sprintf("recur without matching loop for ID: %v", recurNode.LoopID))
	}

	// Verify the number of recur expressions matches the number of loop bindings
	if len(recurNode.Exprs) != len(ctx.bindings) {
		panic(fmt.Sprintf("recur expects %d arguments, got %d", len(ctx.bindings), len(recurNode.Exprs)))
	}

	// Generate temporary variables to hold the new values
	// This prevents issues with bindings that reference each other
	tempVars := make([]string, len(recurNode.Exprs))
	for i, expr := range recurNode.Exprs {
		tempVar := g.allocateTempVar()
		tempVars[i] = tempVar
		exprCode := g.generateASTNode(expr)
		g.writef("var %s any = %s\n", tempVar, exprCode)
	}

	// Assign the temporary values to the loop bindings
	for i, bindingVar := range ctx.bindings {
		g.writef("%s = %s\n", bindingVar, tempVars[i])
	}

	if ctx.useGoto {
		// Use a goto statement to jump back to the loop label
		g.writef("goto recur_%s\n", ctx.loopID.Name())
	} else {
		// Continue the loop
		g.writef("continue\n")
	}

	// Return empty string since recur doesn't produce a value
	// (control flow never reaches past the continue)
	return ""
}

// generateThrow generates code for a throw node
func (g *Generator) generateThrow(node *ast.Node) string {
	throwNode := node.Sub.(*ast.ThrowNode)

	// Generate the exception expression
	exceptionExpr := g.generateASTNode(throwNode.Exception)

	// Panic with the exception
	g.writef("panic(%s)\n", exceptionExpr)

	// Return empty string to signal no value is produced
	// The calling function should not generate a return after this
	return ""
}

// generateTry generates code for a try node
func (g *Generator) generateTry(node *ast.Node) string {
	tryNode := node.Sub.(*ast.TryNode)

	// Allocate result variable
	resultVar := g.allocateTempVar()
	g.writef("var %s any\n", resultVar)

	// Use a closure to handle the try logic
	g.writef("func() {\n")

	// Generate finally block if present
	if tryNode.Finally != nil {
		g.writef("defer func() {\n")
		// Finally doesn't affect the return value
		result := g.generateASTNode(tryNode.Finally)
		g.writeAssign("_", result) // Prevent unused variable warning
		g.writef("}()\n")
	}

	// Generate catch blocks if present
	if len(tryNode.Catches) > 0 {
		g.writef("defer func() {\n")
		g.writef("if r := recover(); r != nil {\n")

		for i, catchNode := range tryNode.Catches {
			catch := catchNode.Sub.(*ast.CatchNode)

			// Generate the class/type check
			// For now, we'll handle simple cases
			// TODO: implement proper type matching
			classExpr := g.generateASTNode(catch.Class)

			// For each catch, check if the exception matches
			if i > 0 {
				g.writef("} else ")
			}

			// Check if the exception matches this catch type
			g.writef("if lang.CatchMatches(r, %s) {\n", classExpr)

			// Create new scope for catch binding
			g.pushVarScope()

			// Bind the exception to the catch variable
			bindingNode := catch.Local.Sub.(*ast.BindingNode)
			catchVar := g.allocateLocal(bindingNode.Name.Name())
			g.writef("%s := r\n", catchVar)
			g.writeAssign("_", catchVar) // Mark as used since catch body might not reference it

			// Generate the catch body
			bodyResult := g.generateASTNode(catch.Body)
			g.writeAssign(resultVar, bodyResult)

			g.popVarScope()
		}

		// Re-panic if no catch matched
		g.writef("} else {\n")
		g.writef("panic(r)\n")
		g.writef("}\n")

		g.writef("}\n")
		g.writef("}()\n")
	}

	// Generate the try body
	bodyResult := g.generateASTNode(tryNode.Body)
	g.writeAssign(resultVar, bodyResult)

	g.writef("}()\n")

	return resultVar
}

func (g *Generator) generateGoBuiltin(node *ast.Node) string {
	goBuiltinNode := node.Sub.(*ast.GoBuiltinNode)
	sym := goBuiltinNode.Sym

	_, ok := lang.Builtins[sym.Name()]
	if !ok {
		panic(fmt.Sprintf("unknown Go builtin: %s", sym.Name()))
	}

	return "lang.Builtins[\"" + sym.Name() + "\"]"
}

// generateWithMeta generates code for a WithMeta node
func (g *Generator) generateWithMeta(node *ast.Node) string {
	wmNode := node.Sub.(*ast.WithMetaNode)

	expr := wmNode.Expr
	meta := wmNode.Meta

	exprVal := g.generateASTNode(expr)
	metaVal := g.generateASTNode(meta)

	resultId := g.allocateTempVar()
	g.writef("%s, err := lang.WithMeta(%s, %s.(lang.IPersistentMap))\n", resultId, exprVal, metaVal)
	g.writef("if err != nil {\n")
	g.writef("  panic(err)\n")
	g.writef("}\n")

	return resultId
}

func (g *Generator) generateVector(node *ast.Node) string {
	vectorNode := node.Sub.(*ast.VectorNode)

	itemIds := make([]string, len(vectorNode.Items))
	for i, item := range vectorNode.Items {
		itemId := g.generateASTNode(item)
		itemIds[i] = itemId
	}
	vecId := g.allocateTempVar()
	g.writef("%s := lang.NewVector(%s)\n", vecId, strings.Join(itemIds, ", "))

	return vecId
}

func (g *Generator) generateMap(node *ast.Node) string {
	mapNode := node.Sub.(*ast.MapNode)

	keyValueCount := len(mapNode.Keys) * 2
	if keyValueCount > lang.PersistentArrayMapInlineKeyValueCount &&
		keyValueCount <= lang.PersistentArrayMapMaxKeywordKeyValueCount {
		if names, ok := staticKeywordMapNames(mapNode.Keys); ok {
			valueIDs := make([]string, len(mapNode.Vals))
			for i, value := range mapNode.Vals {
				valueIDs[i] = g.generateASTNode(value)
			}
			constructor := g.allocKeywordMapConstructor(names)
			mapID := g.allocateTempVar()
			g.writef("%s := %s(%s)\n",
				mapID, constructor, strings.Join(valueIDs, ", "))
			return mapID
		}
	}

	keyValIds := make([]string, 2*len(mapNode.Keys))
	for i, key := range mapNode.Keys {
		keyId := g.generateASTNode(key)

		valNode := mapNode.Vals[i]
		valId := g.generateASTNode(valNode)

		keyValIds[2*i] = keyId   // key
		keyValIds[2*i+1] = valId // value
	}
	mapId := g.allocateTempVar()
	constructor := "lang.NewMap"
	if len(keyValIds) > lang.PersistentArrayMapInlineKeyValueCount {
		constructor = "lang.NewMapUniqueKeys"
	}
	g.writef("%s := %s(%s)\n", mapId, constructor, strings.Join(keyValIds, ", "))

	return mapId
}

func staticKeywordMapNames(keys []*ast.Node) ([]string, bool) {
	names := make([]string, len(keys))
	for i, key := range keys {
		if key.Op != ast.OpConst {
			return nil, false
		}
		keyword, ok := key.Sub.(*ast.ConstNode).Value.(lang.Keyword)
		if !ok {
			return nil, false
		}
		if ns := keyword.Namespace(); ns != nil {
			names[i] = fmt.Sprintf("%s/%s", ns, keyword.Name())
		} else {
			names[i] = keyword.Name()
		}
	}
	return names, true
}

func keysFromAssoc(assoc *ast.AssocNode) []*ast.Node {
	keys := make([]*ast.Node, len(assoc.Entries))
	for i := range assoc.Entries {
		keys[i] = assoc.Entries[i].Key
	}
	return keys
}

func keywordName(keyword lang.Keyword) string {
	if ns := keyword.Namespace(); ns != nil {
		return fmt.Sprintf("%s/%s", ns, keyword.Name())
	}
	return keyword.Name()
}

func (g *Generator) allocKeywordMapConstructor(names []string) string {
	key := strings.Join(names, "\x00")
	if constructor, ok := g.keywordMapConstructors[key]; ok {
		return constructor
	}
	index := len(g.keywordMapConstructors)
	shape := fmt.Sprintf("aotKeywordMapShape%d", index)
	constructor := fmt.Sprintf("aotKeywordMapNew%d", index)
	storage := fmt.Sprintf("aotKeywordMapStorage%d", index)
	quoted := make([]string, len(names))
	params := make([]string, len(names))
	values := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
		params[i] = fmt.Sprintf("v%d any", i)
		values[i] = fmt.Sprintf("v%d", i)
	}
	fmt.Fprintf(&g.aotDeclarations,
		`var %s = lang.NewKeywordMapShape(%s)
type %s struct {
	lang.Map
	values [%d]any
}
func %s(%s) *lang.Map {
	storage := &%s{}
	storage.values = [%d]any{%s}
	return lang.InitStaticKeywordMap(
		&storage.Map,
		%s,
		storage.values[:],
	)
}
`,
		shape,
		strings.Join(quoted, ", "),
		storage,
		len(names),
		constructor,
		strings.Join(params, ", "),
		storage,
		len(names),
		strings.Join(values, ", "),
		shape,
	)
	g.keywordMapConstructors[key] = constructor
	return constructor
}

func (g *Generator) generateSet(node *ast.Node) string {
	setNode := node.Sub.(*ast.SetNode)

	itemIds := make([]string, len(setNode.Items))
	for i, item := range setNode.Items {
		itemId := g.generateASTNode(item)
		itemIds[i] = itemId
	}
	setId := g.allocateTempVar()
	g.writef("%s := lang.NewSet(%s)\n", setId, strings.Join(itemIds, ", "))
	return setId
}

var (
	// TODO: fix all these invalid imports
	expectedInvalidImports = map[string]bool{
		"ExceptionInfo":                            true,
		"LinkedBlockingQueue":                      true,
		"clojure.lang.LineNumberingPushbackReader": true,
		"clojure.lang":                             true,
		"java.io.InputStreamReader":                true,
		"java.io.StringReader":                     true,
		"java.util.concurrent.CountDownLatch":      true,
		"java.util.concurrent":                     true,
		"java.lang":                                true,
	}
)

func (g *Generator) generateMaybeClass(node *ast.Node) string {
	sym := node.Sub.(*ast.MaybeClassNode).Class.(*lang.Symbol)
	pkg := sym.FullName()

	v, ok := pkgmap.Get(sym.FullName())
	// special-case for reflect.Types
	//
	// NB: we're allowing references to exports of packages that aren't in the package map
	// This implies a difference in behavior, where the interpreter would fail while
	// the compiled code would succeed, because the import will cause the go toolchain
	// to pull in the package.
	if ok {
		if c, ok := v.(*lang.Class); ok {
			return g.generateClassValue(c)
		}
		if t, ok := v.(reflect.Type); ok {
			return g.generateTypeValue(t)
		}
	}

	return g.generateGoExportedName(pkg)
}

func (g *Generator) generateGoExportedName(pkg string) string {
	// find last dot in the package name
	dotIndex := strings.LastIndex(pkg, ".")
	if dotIndex == -1 {
		// TODO: panic
		// For now, return a nil value to avoid panic
		fmt.Println("Warning: invalid package reference:", pkg)
		return "nil"
		//panic(fmt.Sprintf("invalid package reference: %s", pkg))
	}
	mungedPkgName := pkg[:dotIndex]
	exportedName := pkg[dotIndex+1:]

	packageName := pkgmap.UnmungePkg(mungedPkgName)

	if _, ok := expectedInvalidImports[packageName]; ok {
		// TODO: fix all these invalid imports
		fmt.Println("Warning: skipping invalid import:", packageName)
		return "nil"
	}
	alias := g.addImportWithAlias(packageName)

	if strings.HasPrefix(exportedName, "*") {
		// pointers look like package.*Type
		exportedName = exportedName[1:]
		alias = "*" + alias
	}
	return alias + "." + exportedName
}

func (g *Generator) generateHostCall(node *ast.Node) string {
	hostCallNode := node.Sub.(*ast.HostCallNode)

	tgt := hostCallNode.Target
	method := hostCallNode.Method
	args := hostCallNode.Args

	tgtId := g.generateASTNode(tgt)

	argIds := make([]string, len(args))
	for i, arg := range args {
		argIds[i] = g.generateASTNode(arg)
	}

	methodName := method.Name()
	if directMethod, directArgs, ok := directHostCall(tgt, methodName, argIds); ok {
		resultId := g.allocateTempVar()
		g.writef("%s := %s.%s(%s)\n", resultId, tgtId, directMethod, strings.Join(directArgs, ", "))
		return resultId
	}
	if directMethod, receiver, directArgs, ok := g.directInferredHostCall(
		tgt,
		tgtId,
		methodName,
		argIds,
	); ok {
		resultId := g.allocateTempVar()
		g.writef(
			"%s := %s.%s(%s)\n",
			resultId,
			receiver,
			directMethod,
			strings.Join(directArgs, ", "),
		)
		return resultId
	}

	methodId := g.allocateTempVar()
	g.writef("%s, _ := lang.FieldOrMethod(%s, %q)\n", methodId, tgtId, methodName)
	g.writef("if reflect.TypeOf(%s).Kind() != reflect.Func {\n", methodId)
	g.writef("  panic(lang.NewIllegalArgumentError(fmt.Sprintf(\"%s is not a function\")))\n", methodName)
	g.writef("}\n")

	resultId := g.allocateTempVar()
	n := len(argIds)
	switch n {
	case 0:
		g.writef("%s := lang.Apply0(%s)\n", resultId, methodId)
	case 1:
		g.writef("%s := lang.Apply1(%s, %s)\n", resultId, methodId, argIds[0])
	case 2:
		g.writef("%s := lang.Apply2(%s, %s)\n", resultId, methodId, strings.Join(argIds, ", "))
	case 3:
		g.writef("%s := lang.Apply3(%s, %s)\n", resultId, methodId, strings.Join(argIds, ", "))
	case 4:
		g.writef("%s := lang.Apply4(%s, %s)\n", resultId, methodId, strings.Join(argIds, ", "))
	default:
		g.writef("%s := lang.Apply(%s, []any{%s})\n", resultId, methodId, strings.Join(argIds, ", "))
	}

	return resultId
}

func directHostMethod(target *ast.Node, name string, arity int) (string, bool) {
	args := make([]string, arity)
	for i := range args {
		args[i] = fmt.Sprintf("arg%d", i)
	}
	method, converted, ok := directHostCall(target, name, args)
	if !ok {
		return "", false
	}
	for i := range args {
		if converted[i] != args[i] {
			return "", false
		}
	}
	return method, true
}

func directHostCall(
	target *ast.Node,
	name string,
	args []string,
) (string, []string, bool) {
	if target.Op != ast.OpConst || name == "" {
		return "", nil, false
	}
	value := target.Sub.(*ast.ConstNode).Value
	typ := reflect.TypeOf(value)
	if typ == nil {
		return "", nil, false
	}
	method, receiverOffset, ok := directHostMethodForType(typ, name)
	if !ok {
		return "", nil, false
	}
	converted, ok := convertDirectHostArgs(
		method,
		receiverOffset,
		args,
		convertDirectHostArg,
	)
	if !ok {
		return "", nil, false
	}
	return method.Name, converted, true
}

func convertDirectHostArg(paramType reflect.Type, arg string) (string, bool) {
	switch paramType {
	case reflect.TypeFor[any]():
		return arg, true
	case reflect.TypeFor[int]():
		return "lang.IntCast(" + arg + ")", true
	default:
		// Keep the reflective path when codegen cannot preserve Glojure's
		// host-argument coercion semantics.
		return "", false
	}
}

func directHostMethodForType(
	typ reflect.Type,
	name string,
) (reflect.Method, int, bool) {
	if typ == nil || name == "" {
		return reflect.Method{}, 0, false
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		name = string(name[0]-'a'+'A') + name[1:]
	}
	method, ok := typ.MethodByName(name)
	receiverOffset := 1
	if typ.Kind() == reflect.Interface {
		receiverOffset = 0
	}
	if !ok && typ.Kind() != reflect.Pointer {
		typ = reflect.PointerTo(typ)
		method, ok = typ.MethodByName(name)
		receiverOffset = 1
	}
	if !ok || method.Type.NumOut() != 1 {
		return reflect.Method{}, 0, false
	}
	return method, receiverOffset, true
}

func convertDirectHostArgs(
	method reflect.Method,
	receiverOffset int,
	args []string,
	convert func(reflect.Type, string) (string, bool),
) ([]string, bool) {
	fixedArgCount := method.Type.NumIn() - receiverOffset
	if method.Type.IsVariadic() {
		fixedArgCount--
		if len(args) < fixedArgCount {
			return nil, false
		}
	} else if len(args) != fixedArgCount {
		return nil, false
	}

	converted := make([]string, len(args))
	for i := range args {
		var paramType reflect.Type
		if i < fixedArgCount {
			paramType = method.Type.In(i + receiverOffset)
		} else {
			paramType = method.Type.In(method.Type.NumIn() - 1).Elem()
		}
		var ok bool
		converted[i], ok = convert(paramType, args[i])
		if !ok {
			return nil, false
		}
	}
	return converted, true
}

func (g *Generator) directInferredHostCall(
	target *ast.Node,
	targetID string,
	name string,
	args []string,
) (methodName, receiver string, converted []string, ok bool) {
	typ, ok := inferredHostType(target)
	if !ok {
		return "", "", nil, false
	}
	method, receiverOffset, ok := directHostMethodForType(typ, name)
	if !ok {
		return "", "", nil, false
	}
	converted, ok = convertDirectHostArgs(
		method,
		receiverOffset,
		args,
		g.convertInferredDirectHostArg,
	)
	if !ok {
		return "", "", nil, false
	}
	if target.Op == ast.OpConst {
		return method.Name, targetID, converted, true
	}
	interfaceExpr, ok := g.hostMethodInterfaceExpr(method, receiverOffset)
	if !ok {
		return "", "", nil, false
	}
	return method.Name,
		fmt.Sprintf("%s.(%s)", targetID, interfaceExpr),
		converted,
		true
}

func (g *Generator) convertInferredDirectHostArg(
	paramType reflect.Type,
	arg string,
) (string, bool) {
	if converted, ok := convertDirectHostArg(paramType, arg); ok {
		return converted, true
	}
	if paramType.Kind() != reflect.Interface {
		return "", false
	}
	typeExpr, ok := g.goTypeExpr(paramType)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("lang.MustHostCast[%s](%s)", typeExpr, arg), true
}

func inferredHostType(target *ast.Node) (reflect.Type, bool) {
	if target != nil && target.Op == ast.OpConst {
		typ := reflect.TypeOf(target.Sub.(*ast.ConstNode).Value)
		if typ != nil {
			return typ, true
		}
	}
	var tag *lang.Symbol
	if target != nil {
		if withMeta, ok := target.Form.(lang.IMeta); ok {
			tag, _ = lang.Get(withMeta.Meta(), lang.KWTag).(*lang.Symbol)
		}
		if tag == nil && target.Op == ast.OpLocal {
			tag, _ = lang.Get(
				target.Sub.(*ast.LocalNode).Name.Meta(),
				lang.KWTag,
			).(*lang.Symbol)
		}
	}
	if tag == nil {
		return nil, false
	}
	tagName := tag.FullName()
	value, ok := pkgmap.Get(tagName)
	if !ok && strings.HasPrefix(tagName, "clojure.lang.") {
		value, ok = pkgmap.Get(
			"github.com/glojurelang/glojure/pkg/lang." +
				strings.TrimPrefix(tagName, "clojure.lang."),
		)
	}
	if !ok {
		return nil, false
	}
	switch value := value.(type) {
	case reflect.Type:
		return value, true
	case *lang.Class:
		return value.Type, value.Type != nil
	default:
		return nil, false
	}
}

func (g *Generator) hostMethodInterfaceExpr(
	method reflect.Method,
	receiverOffset int,
) (string, bool) {
	params := make([]string, 0, method.Type.NumIn()-receiverOffset)
	for i := receiverOffset; i < method.Type.NumIn(); i++ {
		paramType := method.Type.In(i)
		variadic := method.Type.IsVariadic() && i == method.Type.NumIn()-1
		if variadic {
			paramType = paramType.Elem()
		}
		typeExpr, ok := g.goTypeExpr(paramType)
		if !ok {
			return "", false
		}
		if variadic {
			typeExpr = "..." + typeExpr
		}
		params = append(params, typeExpr)
	}
	result, ok := g.goTypeExpr(method.Type.Out(0))
	if !ok {
		return "", false
	}
	return fmt.Sprintf(
		"interface { %s(%s) %s }",
		method.Name,
		strings.Join(params, ", "),
		result,
	), true
}

func (g *Generator) goTypeExpr(typ reflect.Type) (string, bool) {
	if typ == reflect.TypeFor[any]() {
		return "any", true
	}
	if typ.Name() != "" {
		if typ.PkgPath() == "" {
			return typ.Name(), true
		}
		alias := g.addImportWithAlias(typ.PkgPath())
		return alias + "." + typ.Name(), true
	}
	switch typ.Kind() {
	case reflect.Pointer:
		elem, ok := g.goTypeExpr(typ.Elem())
		return "*" + elem, ok
	case reflect.Slice:
		elem, ok := g.goTypeExpr(typ.Elem())
		return "[]" + elem, ok
	case reflect.Map:
		key, keyOK := g.goTypeExpr(typ.Key())
		value, valueOK := g.goTypeExpr(typ.Elem())
		return "map[" + key + "]" + value, keyOK && valueOK
	case reflect.Interface:
		if typ.NumMethod() == 0 {
			return "any", true
		}
	}
	return "", false
}

func (g *Generator) generateHostInterop(node *ast.Node) string {
	hostInteropNode := node.Sub.(*ast.HostInteropNode)

	tgtId := g.generateASTNode(hostInteropNode.Target)

	mOrF := hostInteropNode.MOrF.Name()
	if directMethod, receiver, _, ok := g.directInferredHostCall(
		hostInteropNode.Target,
		tgtId,
		mOrF,
		nil,
	); ok {
		resultId := g.allocateTempVar()
		g.writef("%s := %s.%s()\n", resultId, receiver, directMethod)
		return resultId
	}
	mOrFId := g.allocateTempVar()
	g.writef("%s, ok := lang.FieldOrMethod(%s, %q)\n", mOrFId, tgtId, mOrF)
	g.writef("if !ok {\n")
	g.writef("  panic(lang.NewIllegalArgumentError(fmt.Sprintf(\"no such field or method on %%T: %%s\", %s, %q)))\n", tgtId, mOrF)
	g.writef("}\n")

	resultId := g.allocateTempVar()
	g.writef("var %s any\n", resultId)
	g.writef("switch reflect.TypeOf(%s).Kind() {\n", mOrFId)
	g.writef("case reflect.Func:\n")
	g.writef("  %s = lang.Apply(%s, nil)\n", resultId, mOrFId)
	g.writef("default:\n")
	g.writef("  %s = %s\n", resultId, mOrFId)
	g.writef("}\n")

	return resultId
}

// generateMaybeHostForm preserves the evaluator's late-bound host lookup.
// Some portable Clojure forms use JVM-style class names that compatibility
// bridges register only at runtime (for example clojure.lang.MapEntry/create).
func (g *Generator) generateMaybeHostForm(node *ast.Node) string {
	maybeHostNode := node.Sub.(*ast.MaybeHostFormNode)
	export := maybeHostNode.Class + "." + maybeHostNode.Field.Name()
	resultID := g.allocateTempVar()
	alias := g.addImportWithAlias("github.com/glojurelang/glojure/pkg/pkgmap")
	g.writef("%s, ok := %s.Get(%q)\n", resultID, alias, export)
	g.writef("if !ok {\n")
	g.writef("  panic(lang.NewIllegalArgumentError(%q))\n", "unable to resolve host form: "+export)
	g.writef("}\n")
	return resultID
}

func (g *Generator) generateTheVar(node *ast.Node) string {
	theVarNode := node.Sub.(*ast.TheVarNode)
	varSym := theVarNode.Var
	ns := varSym.Namespace()
	name := varSym.Symbol()

	resultId := g.allocateTempVar()
	g.writef("%s := lang.InternVarName(%s, %s)\n", resultId, g.allocSymVar(ns.Name().Name()), g.allocSymVar(name.Name()))
	return resultId
}

// generateSetBang generates code for a set! operation
func (g *Generator) generateSetBang(node *ast.Node) string {
	setBangNode := node.Sub.(*ast.SetBangNode)

	// Generate the value expression
	valExpr := g.generateASTNode(setBangNode.Val)

	// Handle the target
	target := setBangNode.Target
	switch target.Op {
	case ast.OpVar:
		// Setting a Var
		varNode := target.Sub.(*ast.VarNode)
		varNamespace := varNode.Var.Namespace()
		varSymbol := varNode.Var.Symbol()

		// Look up the var variable
		varId := g.allocVarVar(varNamespace.Name().String(), varSymbol.String())

		// Call Set on the Var and return the value
		resultId := g.allocateTempVar()
		g.writef("%s := %s.Set(%s)\n", resultId, varId, valExpr)
		return resultId

	case ast.OpHostInterop:
		// Setting a host field
		interopNode := target.Sub.(*ast.HostInteropNode)
		tgt := interopNode.Target
		targetExpr := g.generateASTNode(tgt)
		field := interopNode.MOrF

		resultId := g.allocateTempVar()

		// Generate reflection-based field setting
		g.writef("// set! host field\n")
		g.writef("var %s any\n", resultId)
		g.writef("{\n")
		g.writef("  targetV := reflect.ValueOf(%s)\n", targetExpr)
		g.writef("  if targetV.Kind() == reflect.Ptr {\n")
		g.writef("    targetV = targetV.Elem()\n")
		g.writef("  }\n")
		g.writef("  fieldVal := targetV.FieldByName(%q)\n", field.Name())
		g.writef("  if !fieldVal.IsValid() {\n")
		g.writef("    panic(fmt.Errorf(\"no such field %s\"))\n", field.Name())
		g.writef("  }\n")
		g.writef("  if !fieldVal.CanSet() {\n")
		g.writef("    panic(fmt.Errorf(\"cannot set field %s\"))\n", field.Name())
		g.writef("  }\n")
		g.writef("  valV := reflect.ValueOf(%s)\n", valExpr)
		g.writef("  if !valV.IsValid() {\n")
		g.writef("    switch fieldVal.Kind() {\n")
		g.writef("    case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:\n")
		g.writef("      fieldVal.Set(reflect.Zero(fieldVal.Type()))\n")
		g.writef("    default:\n")
		g.writef("      panic(fmt.Errorf(\"cannot set field %s to nil\"))\n", field.Name())
		g.writef("    }\n")
		g.writef("  } else {\n")
		g.writef("    fieldVal.Set(valV)\n")
		g.writef("  }\n")
		g.writef("  %s = %s\n", resultId, valExpr)
		g.writef("}\n")
		return resultId

	default:
		//return fmt.Sprintf("%q", "unimplemented: set! target type")
		return `"unimplemented: set! target type"`
		//panic(fmt.Sprintf("unsupported set! target: %v", target.Op))
	}
}

func (g *Generator) generateNew(node *ast.Node) string {
	newNode := node.Sub.(*ast.NewNode)

	if newNode.Class.Op == ast.OpVar {
		vr := newNode.Class.Sub.(*ast.VarNode).Var
		if recordType, ok := codegenVarValue(vr).(*lang.RecordType); ok {
			record := g.allocAOTRecordType(recordType)
			args := make([]string, len(newNode.Args))
			for i, arg := range newNode.Args {
				args[i] = g.generateASTNode(arg)
			}
			resultID := g.allocateTempVar()
			g.writef("%s := %s(%s)\n",
				resultID,
				record.constructor,
				strings.Join(args, ", "),
			)
			return resultID
		}
	}

	// the interpreter is more lax; it allows for expressions that evaluate to a type
	// here we assume the class is a constant type. clojure's new form is similar
	switch sub := newNode.Class.Sub.(type) {
	case *ast.ConstNode:
		if recordType, ok := sub.Value.(*lang.RecordType); ok {
			record := g.allocAOTRecordType(recordType)
			args := make([]string, len(newNode.Args))
			for i, arg := range newNode.Args {
				args[i] = g.generateASTNode(arg)
			}
			resultID := g.allocateTempVar()
			g.writef("%s := %s(%s)\n",
				resultID,
				record.constructor,
				strings.Join(args, ", "),
			)
			return resultID
		}
		class, ok := sub.Value.(reflect.Type)
		if !ok {
			fmt.Printf("Warning: glojure codegen only supports new with constant class types. Got %T\n", sub.Value)
			return fmt.Sprintf("%q", "unimplemented: new with non-constant class type")
		}
		// generate a reflect.Type for the class
		classId := g.generateValue(class)
		resultId := g.allocateTempVar()
		g.writef("%s := reflect.New(%s).Interface()\n", resultId, classId)
		return resultId
	case *ast.MaybeClassNode:
		resultId := g.allocateTempVar()
		className := g.generateGoExportedName(sub.Class.(*lang.Symbol).FullName())
		if className == "nil" {
			fmt.Printf("Failed to resolve class for new, generating nil: %v\n", sub.Class)
			return "nil"
		}
		g.writef("%s := new(%s)\n", resultId, className)
		return resultId
	default:
		fmt.Printf("Warning: glojure codegen only supports new with constant class types. Got %T\n", newNode.Class.Sub)
		return fmt.Sprintf("%q", "unimplemented: new with non-constant class type")
	}
}

////////////////////////////////////////////////////////////////////////////////

func (g *Generator) addImport(pkg string) {
	parts := strings.Split(pkg, "/")
	alias := parts[len(parts)-1]
	g.imports[pkg] = alias
}

func (g *Generator) addImportWithAlias(pkg string) string {
	if pkg == "glojure.lang.LineNumberingPushbackReader" {
		panic("glojure.lang.LineNumberingPushbackReader is not a valid Go package")
	}

	// Check if the package is already imported
	if alias, ok := g.imports[pkg]; ok {
		return alias // Return existing alias
	}
	// Generate a new alias based on the last part of the package name.
	// Sanitize the segment so it is a valid Go identifier: characters
	// that are not letters, digits, or underscores become underscores,
	// and a leading digit is prefixed with an underscore.
	parts := strings.Split(pkg, "/")
	alias := fmt.Sprintf("%s%d", sanitizeGoIdent(parts[len(parts)-1]), len(g.imports))
	g.imports[pkg] = alias // Store the alias for this package

	return alias
}

func sanitizeGoIdent(s string) string {
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_':
			return r
		default:
			return '_'
		}
	}, s)
	if len(out) > 0 && out[0] >= '0' && out[0] <= '9' {
		out = "_" + out
	}
	return out
}

func (g *Generator) header(pkgName string) string {
	header := fmt.Sprintf(`// Code generated by glojure codegen. DO NOT EDIT.

package %s

import (
`, pkgName)

	// sort the imports by their package name for deterministic output
	keys := make([]string, 0, len(g.imports))
	for k := range g.imports {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return g.imports[keys[i]] < g.imports[keys[j]]
	})

	for _, pkg := range keys {
		alias := g.imports[pkg]
		header += fmt.Sprintf("  %s \"%s\"\n", alias, pkg)
	}

	header += ")\n"
	return header
}

func (g *Generator) writef(format string, args ...any) error {
	_, err := fmt.Fprintf(g.currentWriter, format, args...)
	return err
}

// writeAssign writes an assignment iff the r-value string is non-empty
func (g *Generator) writeAssign(varName, rValue string) {
	if rValue == "" || rValue == "nil" {
		return
	}
	g.writef("%s = %s\n", varName, rValue)
}

func (g *Generator) startNewValueInit(name string) *valueInit {
	valInit := &valueInit{
		name: name,
		deps: make(map[string]struct{}),
	}
	g.currentValueInit = valInit
	g.currentWriter = &valInit.buf
	g.valueInits = append(g.valueInits, valInit)
	return valInit
}

////////////////////////////////////////////////////////////////////////////////
// Variable scope management and other helpers

// PushVarScope creates a new variable scope
func (g *Generator) pushVarScope() {
	// Get the current scope's next number as the start for the new scope
	nextNum := 0
	if len(g.varScopes) > 0 {
		currentScope := &g.varScopes[len(g.varScopes)-1]
		nextNum = currentScope.nextNum
	}

	// Push new scope onto the stack
	g.varScopes = append(g.varScopes, varScope{
		nextNum:    nextNum,
		names:      make(map[string]string),
		localAtoms: make(map[string]bool),
	})
}

// PopVarScope removes the current variable scope
func (g *Generator) popVarScope() {
	if len(g.varScopes) <= 1 {
		panic("cannot pop the root variable scope")
	}
	g.varScopes = g.varScopes[:len(g.varScopes)-1]
}

// allocateLocal allocates a Go variable name for the given Clojure name in the current scope
// If the name already exists in the current scope, it returns the existing Go variable name
func (g *Generator) allocateLocal(name string) string {
	if len(g.varScopes) == 0 {
		panic("no variable scope available")
	}

	currentScope := &g.varScopes[len(g.varScopes)-1]

	// Allocate new variable name
	varName := fmt.Sprintf("v%d", currentScope.nextNum)
	currentScope.names[name] = varName
	currentScope.nextNum++

	return varName
}

// makeLiftedKey creates a composite key for deduplicating lifted values
func (g *Generator) makeLiftedKey(value any) liftedKey {
	// Handle primitive types that should be compared by value
	switch v := value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, complex64, complex128,
		bool, string, lang.Keyword, lang.Char:
		// Primitive types - use value-based comparison
		return liftedKey{
			isPointer: false,
			value:     value,
		}
	case *lang.Symbol:
		// Symbols are immutable singletons, use value comparison
		return liftedKey{
			isPointer: false,
			value:     v.FullName(), // Use string representation for key
		}
	default:
		// Reference types - use pointer-based comparison
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
			if rv.IsNil() {
				return liftedKey{isPointer: false, value: nil}
			}
			if rv.CanAddr() || rv.Kind() == reflect.Ptr {
				return liftedKey{
					isPointer: true,
					pointer:   rv.Pointer(),
				}
			}
		}
		// Fallback: use value comparison
		return liftedKey{
			isPointer: false,
			value:     value,
		}
	}
}

func (g *Generator) liftValue(value any) string {
	key := g.makeLiftedKey(value)
	if lifted, ok := g.liftedValues[key]; ok {
		return lifted.varName
	}

	varName := fmt.Sprintf("closed%d", g.liftedCounter)
	g.liftedCounter++
	g.liftedValues[key] = &liftedValue{
		value:   value,
		varName: varName,
	}
	if g.currentValueInit != nil && varName != g.currentValueInit.name {
		g.currentValueInit.deps[varName] = struct{}{}
	}
	return varName
}

func (g *Generator) getLocal(name string) string {
	// First check normal scopes
	for i := len(g.varScopes) - 1; i >= 0; i-- {
		currentScope := &g.varScopes[i]
		if varName, ok := currentScope.names[name]; ok {
			return varName
		}
	}

	// Not in scope - check if we have a captured environment
	if g.currentFnEnv != nil {
		// Look up in the environment using the new public method
		if value, found := g.currentFnEnv.LookupLocal(name); found {
			return g.liftValue(value)
		}
	}

	panic(fmt.Sprintf("variable %s not found in any scope", name))
}

func (g *Generator) markLocalAtom(name string) {
	g.varScopes[len(g.varScopes)-1].localAtoms[name] = true
}

func (g *Generator) getLocalAtom(name string) (string, bool) {
	for i := len(g.varScopes) - 1; i >= 0; i-- {
		scope := &g.varScopes[i]
		if varName, ok := scope.names[name]; ok {
			return varName, scope.localAtoms[name]
		}
	}
	return "", false
}

// allocateTempVar allocates a fresh temporary variable without name tracking
func (g *Generator) allocateTempVar() string {
	if len(g.varScopes) == 0 {
		panic("no variable scope available")
	}

	currentScope := &g.varScopes[len(g.varScopes)-1]
	varName := fmt.Sprintf("tmp%d", currentScope.nextNum)
	currentScope.nextNum++
	return varName
}

var (
	replacements = map[rune]string{
		'!':  "_BANG_",
		'?':  "_QMARK_",
		'-':  "_DASH_",
		'+':  "_PLUS_",
		'*':  "_STAR_",
		'/':  "_SLASH_",
		'=':  "_EQ_",
		'<':  "_LT_",
		'>':  "_GT_",
		'&':  "_AMP_",
		'%':  "_PCT_",
		'$':  "_DOLLAR_",
		'^':  "_CARET_",
		'~':  "_TILDE_",
		'.':  "_DOT_",
		':':  "_COLON_",
		'@':  "_AT_",
		'#':  "_HASH_",
		'\'': "_TICK_",
	}
)

func mungeID(name string) string {
	var sb strings.Builder
	for _, ch := range name {
		if repl, ok := replacements[ch]; ok {
			sb.WriteString(repl)
		} else if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			sb.WriteRune(ch)
		} else {
			// Replace any other non-alphanumeric character with its Unicode code point
			sb.WriteString(fmt.Sprintf("_U%04X_", ch))
		}
	}
	return sb.String()
}

func mungePackageName(name string) string {
	name = mungeID(name)
	if !token.IsIdentifier(name) {
		return "pkg_" + name
	}
	return name
}

func getLastNSPart(ns string) string {
	parts := strings.Split(ns, ".")
	return parts[len(parts)-1]
}

func (g *Generator) pushRecurContext(loopID *lang.Symbol, bindings []string, useGoto bool) {
	g.recurStack = append(g.recurStack, recurContext{
		loopID:   loopID,
		bindings: bindings,
		useGoto:  useGoto,
	})
}

func (g *Generator) popRecurContext() {
	if len(g.recurStack) == 0 {
		panic("no recur context to pop")
	}
	g.recurStack = g.recurStack[:len(g.recurStack)-1]
}

func (g *Generator) currentRecurContext() *recurContext {
	if len(g.recurStack) == 0 {
		return nil // No recur context available
	}
	return &g.recurStack[len(g.recurStack)-1]
}

func (g *Generator) allocVarVar(ns, sym string) string {
	varInfo := varInfo{ns: ns, sym: sym}
	if v, ok := g.varVariables[varInfo]; ok {
		return v
	}

	// also allocate for ns and symbols
	g.allocSymVar(ns)
	g.allocSymVar(sym)

	varName := "var_" + mungeID(ns) + "_" + mungeID(sym)
	g.varVariables[varInfo] = varName
	return varName
}

func (g *Generator) allocSymVar(sym string) string {
	if v, ok := g.symbolVariables[sym]; ok {
		return v
	}
	varName := "sym_" + mungeID(sym)
	g.symbolVariables[sym] = varName
	return varName
}

func (g *Generator) allocKWVar(kw string) string {
	if v, ok := g.kwVariables[kw]; ok {
		return v
	}
	varName := "kw_" + mungeID(kw)
	g.kwVariables[kw] = varName
	return varName
}

////////////////////////////////////////////////////////////////////////////////

var (
	runtimeOwnedVars = map[string]bool{
		// NewEnvironment supplies these roots outside the stdlib loader.
		"defrecord": true,
		"in-ns":     true,
		"record?":   true,
	}

	wellKnownFunctions = map[uintptr]string{
		reflect.ValueOf(lang.NewList).Pointer():      "lang.NewList",
		reflect.ValueOf(lang.MustAsNumber).Pointer(): "lang.MustAsNumber",
		reflect.ValueOf(lang.Equiv).Pointer():        "lang.Equiv",
		reflect.ValueOf(math.IsNaN).Pointer():        "math.IsNaN",
	}
)

func isRuntimeOwnedVar(v *lang.Var) bool {
	// namespace must be clojure.core
	if v.Namespace().Name().Name() != "clojure.core" {
		return false
	}

	return runtimeOwnedVars[v.Symbol().Name()]
}

func getWellKnownFunctionName(fn any) (string, bool) {
	val := reflect.ValueOf(fn)
	// ensure it's a function
	if val.Kind() != reflect.Func {
		return "", false
	}
	ptr := val.Pointer()
	name, ok := wellKnownFunctions[ptr]
	return name, ok
}

func pathToNS(path string) string {
	// remove file extension if present
	if ext := filepath.Ext(path); ext != "" {
		path = path[:len(path)-len(ext)]
	}
	path = strings.ReplaceAll(path, "_", "-")
	path = strings.ReplaceAll(path, "/", ".")
	return path
}

func nsToPath(ns string) string {
	// replace dashes with underscores
	ns = strings.ReplaceAll(ns, "-", "_")
	// replace dots with slashes
	ns = strings.ReplaceAll(ns, ".", "/")
	return ns
}
