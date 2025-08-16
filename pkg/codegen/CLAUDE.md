# Glojure Codegen System

This document provides guidance for understanding and working with Glojure's ahead-of-time (AOT) code generation system.

## Overview

The codegen package transforms Glojure AST nodes into Go source code, enabling ahead-of-time compilation. This is a work-in-progress alternative to the default tree-walking interpreter that offers potential performance benefits through static compilation.

## Architecture

### Compilation Pipeline

```
Source (.glj) → Reader → S-expressions → Analyzer → AST → Codegen → Go Source → go build → Native Binary
                                                      ↓
                                                   Runtime Eval (default path)
```

### Key Components

- **Generator** (pkg/codegen/codegen.go:32-46): Main code generation engine
  - Manages variable scopes and recur contexts
  - Handles output buffering and Go code formatting
  
- **AST Nodes** (pkg/ast/ast.go:17-158): 44 different operation types
  - Each node has an `Op` field determining its type
  - `Sub` field contains op-specific data structures

- **Analyzer** (pkg/compiler/analyze.go): Creates AST from S-expressions
  - Performs macro expansion (pkg/compiler/analyze.go:87-122)
  - Manages lexical environments (pkg/compiler/analyze.go:32-51)
  - Dispatches to specialized analyzers (pkg/compiler/analyze.go:196-408)

## Current Implementation Status

### ✅ Supported Features

| Feature | Implementation | Reference |
|---------|----------------|-----------|
| Constants | Numbers, strings, keywords, booleans, nil | codegen.go:383-385 |
| Local Variables | Let bindings, function parameters | codegen.go:386-389 |
| Namespace Vars | Var dereference and lookup | codegen.go:410-433 |
| Functions | Single/multi-arity, variadic | codegen.go:258-331 |
| Let/Loop | Including loop/recur | codegen.go:555-614 |
| Recur | Tail recursion within loops | codegen.go:616-658 |
| If/Else | Conditional expressions | codegen.go:479-503 |
| Do Blocks | Sequential evaluation | codegen.go:462-477 |
| Function Calls | Via lang.Apply | codegen.go:435-460 |
| Collections | Vectors, Maps | codegen.go:215-256 |

### ❌ Not Yet Implemented

- Host interop (., .., new)
- Try/catch/finally
- Case expressions
- Set literals
- Metadata on functions
- deftype/defprotocol
- Lazy sequences
- Transducers

## Code Generation Process

### 1. Namespace Generation (codegen.go:50-132)

```go
func (g *Generator) Generate(ns *lang.Namespace) error
```

- Iterates through namespace mappings
- Generates init() function containing var definitions
- Applies go fmt to output

### 2. Var Generation (codegen.go:136-170)

Each var becomes:
```go
{
  varSym := lang.NewSymbol("var-name")
  var := ns.InternWithValue(varSym, value, true)
  // metadata handling...
}
```

### 3. Value Generation (codegen.go:173-213)

Recursively generates Go expressions for Clojure values:
- Primitives: Direct Go literals
- Collections: `lang.NewVector(...)`, `lang.NewMap(...)`
- Functions: `lang.IFnFunc(func(args ...any) any { ... })`

### 4. AST Node Generation (codegen.go:361-408)

Dispatches on `node.Op` to specialized generators:
- Control flow nodes generate Go control structures
- Expression nodes generate Go expressions
- Special forms have custom handling

## Variable Scope Management

### Scope Stack (codegen.go:19-23, 696-741)

```go
type varScope struct {
    nextNum int                    // Counter for unique var names
    names   map[string]string      // Clojure name → Go var name
}
```

- Each let/fn/loop pushes new scope
- Variables allocated as v0, v1, v2...
- Scopes inherit counter from parent

### Example Scoping

```clojure
(let [x 1]           ; x → v0
  (let [x 2 y 3]     ; x → v1 (shadows), y → v2
    (+ x y)))        ; references v1, v2
```

## Loop/Recur Implementation

### Recur Context (codegen.go:25-29)

```go
type recurContext struct {
    loopID   *lang.Symbol  // Matches recur to its loop
    bindings []string      // Go variable names for rebinding
}
```

### Generated Pattern (codegen.go:589-614, 616-658)

```go
// (loop [x 0] ... (recur (inc x)))
var v0 any = 0
for {
    // body...
    var recurTemp0 any = v0 + 1  // Evaluate recur args
    v0 = recurTemp0               // Rebind
    continue                      // Loop
}
```

## Testing Infrastructure

### Test Harness (pkg/codegen/codegen_test.go)

1. **Golden Files** (codegen_test.go:24-71): Compare generated output
   - Input: `testdata/*.glj`
   - Expected: `testdata/*.glj.expected`

2. **Go Vet Validation** (codegen_test.go:207-223): Ensures valid Go syntax

3. **Behavioral Tests** (codegen_test.go:72-172): Run generated code
   - Compiles to temporary binary
   - Executes -main function
   - Verifies output

### Running Tests

```bash
# Run all codegen tests
go test ./pkg/codegen/...

# Update golden files
go test ./pkg/codegen/... -update

# Verbose output with generated code
go test ./pkg/codegen/... -v
```

## Extending the Codegen

### Adding New AST Node Support

1. Add case in `generateASTNode()` (codegen.go:361-408)
2. Implement generator function following pattern:
   ```go
   func (g *Generator) generateNewOp(node *ast.Node) string {
       newOpNode := node.Sub.(*ast.NewOpNode)
       // Generate Go code...
       resultVar := g.allocateVar("result")
       g.writef("...")
       return resultVar
   }
   ```
3. Add test case in `testdata/`
4. Run tests with `-update` to create expected output

### Common Patterns

**R-values vs Statements**: Generators return variable names (r-values) and emit statements to `g.w`:
```go
testExpr := g.generateASTNode(node.Test)    // Get r-value
g.writef("if lang.IsTruthy(%s) {\n", testExpr)  // Use in statement
```

**Temporary Variables**: Use `allocateVar()` for unique names:
```go
tempVar := g.allocateVar("temp")
g.writef("%s := complexExpression()\n", tempVar)
```

**Scope Management**: Always push/pop for new lexical scopes:
```go
g.pushVarScope()
defer g.popVarScope()
```

## Debugging Tips

1. **Examine Generated Code**: Tests output generated code on failure
2. **Check AST Structure**: Use `fmt.Printf("%#v\n", node)` to inspect
3. **Trace Execution**: Add logging to generator methods
4. **Validate Manually**: Copy generated code to test file and run

## Integration Points

### Runtime Compatibility

Generated code uses same primitives as runtime:
- `lang.Apply()` for function calls (pkg/lang/ifn.go:8-25)
- `lang.IsTruthy()` for conditionals (pkg/lang/truthy.go:3-18)
- `lang.NewList/Vector/Map()` for collections (pkg/lang/collections.go)

### Namespace System

Generated code integrates with runtime namespaces:
- `lang.FindOrCreateNamespace()` (pkg/lang/namespace.go:340-350)
- `ns.InternWithValue()` (pkg/lang/namespace.go:112-125)
- Vars are accessible from REPL after loading

## Future Directions

1. **Full AST Coverage**: Implement remaining node types
2. **Optimization**: Dead code elimination, constant folding
3. **Integration**: Add `glj compile` command for AOT compilation
4. **Performance**: Benchmark against runtime interpreter
5. **Debugging**: Source maps for generated code

## Related Files

- **AST Definition**: pkg/ast/ast.go
- **Analyzer**: pkg/compiler/analyze.go
- **Runtime Evaluator**: pkg/runtime/evalast.go (comparison reference)
- **Test Data**: pkg/codegen/testdata/*.glj
- **Language Primitives**: pkg/lang/*.go