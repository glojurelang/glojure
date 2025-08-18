package codegen

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"go/format"
	"io"
	"reflect"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
	"github.com/glojurelang/glojure/pkg/runtime"
)

// TODO
// - handle namespace requires/uses/etc.
// - handle let bindings that are shared across multiple vars

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

// Generator handles the conversion of AST nodes to Go code
type Generator struct {
	originalWriter io.Writer
	w              io.Writer
	varScopes      []varScope     // stack of variable scopes
	recurStack     []recurContext // stack of recur contexts for nested loops

	imports map[string]string // set of imported packages with their aliases
}

// New creates a new code generator
func New(w io.Writer) *Generator {
	return &Generator{
		originalWriter: w,
		w:              w,
		varScopes:      []varScope{{nextNum: 0, names: make(map[string]string)}},
		recurStack:     []recurContext{},
		imports:        make(map[string]string),
	}
}

// Generate takes a namespace and generates Go code that populates the same namespace
func (g *Generator) Generate(ns *lang.Namespace) error {
	// TODO: Implement namespace-based code generation
	// For now, just stub it out
	var buf bytes.Buffer
	g.w = &buf

	g.writef("func init() {\n")

	g.writef("  ns := lang.FindOrCreateNamespace(lang.NewSymbol(%#v))\n", ns.Name().String())
	g.writef("  _ = ns\n")

	// 1. Iterate through ns.Mappings()
	// 2. Generate Go code for each var
	// 3. Create initialization functions
	mappings := ns.Mappings()
	for seq := mappings.Seq(); seq != nil; seq = seq.Next() {
		entry := seq.First()
		name, ok := lang.First(entry).(*lang.Symbol)
		if !ok {
			panic(fmt.Sprintf("expected symbol, got %T", entry))
		}
		second, _ := lang.Nth(entry, 1)
		vr, ok := second.(*lang.Var)
		if !ok {
			panic(fmt.Sprintf("expected var, got %T", second))
		}

		if !(vr.Namespace() == ns && lang.Equals(vr.Symbol(), name)) {
			continue // Skip non-interned mappings
		}

		if err := g.generateVar("ns", name, vr); err != nil {
			return fmt.Errorf("failed to generate code for var %s: %w", name, err)
		}
	}

	g.writef("}\n")

	// Write package header
	sourceBytes := []byte(g.header())
	sourceBytes = append(sourceBytes, buf.Bytes()...)

	// Format the generated code
	formatted, err := format.Source(sourceBytes)
	if err != nil {
		// If formatting fails, write the unformatted code with the error
		return fmt.Errorf("formatting failed: %w\n\nGenerated code:\n%s", err, buf.String())
	}

	// Write formatted code to the original writer
	_, err = g.originalWriter.Write(formatted)
	return err
}

////////////////////////////////////////////////////////////////////////////////

// generateVar generates Go code for a single Var
func (g *Generator) generateVar(nsVariableName string, name *lang.Symbol, vr *lang.Var) error {
	g.pushVarScope()
	defer g.popVarScope()

	fmt.Printf("Generating var: %s\n", name.String())
	g.writef("// %s\n", name.String())
	g.writef("{\n")
	defer g.writef("}\n")

	meta := vr.Meta()
	varSym := g.allocateTempVar()
	if lang.IsNil(meta) {
		g.writef("%s := lang.NewSymbol(\"%s\")\n", varSym, name.String())
	} else {
		metaVariable := g.generateValue(meta)
		g.writef("%s := lang.NewSymbol(\"%s\").WithMeta(%s).(*lang.Symbol)\n", varSym, name.String(), metaVariable)
	}

	// check if the var has a value
	varVar := g.allocateTempVar()
	if vr.IsBound() {
		g.writef("%s := %s.InternWithValue(%s, %s, true)\n", varVar, nsVariableName, varSym, g.generateValue(vr.Get()))
	} else {
		g.writef("%s := %s.Intern(%s)\n", varVar, nsVariableName, varSym)
	}

	// Set metadata on the var if the symbol has metadata
	if meta != nil {
		g.writef("if %s.Meta() != nil {\n", varSym)
		g.writef("\t%s.SetMeta(%s.Meta().(lang.IPersistentMap))\n", varVar, varSym)
		g.writef("}\n")
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
	case *lang.Namespace:
		// Generate code to find or create the namespace
		return fmt.Sprintf("lang.FindOrCreateNamespace(lang.NewSymbol(%#v))", v.Name().String())
	case *runtime.Fn:
		return g.generateFn(v)
	case lang.IPersistentMap:
		return g.generateMapValue(v)
	case lang.IPersistentVector:
		return g.generateVectorValue(v)
	case lang.IPersistentSet:
		return g.generateSetValue(v)
	case lang.Keyword:
		if ns := v.Namespace(); ns != "" {
			return fmt.Sprintf("lang.NewKeyword(\"%s/%s\")", ns, v.Name())
		} else {
			return fmt.Sprintf("lang.NewKeyword(\"%s\")", v.Name())
		}
	case *lang.Symbol:
		return fmt.Sprintf("lang.NewSymbol(\"%s\")", v.FullName())
	case lang.Char:
		return fmt.Sprintf("lang.NewChar(%#v)", rune(v))
	case string:
		// just return the string as a Go string literal
		return fmt.Sprintf("%#v", v)
	case int:
		return fmt.Sprintf("int(%d)", v)
	case int64:
		return fmt.Sprintf("int64(%d)", v)
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
		panic(fmt.Sprintf("unsupported value type %T: %s", v, v))
	}
}

func (g *Generator) generateTypeValue(t reflect.Type) string {
	g.addImport("reflect")

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
		elemExpr := g.generateZeroValueExpr(t.Elem())
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
	buf.WriteString("lang.CreatePersistentTreeSet(")

	idx := 0

	// Iterate through the set elements
	for seq := s.Seq(); seq != nil; seq = seq.Next() {
		if idx > 0 {
			buf.WriteString(", ")
		}
		element := seq.First()
		elementVar := g.generateValue(element)
		buf.WriteString(elementVar)
	}

	buf.WriteString(")")
	return buf.String()
}

func (g *Generator) generateFn(fn *runtime.Fn) string {
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
			defer func() {
				g.writef("%s := %s\n", namedFnVar, fnVar)
				g.writeAssign("_", namedFnVar) // Prevent unused variable warning
			}()
		}
	}

	// If there's only one method and it's not variadic, generate a simple function
	if len(fnNode.Methods) == 1 && !fnNode.IsVariadic {
		method := fnNode.Methods[0]
		methodNode := method.Sub.(*ast.FnMethodNode)

		g.writef("%s = lang.NewFnFunc(func(args ...any) any {\n", fnVar)

		g.addImport("fmt") // Import fmt for error formatting
		// Check arity
		g.writef("  if len(args) != %d {\n", methodNode.FixedArity)
		g.writef("    panic(lang.NewIllegalArgumentError(\"wrong number of arguments (\" + fmt.Sprint(len(args)) + \")\"))\n")
		g.writef("  }\n")

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
			g.writef("    if len(args) < %d {\n", variadicMethodNode.FixedArity)
			g.writef("      panic(lang.NewIllegalArgumentError(\"wrong number of arguments (\" + fmt.Sprint(len(args)) + \")\"))\n")
			g.writef("    }\n")
			g.generateFnMethod(variadicMethodNode, "args")
		} else {
			// No variadic method - error on any other arity
			g.writef("  default:\n")
			g.writef("    panic(lang.NewIllegalArgumentError(\"wrong number of arguments (\" + fmt.Sprint(len(args)) + \")\"))\n")
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
			paramVars[i] = paramVar
		} else {
			// Variadic parameter - collect rest args
			g.writef("%s := lang.NewList(%s[%d:]...)\n", paramVar, argsVar, methodNode.FixedArity)
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
func (g *Generator) generateASTNode(node *ast.Node) string {
	switch node.Op {
	// OpDef
	// OpSetBang
	// OpMap
	// OpLetFn
	// OpGo
	// OpCase
	// OpNew
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
		return g.generateFn(runtime.NewFn(node, nil))
	case ast.OpHostCall:
		return g.generateHostCall(node)
	case ast.OpHostInterop:
		return g.generateHostInterop(node)
	case ast.OpMaybeHostForm:
		return g.generateMaybeHostForm(node)
	case ast.OpTheVar:
		return g.generateTheVar(node)
	default:
		fmt.Printf("Generating code for AST node: %T %+v\n", node.Sub, node.Sub)
		panic(fmt.Sprintf("unsupported AST node type %T", node.Sub))
	}
}

// generateVarDeref generates code for a Var dereference
func (g *Generator) generateVarDeref(node *ast.Node) string {
	varNode := node.Sub.(*ast.VarNode)

	varNamespace := varNode.Var.Namespace()
	varSymbol := varNode.Var.Symbol()

	// generate code to look up the var in the namespace
	nsVar := g.allocateTempVar()
	g.writef("%s := lang.FindNamespace(lang.NewSymbol(\"%s\"))\n", nsVar, varNamespace.Name())
	// look up the var in the namespace
	varId := g.allocateTempVar()
	g.writef("%s := %s.FindInternedVar(lang.NewSymbol(\"%s\"))\n", varId, nsVar, varSymbol.Name())

	// if macro, panic with 'can't take value of macro: %v'
	g.writef("if %s.IsMacro() {\n", varId)
	g.writef("  panic(lang.NewIllegalArgumentError(fmt.Sprintf(\"can't take value of macro: %%v\", %s)))\n", varId)
	g.writef("}\n")
	// else, return Get()
	resultId := g.allocateTempVar()
	g.writef("%s := %s.Get()\n", resultId, varId)

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
		g.writeAssign("_", stmtResult) // Discard intermediate results
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

// generateLet generates code for a Let node
func (g *Generator) generateLet(node *ast.Node, isLoop bool) string {
	letNode := node.Sub.(*ast.LetNode)

	// Push a new variable scope for the let bindings
	g.pushVarScope()
	defer g.popVarScope()

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
		varName := g.allocateLocal(name)

		// Generate initialization code
		initCode := g.generateASTNode(init)
		g.writef("var %s any = %s\n", varName, initCode)
		g.writeAssign("_", varName) // Prevent unused variable warning

		// Collect binding variables for loop
		if isLoop {
			bindingVars = append(bindingVars, varName)
		}
	}

	resultId := g.allocateTempVar()
	if isLoop {
		// Push recur context for this loop
		g.pushRecurContext(letNode.LoopID, bindingVars, false)
		defer g.popRecurContext()

		g.writef("var %s any\n", resultId)
		g.writef("for {\n")
	}

	// Return the body expression (r-value)
	result := g.generateASTNode(letNode.Body)
	if isLoop {
		g.writeAssign(resultId, result)
		g.writef("  break\n") // Break out of the loop after the body
		g.writef("}\n")
		return resultId
	} else {
		return result
	}
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
		_ = g.generateASTNode(tryNode.Finally)
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
	g.writef("%s := lang.CreatePersistentTreeSet(%s)\n", setId, strings.Join(itemIds, ", "))
	return setId
}

func (g *Generator) generateMaybeClass(node *ast.Node) string {
	sym := node.Sub.(*ast.MaybeClassNode).Class.(*lang.Symbol)
	pkg := sym.FullName()

	// find last dot in the package name
	dotIndex := strings.LastIndex(pkg, ".")
	if dotIndex == -1 {
		panic(fmt.Sprintf("invalid package reference: %s", pkg))
	}
	mungedPkgName := pkg[:dotIndex]
	exportedName := pkg[dotIndex+1:]

	packageName := pkgmap.UnmungePkg(mungedPkgName)
	alias := g.addImportWithAlias(packageName)

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

	g.addImport("reflect")

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

	g.addImport("reflect")

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
	field := maybeHostNode.Field

	panic(fmt.Sprintf("unsupported form: %s/%s", maybeHostNode.Class, field))
}

func (g *Generator) generateTheVar(node *ast.Node) string {
	theVarNode := node.Sub.(*ast.TheVarNode)
	varSym := theVarNode.Var
	ns := varSym.Namespace()
	name := varSym.Symbol()

	resultId := g.allocateTempVar()
	g.writef("%s := lang.InternVarName(lang.NewSymbol(\"%s\"), lang.NewSymbol(\"%s\"))\n", resultId, ns.Name(), name.Name())
	return resultId
}

////////////////////////////////////////////////////////////////////////////////

func (g *Generator) addImport(pkg string) {
	parts := strings.Split(pkg, "/")
	alias := parts[len(parts)-1]
	g.imports[pkg] = alias
}

func (g *Generator) addImportWithAlias(pkg string) string {
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

func (g *Generator) header() string {
	header := `// Code generated by glojure codegen. DO NOT EDIT.

package generated

import (
  "github.com/glojurelang/glojure/pkg/lang"
`

	for pkg, alias := range g.imports {
		header += fmt.Sprintf("  %s \"%s\"\n", alias, pkg)
	}

	header += ")\n"
	return header
}

func (g *Generator) writef(format string, args ...any) error {
	_, err := fmt.Fprintf(g.w, format, args...)
	return err
}

// writeAssign writes an assignment iff the r-value string is non-empty
func (g *Generator) writeAssign(varName, rValue string) {
	if rValue == "" {
		return
	}
	g.writef("%s = %s\n", varName, rValue)
}

////////////////////////////////////////////////////////////////////////////////
// Variable Scope Management

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

func (g *Generator) getLocal(name string) string {
	for i := len(g.varScopes) - 1; i >= 0; i-- {
		currentScope := &g.varScopes[i]
		if varName, ok := currentScope.names[name]; ok {
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
	varName := fmt.Sprintf("v%d", currentScope.nextNum)
	currentScope.nextNum++
	return varName
}

func mungeID(name string) string {
	return strings.ReplaceAll(name, "-", "__")
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
