package runtime

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"go/format"
	"io"
	"path/filepath"
	"reflect"
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
	nextNum int
	names   map[string]string // maps Clojure names to Go variable names
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

type valueInit struct {
	name string              // Name of the variable or var being initialized
	buf  bytes.Buffer        // Buffer holding the initialization code
	deps map[string]struct{} // Set of var/value names this value depends on
}

// Generator handles the conversion of AST nodes to Go code
type Generator struct {
	originalWriter io.Writer

	currentWriter    io.Writer
	currentValueInit *valueInit // current value initialization being generated

	varScopes  []varScope     // stack of variable scopes
	recurStack []recurContext // stack of recur contexts for nested loops

	imports         map[string]string  // set of imported packages with their aliases
	varVariables    map[varInfo]string // map of vars to their Go variable names
	symbolVariables map[string]string  // set of all generated symbols to minimize allocations
	kwVariables     map[string]string  // set of all generated keywords to minimize allocations

	valueInits []*valueInit // map of value initializations

	// Fields for handling closures
	liftedValues  map[liftedKey]*liftedValue // Dedupe by composite key
	liftedCounter int                        // Counter for closed0, closed1...
	currentFnEnv  lang.Environment           // Current function's captured env
}

var (
	omittedVars = map[string]bool{
		// initialized by the runtime
		"#'clojure.core/*in*":            true,
		"#'clojure.core/*out*":           true,
		"#'clojure.core/*compile-files*": true,
		"#'clojure.core/load-file":       true,
	}
)

// NewGenerator creates a new code generator
func NewGenerator(w io.Writer) *Generator {
	return &Generator{
		originalWriter:  w,
		currentWriter:   w,
		varScopes:       []varScope{{nextNum: 0, names: make(map[string]string)}},
		recurStack:      []recurContext{},
		imports:         make(map[string]string),
		varVariables:    make(map[varInfo]string),
		symbolVariables: make(map[string]string),
		kwVariables:     make(map[string]string),
		liftedValues:    make(map[liftedKey]*liftedValue),
		liftedCounter:   0,
	}
}

// Generate takes a namespace and generates Go code that populates the same namespace
func (g *Generator) Generate(ns *lang.Namespace) error {
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

	type namedVar struct {
		name *lang.Symbol
		vr   *lang.Var
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
		// Sort by variable name for deterministic output
		var sortedLifted []*liftedValue
		for _, lifted := range g.liftedValues {
			sortedLifted = append(sortedLifted, lifted)
		}
		sort.Slice(sortedLifted, func(i, j int) bool {
			return sortedLifted[i].varName < sortedLifted[j].varName
		})

		// Generate code for each lifted value
		for _, lifted := range sortedLifted {
			g.startNewValueInit(lifted.varName)
			// Generate the value - this will write any needed initialization
			g.writef("var %s any\n", lifted.varName)
			g.pushVarScope()
			g.writef("{\n")
			valueCode := g.generateValue(lifted.value)
			// Declare the lifted variable with the final value
			g.writef("%s = %s\n", lifted.varName, valueCode)
			g.writef("}\n")
			g.popVarScope()
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
		initBuf.WriteString(fmt.Sprintf("%s := lang.NewSymbol(%q)\n", varName, sym))
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

	////////////////////////////////////////////////////////////////////////////////

	// Prepare the final source
	sourceBytes := []byte(g.header(mungeID(getLastNSPart(ns.Name().String())))) // File header with package and imports
	sourceBytes = append(sourceBytes, initBuf.Bytes()...)                       // The complete init function

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

	g.writef("// %s\n", name.String())
	g.writef("{\n")
	defer g.writef("}\n")

	meta := vr.Meta()
	varSym := g.allocateTempVar()
	var isDynamic bool
	if lang.IsNil(meta) {
		g.writef("%s := %s\n", varSym, g.allocSymVar(name.String()))
	} else {
		metaVariable := g.generateValue(meta)
		g.writef("%s := %s.WithMeta(%s).(*lang.Symbol)\n", varSym, g.allocSymVar(name.String()), metaVariable)
		if RT.BooleanCast(lang.Get(meta, lang.KWDynamic)) {
			isDynamic = true
		}
	}

	// check if the var has a value
	if vr.IsBound() {
		// we call Get() on a new goroutine to ensure we get the root value in the case
		// of dynamic vars
		valChan := make(chan any)
		go func() {
			valChan <- vr.Get()
		}()
		v := <-valChan
		g.writef("%s = %s.InternWithValue(%s, %s, true)\n", varVar, nsVariableName, varSym, g.generateValue(v))
	} else {
		g.writef("%s = %s.Intern(%s)\n", varVar, nsVariableName, varSym)
	}

	// Set metadata on the var if the symbol has metadata
	if meta != nil {
		g.writef("if %s.Meta() != nil {\n", varSym)
		g.writef("\t%s.SetMeta(%s.Meta().(lang.IPersistentMap))\n", varVar, varSym)
		g.writef("}\n")
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
	case lang.Keyword:
		if ns := v.Namespace(); ns != "" {
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
		return fmt.Sprintf("float64(%g)", v)
	case float32:
		return fmt.Sprintf("float32(%g)", v)
	case time.Duration:
		alias := g.addImportWithAlias("time")
		return fmt.Sprintf("%s.Duration(%d)", alias, int64(v))
	case *lang.BigDecimal:
		return g.generateBigDecimalValue(v)
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
			return fname
		}

		panic(fmt.Sprintf("unsupported value type %T: %s", v, v))
	}
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
	buf.WriteString("lang.NewMap(")

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

// generateSetValue generates Go code for a Clojure set
func (g *Generator) generateSetValue(s lang.IPersistentSet) string {
	var buf bytes.Buffer
	buf.WriteString("lang.CreatePersistentTreeSet(lang.NewSliceSeq([]any{")

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

	buf.WriteString("}))")

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

	// Add all methods from the method table
	methodTable := mf.GetMethodTable()
	if methodTable != nil && methodTable.Count() > 0 {
		for seq := lang.Seq(methodTable); seq != nil; seq = seq.Next() {
			entry := seq.First().(lang.IMapEntry)
			dispatchVal := entry.Key()
			method := entry.Val()

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
	panic("cannot generate opaque go function values")
}

func (g *Generator) generateFn(fn *Fn) string {
	// Save and restore current environment
	prevEnv := g.currentFnEnv
	g.currentFnEnv = fn.GetEnvironment() // Set the captured environment for this function
	defer func() { g.currentFnEnv = prevEnv }()

	astNode := fn.ASTNode()
	fnNode := astNode.Sub.(*ast.FnNode)

	// Allocate a variable to return the function
	fnVar := g.allocateTempVar()

	// declare it now to make sure it's in the scope of the caller
	// we may add a nested scope to declare the function in to keep a
	// scoped variable for the function itelf, if the function is named
	g.writef("var %s lang.FnFunc\n", fnVar)

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
			g.writef("var %s lang.FnFunc\n", namedFnVar)
			defer func() {
				g.writeAssign(namedFnVar, fnVar)
				g.writeAssign("_", namedFnVar) // Prevent unused variable warning
			}()
		}
	}

	// If there's only one method and it's not variadic, generate a simple function
	if len(fnNode.Methods) == 1 && !fnNode.IsVariadic {
		method := fnNode.Methods[0]
		methodNode := method.Sub.(*ast.FnMethodNode)

		g.writef("%s = lang.NewFnFunc(func(args ...any) any {\n", fnVar)

		// Check arity
		g.writef("checkArity(args, %d)\n", methodNode.FixedArity)

		// Generate method body
		g.generateFnMethod(methodNode, "args")

		g.writef("})\n")
	} else {
		// Multiple arities or variadic - need to dispatch
		g.writef("%s = lang.NewFnFunc(func(args ...any) any {\n", fnVar)
		g.writef("  switch len(args) {\n")

		// Generate cases for fixed arity methods
		var variadicMethod *ast.Node
		for _, method := range fnNode.Methods {
			methodNode := method.Sub.(*ast.FnMethodNode)
			if methodNode.IsVariadic {
				variadicMethod = method
				continue
			}

			g.writef("  case %d:\n", methodNode.FixedArity)
			g.generateFnMethod(methodNode, "args")
		}

		// Generate default case for variadic method
		if variadicMethod != nil {
			variadicMethodNode := variadicMethod.Sub.(*ast.FnMethodNode)
			g.writef("  default:\n")
			g.writef("checkArityGTE(args, %d)\n", variadicMethodNode.FixedArity)
			g.generateFnMethod(variadicMethodNode, "args")
		} else {
			// No variadic method - error on any other arity
			g.writef("  default:\n")
			g.writef("    checkArity(args, -1)\n")
			g.writef("    panic(\"unreachable\")\n")
		}

		g.writef("  }\n")
		g.writef("})\n")
	}

	// Handle metadata if present
	// NB: we've got metadata with :rettag on our function, but clojure's functions have no metadata...
	// TODO: before merge, investigate this.
	if meta := fn.Meta(); meta != nil {
		metaVar := g.generateValue(meta)
		g.writeAssign(fnVar, fmt.Sprintf("%s.WithMeta(%s).(lang.FnFunc)", fnVar, metaVar))
	}

	// Return the function variable
	return fnVar
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

	// Generate the function expression
	fnExpr := g.generateASTNode(invokeNode.Fn)

	// Generate the arguments
	var argExprs []string
	for _, arg := range invokeNode.Args {
		argExprs = append(argExprs, g.generateASTNode(arg))
	}

	// Allocate a result variable for the invocation
	resultVar := g.allocateTempVar()

	// Emit the invocation
	if len(argExprs) == 0 {
		g.writef("%s := lang.Apply(%s, nil)\n", resultVar, fnExpr)
	} else {
		g.writef("%s := lang.Apply(%s, []any{%s})\n", resultVar, fnExpr, strings.Join(argExprs, ", "))
	}

	// Return the result variable
	return resultVar
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
	testExpr := g.generateASTNode(ifNode.Test)
	g.writef("if lang.IsTruthy(%s) {\n", testExpr)
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
	// if no tests match, we evaluate the default body (if any) and assign to resultVar
	// if no default body, panic
	first := true
	for i, node := range caseNode.Nodes {
		caseNodeNode := node.Sub.(*ast.CaseNodeNode)
		tests := caseNodeNode.Tests
		g.writef("// case clause %d\n", i)
		for _, test := range tests {
			caseTestExpr := g.generateASTNode(test)
			if first {
				g.writef("if lang.Equals(%s, %s) {\n", testExpr, caseTestExpr)
				first = false
			} else {
				g.writef("} else if lang.Equals(%s, %s) {\n", testExpr, caseTestExpr)
			}
			// Generate the then body
			thenExpr := g.generateASTNode(caseNodeNode.Then)
			g.writeAssign(resultVar, thenExpr)
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
	for _, binding := range letNode.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		name := bindingNode.Name.Name()
		init := bindingNode.Init

		// Allocate a Go variable for the Clojure name
		g.writef("// let binding \"%s\"\n", name)

		// Generate initialization code
		initCode := g.generateASTNode(init)
		varName := g.allocateLocal(name)
		g.writef("var %s any = %s\n", varName, initCode)
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

		// Allocate a Go variable for the Clojure name
		g.writef("// letfn binding \"%s\"\n", name)
		varName := g.allocateLocal(name.Name())
		// declare the variable now to allow for recursion
		g.writef("var %s lang.FnFunc\n", varName)
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

	keyValIds := make([]string, 2*len(mapNode.Keys))
	for i, key := range mapNode.Keys {
		keyId := g.generateASTNode(key)

		valNode := mapNode.Vals[i]
		valId := g.generateASTNode(valNode)

		keyValIds[2*i] = keyId   // key
		keyValIds[2*i+1] = valId // value
	}
	mapId := g.allocateTempVar()
	g.writef("%s := lang.NewMap(%s)\n", mapId, strings.Join(keyValIds, ", "))

	return mapId
}

func (g *Generator) generateSet(node *ast.Node) string {
	setNode := node.Sub.(*ast.SetNode)

	itemIds := make([]string, len(setNode.Items))
	for i, item := range setNode.Items {
		itemId := g.generateASTNode(item)
		itemIds[i] = itemId
	}
	setId := g.allocateTempVar()
	g.writef("%s := lang.CreatePersistentTreeSet(lang.NewSliceSeq([]any{%s}))\n", setId, strings.Join(itemIds, ", "))
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
	methodId := g.allocateTempVar()
	g.writef("%s, _ := lang.FieldOrMethod(%s, %q)\n", methodId, tgtId, methodName)
	g.writef("if reflect.TypeOf(%s).Kind() != reflect.Func {\n", methodId)
	g.writef("  panic(lang.NewIllegalArgumentError(fmt.Sprintf(\"%s is not a function\")))\n", methodName)
	g.writef("}\n")

	resultId := g.allocateTempVar()
	g.writef("%s := lang.Apply(%s, []any{%s})\n", resultId, methodId, strings.Join(argIds, ", "))

	return resultId
}

func (g *Generator) generateHostInterop(node *ast.Node) string {
	hostInteropNode := node.Sub.(*ast.HostInteropNode)

	tgtId := g.generateASTNode(hostInteropNode.Target)

	mOrF := hostInteropNode.MOrF.Name()
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

// generateMaybeHostForm generates code for a MaybeHostForm node
func (g *Generator) generateMaybeHostForm(node *ast.Node) string {
	maybeHostNode := node.Sub.(*ast.MaybeHostFormNode)
	class := maybeHostNode.Class
	field := maybeHostNode.Field

	// TODO: implement support for host forms or disallow entirely
	//panic(fmt.Sprintf("unsupported form: %s/%s", maybeHostNode.Class, field))

	fmt.Printf("skipping host form: %s/%s\n", class, field)
	return "nil"
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

	// the interpreter is more lax; it allows for expressions that evaluate to a type
	// here we assume the class is a constant type. clojure's new form is similar
	switch sub := newNode.Class.Sub.(type) {
	case *ast.ConstNode:
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
	// Generate a new alias based on the last part of the package name
	parts := strings.Split(pkg, "/")
	// Use the last part of the package name and current import count
	alias := fmt.Sprintf("%s%d", parts[len(parts)-1], len(g.imports))
	g.imports[pkg] = alias // Store the alias for this package

	return alias
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
		nextNum: nextNum,
		names:   make(map[string]string),
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
			// Create a key for deduplication
			key := g.makeLiftedKey(value)

			// Check if already lifted
			if lifted, ok := g.liftedValues[key]; ok {
				return lifted.varName
			}

			// Create new lifted value
			varName := fmt.Sprintf("closed%d", g.liftedCounter)
			g.liftedCounter++
			g.liftedValues[key] = &liftedValue{
				value:   value,
				varName: varName,
			}

			// Add as a dependency to the current value init if we're in one
			if g.currentValueInit != nil && varName != g.currentValueInit.name {
				g.currentValueInit.deps[varName] = struct{}{}
			}

			return varName
		}
	}

	panic(fmt.Sprintf("variable %s not found in any scope", name))
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
		"in-ns": true,
	}

	wellKnownFunctions = map[uintptr]string{
		reflect.ValueOf(lang.NewList).Pointer(): "lang.NewList",
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

///////////////////////////////////////////////////////////////////////////////

func nodeRecurs(n *ast.Node, loopID string) bool {
	switch n.Op {
	case ast.OpRecur:
		recurNode := n.Sub.(*ast.RecurNode)
		return recurNode.LoopID.Name() == loopID
	case ast.OpDo:
		doNode := n.Sub.(*ast.DoNode)
		return nodeRecurs(doNode.Ret, loopID)
	case ast.OpLet, ast.OpLoop:
		letNode := n.Sub.(*ast.LetNode)
		return nodeRecurs(letNode.Body, loopID)
	case ast.OpLetFn:
		letFnNode := n.Sub.(*ast.LetFnNode)
		return nodeRecurs(letFnNode.Body, loopID)
	case ast.OpIf:
		ifNode := n.Sub.(*ast.IfNode)
		return nodeRecurs(ifNode.Then, loopID) || nodeRecurs(ifNode.Else, loopID)
	case ast.OpTry:
		tryNode := n.Sub.(*ast.TryNode)
		if nodeRecurs(tryNode.Body, loopID) {
			return true
		}
		for _, catch := range tryNode.Catches {
			if nodeRecurs(catch, loopID) {
				return true
			}
		}
	case ast.OpCatch:
		catchNode := n.Sub.(*ast.CatchNode)
		return nodeRecurs(catchNode.Body, loopID)
	case ast.OpCase:
		caseNode := n.Sub.(*ast.CaseNode)
		if nodeRecurs(caseNode.Default, loopID) {
			return true
		}
		for _, branch := range caseNode.Nodes {
			if nodeRecurs(branch, loopID) {
				return true
			}
		}
	case ast.OpCaseNode:
		caseNode := n.Sub.(*ast.CaseNodeNode)
		return nodeRecurs(caseNode.Then, loopID)
	default:
		return false // can't recur in this node type
	}

	return false
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
