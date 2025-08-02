# Glojure Code Generator

This package implements ahead-of-time (AOT) compilation of Glojure code to Go.

## Structure

- `codegen.go` - Core code generation logic
- `codegen_test.go` - Test harness that verifies generated code
- `testdata/` - Test cases with input `.glj` files and expected `.go` output

## Running Tests

```bash
# Run tests and verify generated code matches golden files
go test ./pkg/codegen

# Update golden files when code generation changes
go test ./pkg/codegen -update
```

## How It Works

1. **Input**: Glojure source code (`.glj` files)
2. **Parse**: Use Glojure reader to parse into s-expressions  
3. **Analyze**: Use Glojure analyzer to produce AST nodes
4. **Generate**: Convert AST nodes to Go code
5. **Verify**: Compare with golden files and test behavior

## Current Status

- [x] Basic test harness infrastructure
- [x] OpConst support for numbers, strings, keywords
- [ ] OpConst support for other types (symbols, collections)
- [ ] Variable references (OpVar, OpLocal)
- [ ] Collection literals (OpVector, OpMap, OpSet)
- [ ] Control flow (OpIf, OpDo)
- [ ] Function invocation and definition
- [ ] Behavioral testing (compile and run generated code)

## Next Steps

1. Add support for more constant types
2. Implement behavioral testing that compiles and runs generated code
3. Add variable reference support
4. Expand to collection literals
5. Build up to more complex AST nodes