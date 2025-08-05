package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/runtime"
)

// varScope represents a variable allocation scope
type varScope struct {
	nextNum int
	names   map[string]string // maps Clojure names to Go variable names
}

// Generator handles the conversion of AST nodes to Go code
type Generator struct {
	originalWriter io.Writer
	w              io.Writer
	varScopes      []varScope // stack of variable scopes
}

// New creates a new code generator
func New(w io.Writer) *Generator {
	return &Generator{
		originalWriter: w,
		w:              w,
		varScopes:      []varScope{{nextNum: 0, names: make(map[string]string)}},
	}
}

// Generate takes a namespace and generates Go code that populates the same namespace
func (g *Generator) Generate(ns *lang.Namespace) error {
	// TODO: Implement namespace-based code generation
	// For now, just stub it out
	var buf bytes.Buffer
	g.w = &buf

	// Check if we need fmt import (for functions with arity checks)
	needsFmt := false
	mappings := ns.Mappings()

	// Only check vars that are interned in this namespace
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

		// Skip non-interned mappings
		if !(vr.Namespace() == ns && lang.Equals(vr.Symbol(), name)) {
			continue
		}

		if vr.IsBound() {
			if _, ok := vr.Get().(*runtime.Fn); ok {
				needsFmt = true
				break
			}
		}
	}

	// Write package header
	if err := g.writeHeader(needsFmt); err != nil {
		return err
	}

	g.writef("func init() {\n")

	g.writef("  ns := lang.FindOrCreateNamespace(lang.NewSymbol(\"%s\"))\n", ns.Name().String())
	g.writef("  _ = ns\n")

	// TODO: Generate code to populate the namespace
	// This will involve:
	// 1. Iterating through ns.Mappings()
	// 2. Generating Go code for each var
	// 3. Creating initialization functions
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

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
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

	g.writef("// %s\n", name.String())
	g.writef("{\n")
	defer g.writef("}\n")

	meta := name.Meta()
	varSym := g.allocateVar("varSym")
	if meta == nil {
		g.writef("%s := lang.NewSymbol(\"%s\")\n", varSym, name.String())
	} else {
		metaVariable := g.generateValue(meta)
		g.writef("%s := lang.NewSymbol(\"%s\").WithMeta(%s).(*lang.Symbol)\n", varSym, name.String(), metaVariable)
	}

	// check if the var has a value
	if vr.IsBound() {
		g.writef("%s.InternWithValue(%s, %s, true)\n", nsVariableName, varSym, g.generateValue(vr.Get()))
	} else {
		g.writef("%s.Intern(%s)\n", nsVariableName, varSym)
	}

	return nil
}

// returns the variable name or constant expression for the value
func (g *Generator) generateValue(value any) string {
	switch v := value.(type) {
	case *runtime.Fn:
		return g.generateFn(v)
	case *lang.Map:
		return g.generateMap(v)
	case *lang.Vector:
		return g.generateVector(v)
	case lang.Keyword:
		if ns := v.Namespace(); ns != "" {
			return fmt.Sprintf("lang.NewKeyword(\"%s/%s\")", ns, v.Name())
		} else {
			return fmt.Sprintf("lang.NewKeyword(\"%s\")", v.Name())
		}
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

// generateMap generates Go code for a Clojure map
func (g *Generator) generateMap(m *lang.Map) string {
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

// generateVector generates Go code for a Clojure vector
func (g *Generator) generateVector(v *lang.Vector) string {
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
	fnVar := g.allocateVar("fn")

	// Start building the function
	var buf bytes.Buffer

	// Generate an immediately invoked function expression (IIFE) to define and return the function
	buf.WriteString("func() interface{} {\n")

	// Push a new scope for the function definition
	g.pushVarScope()
	defer g.popVarScope()

	// If there's only one method and it's not variadic, generate a simple function
	if len(fnNode.Methods) == 1 && !fnNode.IsVariadic {
		method := fnNode.Methods[0]
		methodNode := method.Sub.(*ast.FnMethodNode)

		buf.WriteString(fmt.Sprintf("  %s := lang.IFnFunc(func(args ...interface{}) interface{} {\n", fnVar))

		// Check arity
		buf.WriteString(fmt.Sprintf("    if len(args) != %d {\n", methodNode.FixedArity))
		buf.WriteString(fmt.Sprintf("      panic(lang.NewIllegalArgumentError(\"wrong number of arguments (\" + fmt.Sprint(len(args)) + \")\"))\n"))
		buf.WriteString("    }\n")

		// Generate method body
		g.generateFnMethod(&buf, methodNode, "args", 0)

		buf.WriteString("  })\n")
	} else {
		// Multiple arities or variadic - need to dispatch
		buf.WriteString(fmt.Sprintf("  %s := lang.IFnFunc(func(args ...interface{}) interface{} {\n", fnVar))
		buf.WriteString("    switch len(args) {\n")

		// Generate cases for fixed arity methods
		var variadicMethod *ast.Node
		for _, method := range fnNode.Methods {
			methodNode := method.Sub.(*ast.FnMethodNode)
			if methodNode.IsVariadic {
				variadicMethod = method
				continue
			}

			buf.WriteString(fmt.Sprintf("    case %d:\n", methodNode.FixedArity))
			g.generateFnMethod(&buf, methodNode, "args", 2)
		}

		// Generate default case for variadic method
		if variadicMethod != nil {
			variadicMethodNode := variadicMethod.Sub.(*ast.FnMethodNode)
			buf.WriteString("    default:\n")
			buf.WriteString(fmt.Sprintf("      if len(args) < %d {\n", variadicMethodNode.FixedArity))
			buf.WriteString(fmt.Sprintf("        panic(lang.NewIllegalArgumentError(\"wrong number of arguments (\" + fmt.Sprint(len(args)) + \")\"))\n"))
			buf.WriteString("      }\n")
			g.generateFnMethod(&buf, variadicMethodNode, "args", 2)
		} else {
			// No variadic method - error on any other arity
			buf.WriteString("    default:\n")
			buf.WriteString(fmt.Sprintf("      panic(lang.NewIllegalArgumentError(\"wrong number of arguments (\" + fmt.Sprint(len(args)) + \")\"))\n"))
		}

		buf.WriteString("    }\n")
		buf.WriteString("  })\n")
	}

	// Handle metadata if present
	if meta := fn.Meta(); meta != nil {
		metaVar := g.generateValue(meta)
		// IFnFunc doesn't support metadata directly, so wrap it
		buf.WriteString(fmt.Sprintf("  // Note: metadata on functions is not yet supported in generated code\n"))
		buf.WriteString(fmt.Sprintf("  // Original metadata: %s\n", metaVar))
		buf.WriteString(fmt.Sprintf("  return %s\n", fnVar))
	} else {
		buf.WriteString(fmt.Sprintf("  return %s\n", fnVar))
	}

	buf.WriteString("}()")

	return buf.String()
}

// generateFnMethod generates the body of a function method
func (g *Generator) generateFnMethod(buf *bytes.Buffer, methodNode *ast.FnMethodNode, argsVar string, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)

	// Push a new scope for the method body
	g.pushVarScope()
	defer g.popVarScope()

	// TODO: Handle recur with a label when we implement recur
	// if methodNode.LoopID != nil {
	//     buf.WriteString(fmt.Sprintf("%s  Recur_%s:\n", indent, mungeID(methodNode.LoopID.Name())))
	// }

	// Bind parameters
	for i, param := range methodNode.Params {
		paramNode := param.Sub.(*ast.BindingNode)
		paramVar := g.allocateVar(paramNode.Name.Name())

		if i < methodNode.FixedArity {
			// Regular parameter
			buf.WriteString(fmt.Sprintf("%s    %s := %s[%d]\n", indent, paramVar, argsVar, i))
		} else {
			// Variadic parameter - collect rest args
			buf.WriteString(fmt.Sprintf("%s    %s := lang.NewList(%s[%d:]...)\n", indent, paramVar, argsVar, methodNode.FixedArity))
		}
	}

	// Generate the body
	bodyVar := g.generateASTNode(methodNode.Body)
	buf.WriteString(fmt.Sprintf("%s    return %s\n", indent, bodyVar))
}

// generateASTNode generates code for an AST node
func (g *Generator) generateASTNode(node *ast.Node) string {
	fmt.Printf("Generating code for AST node: %T %+v\n", node.Sub, node.Sub)
	switch node.Op {
	case ast.OpConst:
		constNode := node.Sub.(*ast.ConstNode)
		return g.generateValue(constNode.Value)
	case ast.OpLocal:
		localNode := node.Sub.(*ast.LocalNode)
		// Look up the variable in our scope
		return g.allocateVar(localNode.Name.Name())
	case ast.OpDo:
		return g.generateDo(node)
	case ast.OpLoop:
		return g.generateLet(node, true)
	case ast.OpIf:
		return g.generateIf(node)
	case ast.OpInvoke:
		return g.generateInvoke(node)
	default:
		panic(fmt.Sprintf("unsupported AST node type %T", node.Sub))
	}
}

// generateDo generates code for a Do node
func (g *Generator) generateDo(node *ast.Node) string {
	var buf bytes.Buffer
	doNode := node.Sub.(*ast.DoNode)
	for _, subNode := range doNode.Statements {
		if subNode == nil {
			continue
		}
		subCode := g.generateASTNode(subNode)
		buf.WriteString(subCode + "\n")
	}
	return g.generateASTNode(doNode.Ret)
}

// generateIf generates code for an If node
func (g *Generator) generateIf(node *ast.Node) string {
	ifNode := node.Sub.(*ast.IfNode)

	test := ifNode.Test
	then := ifNode.Then
	els := ifNode.Else

	// testVal, err := env.EvalAST(test)
	// if err != nil {
	// 	return nil, err
	// }
	// if lang.IsTruthy(testVal) {
	// 	return env.EvalAST(then)
	// } else {
	// 	return env.EvalAST(els)
	// }

	var buf bytes.Buffer
	buf.WriteString("if lang.IsTruthy(")
	buf.WriteString(g.generateASTNode(test) + ") {\n")
	buf.WriteString("  return " + g.generateASTNode(then) + "\n")
	if els != nil {
		buf.WriteString("} else {\n")
		buf.WriteString("  return " + g.generateASTNode(els) + "\n")
	}
	buf.WriteString("}\n")
	return buf.String()
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

	var buf bytes.Buffer

	// Bind variables
	for _, binding := range letNode.Bindings {
		bindingNode := binding.Sub.(*ast.BindingNode)
		name := bindingNode.Name.Name()
		init := bindingNode.Init

		// Allocate a Go variable for the Clojure name
		varName := g.allocateVar(name)

		// Generate initialization code
		initCode := g.generateASTNode(init)
		buf.WriteString(fmt.Sprintf("%s := %s\n", varName, initCode))
	}

	// Generate the body of the let
	bodyCode := g.generateASTNode(letNode.Body)
	buf.WriteString(fmt.Sprintf("return %s\n", bodyCode))

	return buf.String()
}

////////////////////////////////////////////////////////////////////////////////

func (g *Generator) writeHeader(needsFmt bool) error {
	header := `// Code generated by glojure codegen. DO NOT EDIT.

package generated

import (
`
	if needsFmt {
		header += `  "fmt"
`
	}
	header += `  "github.com/glojurelang/glojure/pkg/lang"
)

`
	_, err := io.WriteString(g.w, header)
	return err
}

func (g *Generator) writef(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(g.w, format, args...)
	return err
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

func mungeID(name string) string {
	return strings.ReplaceAll(name, "-", "__")
}
