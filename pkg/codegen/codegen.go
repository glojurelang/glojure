package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
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

	g.writef("  ns := lang.FindOrCreateNamespace(lang.NewSymbol(\"%s\"))\n", ns.Name().String())
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

	meta := name.Meta()
	varSym := g.allocateTempVar()
	if meta == nil {
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

// returns the variable name or constant expression for the value
func (g *Generator) generateValue(value any) string {
	switch v := value.(type) {
	case *runtime.Fn:
		return g.generateFn(v)
	case *lang.Map:
		return g.generateMapValue(v)
	case *lang.Vector:
		return g.generateVectorValue(v)
	case *lang.SubVector:
		// XXX TODO: handle sub-vectors
		return fmt.Sprintf("%#v", "subvector not implemented yet")
	case lang.Keyword:
		if ns := v.Namespace(); ns != "" {
			return fmt.Sprintf("lang.NewKeyword(\"%s/%s\")", ns, v.Name())
		} else {
			return fmt.Sprintf("lang.NewKeyword(\"%s\")", v.Name())
		}
	case *lang.Symbol:
		return fmt.Sprintf("lang.NewSymbol(\"%s\")", v.FullName())
	case string:
		// just return the string as a Go string literal
		return fmt.Sprintf("%#v", v)
	case int:
		return fmt.Sprintf("int(%d)", v)
	case int64:
		return fmt.Sprintf("int64(%d)", v)
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

// generateMapValue generates Go code for a Clojure map
func (g *Generator) generateMapValue(m *lang.Map) string {
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
	if buf.Len() > 0 {
		buf.Truncate(buf.Len() - 2)
	}

	buf.WriteString(")")
	return buf.String()
}

// generateVectorValue generates Go code for a Clojure vector
func (g *Generator) generateVectorValue(v *lang.Vector) string {
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

func (g *Generator) generateFn(fn *runtime.Fn) string {
	astNode := fn.ASTNode()
	fnNode := astNode.Sub.(*ast.FnNode)

	// Allocate a variable for the function
	fnVar := g.allocateTempVar()

	// Push a new scope for the function definition
	g.pushVarScope()
	defer g.popVarScope()

	// If there's only one method and it's not variadic, generate a simple function
	if len(fnNode.Methods) == 1 && !fnNode.IsVariadic {
		method := fnNode.Methods[0]
		methodNode := method.Sub.(*ast.FnMethodNode)

		g.writef("%s := lang.IFnFunc(func(args ...any) any {\n", fnVar)

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
		g.writef("%s := lang.IFnFunc(func(args ...any) any {\n", fnVar)
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
	if meta := fn.Meta(); meta != nil {
		metaVar := g.generateValue(meta)
		// IFnFunc doesn't support metadata directly, so wrap it
		g.writef("// Note: metadata on functions is not yet supported in generated code\n")
		g.writef("// Original metadata: %s\n", metaVar)
	}

	// Return the function variable
	return fnVar
}

// generateFnMethod generates the body of a function method
func (g *Generator) generateFnMethod(methodNode *ast.FnMethodNode, argsVar string) {
	// Push a new scope for the method body
	g.pushVarScope()
	defer g.popVarScope()

	// TODO: Handle recur with a label

	// Bind parameters
	for i, param := range methodNode.Params {
		paramNode := param.Sub.(*ast.BindingNode)
		paramVar := g.allocateVar(paramNode.Name.Name())

		if i < methodNode.FixedArity {
			// Regular parameter
			g.writef("%s := %s[%d]\n", paramVar, argsVar, i)
		} else {
			// Variadic parameter - collect rest args
			g.writef("%s := lang.NewList(%s[%d:]...)\n", paramVar, argsVar, methodNode.FixedArity)
		}
	}

	// Generate the body
	bodyVar := g.generateASTNode(methodNode.Body)
	if bodyVar != "" {
		g.writef("return %s\n", bodyVar)
	}
	// If bodyVar is empty (e.g., from throw), no return is generated
}

// generateASTNode generates code for an AST node
func (g *Generator) generateASTNode(node *ast.Node) string {
	switch node.Op {
	// OpDef
	// OpSetBang
	// OpFn
	// OpMap
	// OpSet
	// OpLetFn
	// OpQuote
	// OpGo
	// OpHostCall
	// OpHostInterop
	// OpMaybeHostForm
	// OpCase
	// OpTheVar
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
	case ast.OpLocal:
		localNode := node.Sub.(*ast.LocalNode)
		// Look up the variable in our scope
		return g.allocateVar(localNode.Name.Name())
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

// func (env *environment) EvalASTLet(n *ast.Node, isLoop bool) (interface{}, error) {
// 	letNode := n.Sub.(*ast.LetNode)

// 	newEnv := env.PushScope().(*environment)

// 	var bindNameVals []interface{}

// 	bindings := letNode.Bindings
// 	for _, binding := range bindings {
// 		bindingNode := binding.Sub.(*ast.BindingNode)

// 		name := bindingNode.Name
// 		init := bindingNode.Init
// 		initVal, err := newEnv.EvalAST(init)
// 		if err != nil {
// 			return nil, err
// 		}
// 		// TODO: this should not mutate in-place!
// 		newEnv.BindLocal(name, initVal)

// 		bindNameVals = append(bindNameVals, name, initVal)
// 	}

// Recur:
// 	for i := 0; i < len(bindNameVals); i += 2 {
// 		name := bindNameVals[i].(*lang.Symbol)
// 		val := bindNameVals[i+1]
// 		newEnv.BindLocal(name, val)
// 	}

// 	rt := lang.NewRecurTarget()
// 	recurEnv := newEnv.WithRecurTarget(rt).(*environment)
// 	recurErr := &lang.RecurError{Target: rt}

// 	res, err := recurEnv.EvalAST(letNode.Body)
// 	if isLoop && errors.As(err, &recurErr) {
// 		newVals := recurErr.Args
// 		if len(newVals) != len(bindNameVals)/2 {
// 			return nil, env.errorf(n, "invalid recur, expected %d arguments, got %d", len(bindNameVals)/2, len(newVals))
// 		}
// 		for i := 0; i < len(bindNameVals); i += 2 {
// 			newValsIndex := i / 2
// 			val := newVals[newValsIndex]
// 			bindNameVals[i+1] = val
// 		}
// 		goto Recur
// 	}
// 	return res, err
// }

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
		varName := g.allocateVar(name)

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
		g.recurStack = append(g.recurStack, recurContext{
			loopID:   letNode.LoopID,
			bindings: bindingVars,
		})
		defer func() {
			// Pop recur context when done
			g.recurStack = g.recurStack[:len(g.recurStack)-1]
		}()

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
		tempVar := g.allocateVar(fmt.Sprintf("recurTemp%d", i))
		tempVars[i] = tempVar
		exprCode := g.generateASTNode(expr)
		g.writef("var %s any = %s\n", tempVar, exprCode)
	}

	// Assign the temporary values to the loop bindings
	for i, bindingVar := range ctx.bindings {
		g.writef("%s = %s\n", bindingVar, tempVars[i])
	}

	// Continue the loop
	g.writef("continue\n")

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
			catchVar := g.allocateVar(bindingNode.Name.Name())
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

// AllocateVar allocates a Go variable name for the given Clojure name in the current scope
// If the name already exists in the current scope, it returns the existing Go variable name
func (g *Generator) allocateVar(name string) string {
	if len(g.varScopes) == 0 {
		panic("no variable scope available")
	}

	currentScope := &g.varScopes[len(g.varScopes)-1]

	// Check if already allocated in current scope
	if varName, exists := currentScope.names[name]; exists {
		return varName
	}

	// Allocate new variable name
	varName := fmt.Sprintf("v%d", currentScope.nextNum)
	currentScope.names[name] = varName
	currentScope.nextNum++

	return varName
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
