package codegen

// Example showing the intended use of allocateVar
// This file demonstrates how the variable allocator would be used
// when generating Go code from Clojure expressions.

func exampleUsage(g *Generator) {
	// Example 1: Simple let binding
	// Clojure: (let [x 10 y 20] (+ x y))
	// Would generate something like:
	
	g.writef("// (let [x 10 y 20] (+ x y))\n")
	g.writef("func() interface{} {\n")
	g.pushVarScope() // New scope for let bindings
	
	// Allocate variables for the let bindings
	xVar := g.allocateVar("x")     // Returns "v0"
	yVar := g.allocateVar("y")     // Returns "v1"
	
	// Generate the bindings
	g.writef("  %s := int64(10)\n", xVar)  // v0 := int64(10)
	g.writef("  %s := int64(20)\n", yVar)  // v1 := int64(20)
	
	// Generate the body - when we see 'x' or 'y', we look them up
	xRef := g.allocateVar("x")    // Returns "v0" (same as before)
	yRef := g.allocateVar("y")    // Returns "v1" (same as before)
	
	g.writef("  return %s + %s\n", xRef, yRef) // return v0 + v1
	g.writef("}()\n")
	g.popVarScope()
	
	// Example 2: Nested let bindings with shadowing
	// Clojure: (let [x 10] (let [x 20 y x] (+ x y)))
	
	g.writef("\n// (let [x 10] (let [x 20 y x] (+ x y)))\n")
	g.writef("func() interface{} {\n")
	g.pushVarScope() // Outer let
	
	outerX := g.allocateVar("x")    // Returns "v2" (continuing from previous example)
	g.writef("  %s := int64(10)\n", outerX)  // v2 := int64(10)
	
	// Inner let
	g.writef("  return func() interface{} {\n")
	g.pushVarScope() // Inner let scope
	
	innerX := g.allocateVar("x")    // Returns "v3" (new x, shadows outer x)
	innerY := g.allocateVar("y")    // Returns "v4"
	
	g.writef("    %s := int64(20)\n", innerX)        // v3 := int64(20)
	g.writef("    %s := %s\n", innerY, innerX) // v4 := v3
	
	// In the body, 'x' refers to inner x
	xRef2 := g.allocateVar("x")      // Returns "v3" (finds inner x)
	yRef2 := g.allocateVar("y")      // Returns "v4"
	
	g.writef("    return %s + %s\n", xRef2, yRef2) // return v3 + v4
	g.writef("  }()\n")
	
	g.popVarScope() // Pop inner scope
	g.writef("}()\n")
	g.popVarScope() // Pop outer scope
	
	// Example 3: Function parameters
	// Clojure: (fn [a b] (+ a b))
	
	g.writef("\n// (fn [a b] (+ a b))\n")
	g.writef("func() interface{} {\n")
	g.writef("  return lang.NewFn(func(args ...interface{}) interface{} {\n")
	g.pushVarScope() // Function body scope
	
	// Allocate variables for parameters
	aVar := g.allocateVar("a")      // Returns "v5"
	bVar := g.allocateVar("b")      // Returns "v6"
	
	// Extract parameters from args
	g.writef("    %s := args[0]\n", aVar)  // v5 := args[0]
	g.writef("    %s := args[1]\n", bVar)  // v6 := args[1]
	
	// Generate body
	aRef2 := g.allocateVar("a")      // Returns "v5" (same)
	bRef2 := g.allocateVar("b")      // Returns "v6" (same)
	
	g.writef("    return %s + %s\n", aRef2, bRef2) // return v5 + v6
	g.writef("  })\n")
	g.writef("}()\n")
	g.popVarScope()
}

// The point of the name parameter in allocateVar:
// 
// 1. **Consistency**: When the same Clojure variable is referenced multiple times
//    in the same scope, it should map to the same Go variable. The name ensures
//    we can look up existing allocations.
//
// 2. **Shadowing**: Different scopes can have variables with the same name
//    (like nested lets with the same binding name). The scope stack ensures
//    these get different variable numbers.
//
// 3. **Debugging**: Although we generate names like v0, v1, etc., keeping track
//    of the original Clojure name helps with debugging and potentially generating
//    comments.
//
// Without the name parameter, we'd have to maintain a separate mapping from
// Clojure symbols to variable numbers outside the generator, which would be
// more complex and error-prone.