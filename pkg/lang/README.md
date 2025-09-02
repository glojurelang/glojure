# Glojure Lang Package Reference

This document lists all the public functions available in the `pkg/lang` package that can be called from Go using `lang.Foo(...)`.

## Sequence Operations

### Core Sequence Functions
- `First(x interface{}) interface{}` - Returns the first element of a sequence
- `Rest(x interface{}) interface{}` - Returns all elements after the first
- `Next(x interface{}) ISeq` - Returns the next sequence (nil if empty)
- `Seq(x interface{}) ISeq` - Converts a value to a sequence
- `IsSeq(x interface{}) bool` - Checks if a value is a sequence

### Sequence Construction
- `NewCons(x any, xs any) ISeq` - Creates a new cons cell
- `NewList(values ...any) IPersistentList` - Creates a new list
- `NewVector(values ...any) *Vector` - Creates a new vector
- `NewSliceSeq(x interface{}) ISeq` - Creates a sequence from a Go slice
- `NewStringSeq(x string, start int) ISeq` - Creates a sequence from a string
- `NewGoMapSeq(x interface{}) ISeq` - Creates a sequence from a Go map

## Collection Operations

### Vector Operations
- `NewSubVector(meta IPersistentMap, v IPersistentVector, start, end int) *SubVector`
- `NewTransientVector() ITransientVector`

### List Operations
- `ConsList(item any, next *List) *List`

### Map Operations
- `NewMap(keyVals ...any) IPersistentMap` - Creates a new persistent map
- `SafeMerge(m1, m2 IPersistentMap) IPersistentMap` - Safely merges two maps
- `Merge(m1, m2 IPersistentMap) IPersistentMap` - Merges two maps
- `NewMapEntry(key, val any) *MapEntry` - Creates a new map entry
- `NewMapSeq(kvs []any) ISeq` - Creates a sequence from key-value pairs
- `NewMapKeySeq(s ISeq) ISeq` - Creates a sequence of map keys
- `NewMapValSeq(s ISeq) ISeq` - Creates a sequence of map values

### Collection Utilities
- `Count(coll any) int` - Returns the count of elements in a collection
- `Keys(m Associative) ISeq` - Returns a sequence of map keys
- `Vals(m Associative) ISeq` - Returns a sequence of map values
- `Subvec(v IPersistentVector, start, end int) IPersistentVector` - Creates a subvector

## Functional Programming

### Reduce Operations
- `Reduce(f func(interface{}, interface{}) interface{}, seq ISeq) interface{}`
- `ReduceInit(f func(interface{}, interface{}) interface{}, init interface{}, seq ISeq) interface{}`
- `ReduceKV(f func(init, k, v interface{}) interface{}, init, coll interface{}) interface{}`

### Apply Operations
- `Apply(fn interface{}, args []interface{}) interface{}` - Applies a function to arguments

## Data Structure Creation

### Atom Operations
- `NewAtom(val any) *Atom` - Creates a new atom
- `NewAtomWithMeta(val any, meta IPersistentMap) *Atom` - Creates an atom with metadata

### Reference Types
- `NewRef(val interface{}) *Ref` - Creates a new ref
- `NewVolatile(val interface{}) *Volatile` - Creates a new volatile reference

### Multi-Methods
- `NewMultiFn(name string, dispatchFn IFn, defaultDispatchVal interface{}, hierarchy IRef) *MultiFn`

### Box and Container Operations
- `NewBox(val interface{}) *Box` - Creates a new box

## String and Character Operations

### String Functions
- `ConcatStrings(strs ...string) string` - Concatenates multiple strings
- `ToString(v interface{}) string` - Converts a value to string
- `PrintString(v interface{}) string` - Prints a value to string
- `Print(x interface{}, w io.Writer)` - Prints a value to a writer

### Character Functions
- `NewChar(value rune) Char` - Creates a new character
- `CharAt(s string, idx int) Char` - Gets character at index
- `RuneFromCharLiteral(lit string) (rune, error)` - Parses character literal
- `CharLiteralFromRune(rn rune) string` - Creates character literal

## Numeric Operations

### Number Methods
- `Numbers.UncheckedAdd(x, y any) any`
- `Numbers.UncheckedDec(x any) any`
- `Numbers.UncheckedIntDivide(x, y int) any`
- `Numbers.Add(x, y any) any`
- `Numbers.AddP(x, y any) any`
- `Numbers.Minus(x, y any) any`
- `Numbers.MinusP(x, y any) any`
- `Numbers.Multiply(x, y any) any`
- `Numbers.MultiplyP(x, y any) any`
- `Numbers.Divide(x, y any) any`
- `Numbers.Quotient(x, y any) any`
- `Numbers.Remainder(x, y any) any`
- `Numbers.Unchecked_minus(x, y any) any`
- `Numbers.Unchecked_negate(x any) any`
- `Numbers.Unchecked_multiply(x, y any) any`

### Number Conversion
- `AsInt(v any) (int, bool)` - Converts to int
- `MustAsInt(v any) int` - Converts to int (panics on failure)
- `AsFloat64(x any) float64` - Converts to float64
- `AsByte(x any) byte` - Converts to byte
- `AsNumber(x interface{}) (interface{}, bool)` - Converts to number

## Type and Reflection Operations

### Type Functions
- `HasType(t reflect.Type, v interface{}) bool` - Checks if value has specific type
- `TypeOf(v interface{}) reflect.Type` - Gets type of value
- `FieldOrMethod(v interface{}, name string) (interface{}, bool)` - Gets field or method
- `SetField(target interface{}, name string, val interface{}) error` - Sets struct field

## Comparison and Equality

### Equality Functions
- `Equals(a, b any) bool` - Checks equality
- `Equiv(a, b any) bool` - Checks equivalence
- `Identical(a, b any) bool` - Checks identity

### Comparison Functions
- `Compare(x, y any) int` - Compares two values
- `NumbersEqual(x, y interface{}) bool` - Compares numbers for equality

## Truthiness and Nil Checking

### Truthiness Functions
- `IsTruthy(v interface{}) bool` - Checks if value is truthy
- `IsNil(v interface{}) bool` - Checks if value is nil

## Error Handling

### Error Creation
- `NewError(msg string) error` - Creates a new error
- `NewTimeoutError(msg string) error` - Creates a timeout error
- `NewIndexOutOfBoundsError() error` - Creates index out of bounds error
- `NewIllegalArgumentError(msg string) error` - Creates illegal argument error
- `NewIllegalStateError(msg string) error` - Creates illegal state error
- `NewUnsupportedOperationError(msg string) error` - Creates unsupported operation error
- `NewArithmeticError(msg string) error` - Creates arithmetic error
- `NewCompilerError(file string, line, col int, err error) error` - Creates compiler error

### Exception Handling
- `CatchMatches(r, expect any) bool` - Checks if caught value matches expected type

## Namespace and Symbol Management

### Namespace Functions
- `Namespaces() []*Namespace` - Gets all namespaces
- `FindNamespace(sym *Symbol) *Namespace` - Finds namespace by symbol
- `FindOrCreateNamespace(sym *Symbol) *Namespace` - Finds or creates namespace
- `RemoveNamespace(sym *Symbol)` - Removes namespace
- `NamespaceFor(inns *Namespace, sym *Symbol) *Namespace` - Gets namespace for symbol
- `NewNamespace(name *Symbol) *Namespace` - Creates new namespace

### Symbol Functions
- `NewSymbol(s string) *Symbol` - Creates a new symbol
- `InternSymbol(ns, name interface{}) *Symbol` - Interns a symbol

### Keyword Functions
- `NewKeyword(s string) Keyword` - Creates a new keyword
- `InternKeywordSymbol(s *Symbol) Keyword` - Interns keyword from symbol
- `InternKeywordString(s string) Keyword` - Interns keyword from string
- `InternKeyword(ns, name interface{}) Keyword` - Interns keyword

## Variable Management

### Variable Functions
- `InternVar(ns *Namespace, sym *Symbol, root interface{}, replaceRoot bool) *Var`
- `InternVarReplaceRoot(ns *Namespace, sym *Symbol, root interface{}) *Var`
- `InternVarName(nsSym, nameSym *Symbol) *Var`
- `NewVar(ns *Namespace, sym *Symbol) *Var`

## Collection Utilities

### Indexing and Access
- `Nth(x interface{}, n int) (interface{}, bool)` - Gets nth element
- `MustNth(x interface{}, i int) interface{}` - Gets nth element (panics on failure)

### Sorting
- `SortSlice(slice []any, comp any)` - Sorts a slice with comparator

## Agent and Future Operations

### Agent Functions
- `ShutdownAgents()` - Shuts down all agents
- `AgentSubmit(fn IFn) IBlockingDeref` - Submits function to agent

## Writer Operations

### Writer Functions
- `AppendWriter(w io.Writer, v interface{}) io.Writer` - Appends to writer
- `WriteWriter(w io.Writer, v interface{}) io.Writer` - Writes to writer

## Transaction Support

### Transaction Functions
- `LockingTransaction.RunInTransaction(fn IFn) interface{}` - Runs function in transaction

## Built-in Go Functions

The package also provides access to Go built-in functions:
- `GoAppend(slc interface{}, vals ...interface{}) interface{}`
- `GoCopy(dst, src interface{}) int`
- `GoDelete(m, key interface{})`
- `GoLen(v interface{}) int`
- `GoCap(v interface{}) int`
- `GoMake(typ reflect.Type, args ...interface{}) interface{}`
- `GoNew(typ reflect.Type) interface{}`
- `GoComplex(real, imag interface{}) interface{}`
- `GoReal(c interface{}) interface{}`
- `GoImag(c interface{}) interface{}`
- `GoClose(c interface{})`
- `GoPanic(v interface{})`
- `GoDeref(ptr interface{}) interface{}`
- `GoIndex(slc, i interface{}) interface{}`
- `GoMapIndex(m, k interface{}) interface{}`
- `GoSetMapIndex(m, k, v interface{})`
- `GoSlice(slc interface{}, indices ...interface{}) interface{}`
- `GoChanOf(typ reflect.Type) reflect.Type`
- `GoRecvChanOf(typ reflect.Type) reflect.Type`
- `GoSendChanOf(typ reflect.Type) reflect.Type`
- `GoSend(ch, val interface{})`
- `GoRecv(ch interface{}) (interface{}, bool)`

## Usage Examples

```go
package main

import (
    "fmt"
    "github.com/glojurelang/glojure/pkg/lang"
)

func main() {
    // Create a list
    list := lang.NewList(1, 2, 3, 4, 5)

    // Get first element
    first := lang.First(list)
    fmt.Println("First:", first) // Output: First: 1

    // Create a vector
    vec := lang.NewVector("a", "b", "c")

    // Get element at index
    elem := vec.Nth(1)
    fmt.Println("Element at 1:", elem) // Output: Element at 1: b

    // Create an atom
    atom := lang.NewAtom(42)
    fmt.Println("Atom value:", atom.Deref()) // Output: Atom value: 42

    // Create a symbol
    sym := lang.NewSymbol("my-symbol")
    fmt.Println("Symbol:", sym.Name()) // Output: Symbol: my-symbol

    // Create a keyword
    kw := lang.NewKeyword("my-keyword")
    fmt.Println("Keyword:", kw) // Output: Keyword: :my-keyword

    // Create a map
    m := lang.NewMap("a", 1, "b", 2)
    fmt.Println("Map:", m) // Output: Map: {"a" 1 "b" 2}

    // Use reduce
    sum := lang.ReduceInit(func(acc, val interface{}) interface{} {
        return acc.(int) + val.(int)
    }, 0, list)
    fmt.Println("Sum:", sum) // Output: Sum: 15
}
```

## Notes

- All functions that take `interface{}` parameters accept any Go value
- Functions that return `ISeq` return sequences that can be iterated
- Many functions have both safe versions (returning bool) and unsafe versions (panicking on failure)
- The package provides both Clojure-style persistent data structures and Go-native operations
- Error handling follows Go conventions with panic/recover for exceptional cases
- The package includes comprehensive support for Clojure's core data structures and operations
