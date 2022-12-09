package runtime

import (
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"strings"

	"github.com/glojurelang/glojure/value"
)

var (
	builtinPackages []*Package
)

func init() {
	builtinPackages = []*Package{
		&Package{
			Name: "mrat.core",
			Symbols: []*Symbol{
				// importing/requiring other packages
				funcSymbol("load-file", loadFileBuiltin),

				// type constructors
				funcSymbol("list", listBuiltin),
				funcSymbol("vector", vectorBuiltin),
				funcSymbol("char", charBuiltin),
				funcSymbol("str", strBuiltin),

				// collection functions
				funcSymbol("count", lengthBuiltin),
				funcSymbol("conj", conjBuiltin),
				funcSymbol("first", firstBuiltin),
				funcSymbol("rest", restBuiltin),
				funcSymbol("subvec", subvecBuiltin),

				// math functions
				funcSymbol("pow", powBuiltin),
				// TODO: remove this
				funcSymbol("floor", floorBuiltin),
				funcSymbol("*", mulBuiltin),
				funcSymbol("/", divBuiltin),
				funcSymbol(">", gtBuiltin),

				// function application
				funcSymbol("apply", applyBuiltin),

				// test predicates
				funcSymbol("string?", isStringBuiltin),
				funcSymbol("list?", isListBuiltin),
				funcSymbol("vector?", isVectorBuiltin),
				funcSymbol("seq?", isSeqBuiltin),
				funcSymbol("seqable?", isSeqableBuiltin),
				funcSymbol("eq?", eqBuiltin), // TODO: should be =
				funcSymbol("empty?", emptyBuiltin),
				funcSymbol("not-empty?", notEmptyBuiltin),

				// boolean functions
				funcSymbol("not", notBuiltin),
			},
		},
		&Package{
			Name: "mrat.core.io",
			Symbols: []*Symbol{
				funcSymbol("println", printlnBuiltin),
				funcSymbol("pr", prBuiltin),
			},
		},
	}
}

func addBuiltins(env *environment) {
	for _, pkg := range builtinPackages {
		for _, sym := range pkg.Symbols {
			name := pkg.Name + "/" + sym.Name
			if pkg.Name == "mrat.core" {
				// core symbols are available in the global namespace.
				name = sym.Name
			}
			env.Define(value.NewSymbol(name), sym.Value)
		}
	}
}

func funcSymbol(name string, fn func(value.Environment, []interface{}) (interface{}, error)) *Symbol {
	return &Symbol{
		Name: name,
		Value: &value.BuiltinFunc{
			Applyer: value.ApplyerFunc(fn),
			Name:    name,
		},
	}
}

func loadFile(env value.Environment, filename string) error {
	absFile, ok := env.ResolveFile(filename)
	if !ok {
		return fmt.Errorf("could not resolve file %v", filename)
	}

	buf, err := ioutil.ReadFile(absFile)
	if err != nil {
		return fmt.Errorf("error reading file %v: %w", filename, err)
	}

	prog, err := Parse(strings.NewReader(string(buf)), WithFilename(absFile))
	if err != nil {
		return fmt.Errorf("error parsing file %v: %w", filename, err)
	}

	loadEnv := env.PushLoadPaths([]string{filepath.Dir(absFile)})
	_, err = prog.Eval(withEnv(loadEnv))
	if err != nil {
		return fmt.Errorf("error evaluating file %v: %w", filename, err)
	}

	return nil
}

func loadFileBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("load-file expects 1 argument, got %v", len(args))
	}
	filename, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("load-file expects a string, got %v", args[0])
	}
	return nil, loadFile(env, filename)
}

func listBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	return value.NewList(args), nil
}

func vectorBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	return value.NewVector(args), nil
}

func charBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("char expects 1 argument, got %v", len(args))
	}

	switch arg := args[0].(type) {
	case float64:
		return value.NewChar(rune(arg)), nil
	case uint64:
		return value.NewChar(rune(arg)), nil
	case value.Char:
		return arg, nil
	default:
		intVal, ok := asInt(args[0])
		if !ok {
			return nil, fmt.Errorf("can't convert %v (%T) to char", args[0], args[0])
		}
		return value.NewChar(rune(intVal)), nil
	}
}

func strBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) == 0 {
		return "", nil
	}
	if len(args) == 1 {
		return value.ToString(args[0], value.PrintReadably()), nil
	}
	builder := strings.Builder{}
	for _, arg := range args {
		builder.WriteString(value.ToString(arg, value.PrintReadably()))
	}
	return builder.String(), nil
}

func lengthBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("count expects 1 argument, got %v", len(args))
	}

	coll := args[0]

	switch arg := coll.(type) {
	case nil:
		return 0, nil
	case string:
		return len(arg), nil
	case value.Counter:
		return arg.Count(), nil
	case value.ISeqable:
		coll = arg.Seq()
	}
	seq, ok := coll.(value.ISeq)
	if !ok {
		return nil, fmt.Errorf("count expects a collection, got %v", args[0])
	}
	count := 0
	for !seq.IsEmpty() {
		count++
		seq = seq.Rest()
	}
	return count, nil
}

func conjBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("conj expects at least 2 arguments, got %v", len(args))
	}

	conjer, ok := args[0].(value.Conjer)
	if !ok {
		return nil, fmt.Errorf("conj expects a conjer, got %v", args[0])
	}

	return conjer.Conj(args[1:]...), nil
}

func firstBuiltin(env value.Environment, args []interface{}) (out interface{}, err error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("first expects 1 argument, got %v", len(args))
	}

	if seq := value.Seq(args[0]); seq != nil {
		return seq.First(), nil
	}

	return nil, fmt.Errorf("first expects a sequence, got %v", args[0])
}

func restBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("rest expects 1 argument, got %v", len(args))
	}

	if seq := value.Seq(args[0]); seq != nil {
		return seq.Rest(), nil
	}

	return nil, fmt.Errorf("rest expects a sequence, got %v", args[0])
}

func subvecBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("subvec expects 2 or 3 arguments, got %v", len(args))
	}

	v, ok := args[0].(*value.Vector)
	if !ok {
		return nil, fmt.Errorf("subvec expects a vector as its first argument, got %v", args[0])
	}

	startIdx, ok := asInt(args[1])
	if !ok {
		return nil, fmt.Errorf("subvec expects a number as its second argument, got %v", args[1])
	}

	endIdx := v.Count()

	if len(args) == 3 {
		endIdx, ok = asInt(args[2])
		if !ok {
			return nil, fmt.Errorf("subvec expects a number as its third argument, got %v", args[2])
		}
	}

	if startIdx < 0 || startIdx > v.Count() || endIdx < 0 || endIdx > v.Count() {
		return nil, fmt.Errorf("subvec indices out of bounds: %v %v", startIdx, endIdx)
	}

	return v.SubVector(startIdx, endIdx), nil
}

func notBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("not expects 1 argument, got %v", len(args))
	}
	switch arg := args[0].(type) {
	case bool:
		return !arg, nil
	default:
		return nil, fmt.Errorf("not expects a boolean, got %v", arg)
	}
}

func eqBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("Wrong number of arguments (%d) to =", len(args))
	}
	if len(args) == 1 {
		return true, nil
	}

	for i := 1; i < len(args); i++ {
		if !value.Equal(args[0], args[i]) {
			return false, nil
		}
	}
	return true, nil
}

func isStringBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(string)
	return ok, nil
}

func isListBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("list? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(*value.List)
	return ok, nil
}

func isVectorBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(*value.Vector)
	return ok, nil
}

func isSeqBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments (%d) to seq?", len(args))
	}

	if _, ok := args[0].(value.ISeq); ok {
		return true, nil
	}
	return false, nil
}

func isSeqableBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	panic("not implemented")
	return nil, nil
}

func emptyBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("empty? expects 1 argument, got %v", len(args))
	}

	if c, ok := args[0].(value.Counter); ok {
		return c.Count() == 0, nil
	}
	if seq := value.Seq(args[0]); seq != nil {
		return seq.IsEmpty(), nil
	}
	return nil, fmt.Errorf("empty? expects a collection, got %v", args[0])
}

func notEmptyBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	v, err := emptyBuiltin(env, args)
	if err != nil {
		return nil, err
	}
	return notBuiltin(env, []interface{}{v})
}

func powBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pow expects 2 arguments, got %v", len(args))
	}
	a, ok := asDouble(args[0])
	if !ok {
		return nil, fmt.Errorf("pow expects a number, got %v", args[0])
	}
	b, ok := asDouble(args[1])
	if !ok {
		return nil, fmt.Errorf("pow expects a number, got %v", args[1])
	}
	return float64(math.Pow(a, b)), nil
}

func floorBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor expects 1 argument, got %v", len(args))
	}
	switch arg := args[0].(type) {
	case float64:
		return math.Floor(arg), nil
	case int64:
		return arg, nil
	default:
		return nil, fmt.Errorf("floor expects a number, got %v", args[0])
	}
}

func mulBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	isIntMul := true
	intProduct := int64(1)
	floatProduct := float64(1)

	for _, arg := range args {
		switch arg := arg.(type) {
		case float64:
			if isIntMul {
				isIntMul = false
				floatProduct = float64(intProduct)
			}
			floatProduct *= arg
		case int64:
			if isIntMul {
				intProduct *= arg
			} else {
				floatProduct *= float64(arg)
			}
		default:
			return nil, fmt.Errorf("invalid type for *: %v", value.ToString(arg))
		}
	}

	if isIntMul {
		return int64(intProduct), nil
	}
	return float64(floatProduct), nil
}

// TODO: match clojure behavior
func divBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("div expects 2 arguments, got %v", len(args))
	}
	num, ok := asDouble(args[0])
	if !ok {
		return nil, fmt.Errorf("div expects a number as the first argument, got %v", args[0])
	}
	denom, ok := asDouble(args[1])
	if !ok {
		return nil, fmt.Errorf("div expects a number as the second argument, got %v", args[1])
	}
	// TODO: handle generators
	return float64(num / denom), nil
}

func gtBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("Wrong number of arguments (%d) passed to >", len(args))
	}

	prev := args[0]
	for _, arg := range args[1:] {
		switch prev := prev.(type) {
		case float64:
			switch arg := arg.(type) {
			case float64:
				if prev <= arg {
					return false, nil
				}
			case int64:
				if prev <= float64(arg) {
					return false, nil
				}
			default:
				return nil, fmt.Errorf("invalid type for >: %v", value.ToString(arg))
			}
		case int64:
			switch arg := arg.(type) {
			case float64:
				if float64(prev) <= arg {
					return false, nil
				}
			case int64:
				if prev <= arg {
					return false, nil
				}
			default:
				return nil, fmt.Errorf("invalid type for >: %v", value.ToString(arg))
			}
		default:
			return nil, fmt.Errorf("invalid type for >: %v", value.ToString(prev))
		}
		prev = arg
	}

	return true, nil
}

func applyBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("apply expects 2 arguments, got %v", len(args))
	}

	var values []interface{}
	if !value.Equal(nil, args[1]) {
		seq := value.Seq(args[1])
		if seq == nil {
			return nil, fmt.Errorf("apply expects a seqable as the second argument, got %v", args[1])
		}
		for !seq.IsEmpty() {
			values = append(values, seq.First())
			seq = seq.Rest()
		}
	}

	return value.Apply(env, args[0], values)
}

func printlnBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	for i, arg := range args {
		env.Stdout().Write([]byte(value.ToString(arg, value.PrintReadably())))

		if i < len(args)-1 {
			env.Stdout().Write([]byte(" "))
		}
	}
	env.Stdout().Write([]byte("\n"))
	return nil, nil
}

func prBuiltin(env value.Environment, args []interface{}) (interface{}, error) {
	for i, arg := range args {
		env.Stdout().Write([]byte(value.ToString(arg)))

		if i < len(args)-1 {
			env.Stdout().Write([]byte(" "))
		}
	}
	return nil, nil
}

func asInt(v interface{}) (int, bool) {
	return value.AsInt(v)
}

func asDouble(v interface{}) (float64, bool) {
	switch v := v.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case int16:
		return float64(v), true
	case int8:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint8:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	default:
		return 0, false
	}
}
