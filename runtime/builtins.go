package runtime

import (
	"context"
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

				// collection functions
				funcSymbol("count", lengthBuiltin),
				funcSymbol("conj", conjBuiltin),
				funcSymbol("concat", concatBuiltin),
				funcSymbol("first", firstBuiltin),
				funcSymbol("rest", restBuiltin),
				funcSymbol("subvec", subvecBuiltin),

				// math functions
				funcSymbol("pow", powBuiltin),
				funcSymbol("floor", floorBuiltin),
				funcSymbol("*", mulBuiltin),
				funcSymbol("/", divBuiltin),
				funcSymbol("+", addBuiltin),
				funcSymbol("-", subBuiltin),
				funcSymbol("<", ltBuiltin),
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
			env.Define(name, sym.Value)
		}
	}
}

func funcSymbol(name string, fn func(value.Environment, []value.Value) (value.Value, error)) *Symbol {
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

func loadFileBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("load-file expects 1 argument, got %v", len(args))
	}
	filename, ok := args[0].(*value.Str)
	if !ok {
		return nil, fmt.Errorf("load-file expects a string, got %v", args[0])
	}
	return nil, loadFile(env, filename.Value)
}

func listBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	return value.NewList(args), nil
}

func vectorBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	return value.NewVector(args), nil
}

func charBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("char expects 1 argument, got %v", len(args))
	}

	switch arg := args[0].(type) {
	case *value.Num:
		return value.NewChar(rune(arg.Value)), nil
	case *value.Char:
		return arg, nil
	default:
		return nil, fmt.Errorf("can't convert %v to char", args[0])
	}
}

func lengthBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("length expects 1 argument, got %v", len(args))
	}

	if args[0] == nil {
		return value.NewNum(0), nil
	}

	if c, ok := args[0].(value.Counter); ok {
		return value.NewNum(float64(c.Count())), nil
	}

	enum, ok := args[0].(value.Enumerable)
	if !ok {
		return nil, fmt.Errorf("length expects an enumerable, got %v", args[0])
	}

	ch, cancel := enum.Enumerate()
	defer cancel()

	var count int
	for range ch {
		count++
	}
	return value.NewNum(float64(count)), nil
}

func conjBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("conj expects at least 2 arguments, got %v", len(args))
	}

	conjer, ok := args[0].(value.Conjer)
	if !ok {
		return nil, fmt.Errorf("conj expects a conjer, got %v", args[0])
	}

	return conjer.Conj(args[1:]...), nil
}

func concatBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	enums := make([]value.Enumerable, len(args))
	for i, arg := range args {
		e, ok := arg.(value.Enumerable)
		if !ok {
			return nil, fmt.Errorf("concat arg %d is not enumerable: %v", i, arg)
		}
		enums[i] = e
	}

	enumerable := func() (<-chan value.Value, func()) {
		ch := make(chan value.Value)
		done := make(chan struct{})
		cancel := func() {
			close(done)
		}

		go func() {
			defer close(ch)
			for _, enum := range enums {
				select {
				case <-done:
					return
				default:
				}

				func() { // scope for defer
					eCh, eCancel := enum.Enumerate()
					defer eCancel()
					for v := range eCh {
						select {
						case ch <- v:
						case <-done:
							return
						}
					}
				}()
			}
		}()

		return ch, cancel
	}

	return &value.Seq{
		Enumerable: value.EnumerableFunc(enumerable),
	}, nil
}

func firstBuiltin(env value.Environment, args []value.Value) (out value.Value, err error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("first expects 1 argument, got %v", len(args))
	}

	if args[0] == nil {
		return nil, nil
	}

	switch c := args[0].(type) {
	case *value.List:
		if c.IsEmpty() {
			return value.NilValue, nil
		}
		return c.Item(), nil
	case *value.Vector:
		if c.Count() == 0 {
			return value.NilValue, nil
		}
		return c.ValueAt(0), nil
	}

	enum, ok := args[0].(value.Enumerable)
	if !ok {
		return nil, fmt.Errorf("first expects an enumerable, got %v", args[0])
	}

	itemCh, cancel := enum.Enumerate()
	defer cancel()

	return <-itemCh, nil
}

func restBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("rest expects 1 argument, got %v", len(args))
	}

	switch c := args[0].(type) {
	case *value.List:
		if c.IsEmpty() {
			return c, nil
		}
		return c.Next(), nil
	case *value.Vector:
		if c.Count() == 0 {
			return c, nil
		}
		return c.SubVector(1, c.Count()), nil
	}

	enum, ok := args[0].(value.Enumerable)
	if !ok {
		return nil, fmt.Errorf("rest expects an enumerable, got %v", args[0])
	}

	items := []value.Value{}
	itemCh, cancel := enum.Enumerate()
	defer cancel()

	// skip the first item
	<-itemCh
	for item := range itemCh {
		items = append(items, item)
	}

	// TODO: here and elsewhere, use a Sequence/Seq value type to
	// represent a lazy sequence of values, and use that instead of a
	// List/Vector.
	return value.NewList(items), nil
}

func subvecBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("subvec expects 2 or 3 arguments, got %v", len(args))
	}

	v, ok := args[0].(*value.Vector)
	if !ok {
		return nil, fmt.Errorf("subvec expects a vector as its first argument, got %v", args[0])
	}

	start, ok := args[1].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("subvec expects a number as its second argument, got %v", args[1])
	}

	startIdx := int(start.Value)
	endIdx := v.Count()

	if len(args) == 3 {
		end, ok := args[2].(*value.Num)
		if !ok {
			return nil, fmt.Errorf("subvec expects a number as its third argument, got %v", args[2])
		}
		endIdx = int(end.Value)
	}

	if startIdx < 0 || startIdx > v.Count() || endIdx < 0 || endIdx > v.Count() {
		return nil, fmt.Errorf("subvec indices out of bounds: %v %v", startIdx, endIdx)
	}

	return v.SubVector(startIdx, endIdx), nil
}

func notBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("not expects 1 argument, got %v", len(args))
	}
	switch arg := args[0].(type) {
	case *value.Bool:
		return value.NewBool(!arg.Value), nil
	default:
		return nil, fmt.Errorf("not expects a boolean, got %v", arg)
	}
}

func eqBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("eq? expects 2 arguments, got %v", len(args))
	}
	return value.NewBool(args[0].Equal(args[1])), nil
}

func isStringBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(*value.Str)
	return value.NewBool(ok), nil
}

func isListBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("list? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(*value.List)
	return value.NewBool(ok), nil
}

func isVectorBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(*value.Vector)
	return value.NewBool(ok), nil
}

func isSeqBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("seq? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(*value.Seq)
	return value.NewBool(ok), nil
}

func isSeqableBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("seqable? expects 1 argument, got %v", len(args))
	}
	_, ok := args[0].(value.Enumerable)
	return value.NewBool(ok), nil
}

func emptyBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("empty? expects 1 argument, got %v", len(args))
	}

	switch c := args[0].(type) {
	case *value.List:
		return value.NewBool(c.IsEmpty()), nil
	case *value.Vector:
		return value.NewBool(c.Count() == 0), nil
	}

	if c, ok := args[0].(value.Counter); ok {
		return value.NewBool(c.Count() == 0), nil
	}

	e, ok := args[0].(value.Enumerable)
	if !ok {
		return nil, fmt.Errorf("empty? expects an enumerable, got %v", args[0])
	}
	ch, cancel := e.Enumerate()
	defer cancel()
	// TODO: take a context.Context to support cancelation/timeout.
	_, ok = <-ch
	return value.NewBool(!ok), nil
}

func notEmptyBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	v, err := emptyBuiltin(env, args)
	if err != nil {
		return nil, err
	}
	return notBuiltin(env, []value.Value{v})
}

func powBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pow expects 2 arguments, got %v", len(args))
	}
	a, ok := args[0].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("pow expects a number, got %v", args[0])
	}
	b, ok := args[1].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("pow expects a number, got %v", args[1])
	}
	return value.NewNum(math.Pow(a.Value, b.Value)), nil
}

func floorBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor expects 1 argument, got %v", len(args))
	}
	a, ok := args[0].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("floor expects a number, got %v", args[0])
	}
	return value.NewNum(math.Floor(a.Value)), nil
}

func mulBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	var res float64 = 1
	for _, arg := range args {
		switch arg := arg.(type) {
		case *value.Num:
			res *= arg.Value
		default:
			return nil, fmt.Errorf("invalid type for *: %v", arg)
		}
	}

	return value.NewNum(res), nil
}

func divBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("div expects 2 arguments, got %v", len(args))
	}
	num, ok := args[0].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("div expects a number as the first argument, got %v", args[0])
	}
	denom, ok := args[1].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("div expects a number as the second argument, got %v", args[1])
	}
	// TODO: handle generators
	return value.NewNum(num.Value / denom.Value), nil
}

func addBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	var sum float64 = 0

	// sum all number arguments together
	for _, arg := range args {
		switch arg := arg.(type) {
		case *value.Num:
			sum += arg.Value
		default:
			return nil, fmt.Errorf("invalid type for +: %v", arg)
		}
	}

	return value.NewNum(sum), nil
}

func subBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("sub expects 2 arguments, got %v", len(args))
	}
	a, ok := args[0].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("sub expects a number as the first argument, got %v", args[0])
	}
	b, ok := args[1].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("sub expects a number as the second argument, got %v", args[1])
	}

	// TODO: handle generators
	return value.NewNum(a.Value - b.Value), nil
}

func ltBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("< expects 2 arguments, got %v", len(args))
	}
	a, ok := args[0].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("< expects a number as the first argument, got %v", args[0])
	}
	b, ok := args[1].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("< expects a number as the second argument, got %v", args[1])
	}

	return value.NewBool(a.Value < b.Value), nil
}

func gtBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("> expects 2 arguments, got %v", len(args))
	}
	a, ok := args[0].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("> expects a number as the first argument, got %v", args[0])
	}
	b, ok := args[1].(*value.Num)
	if !ok {
		return nil, fmt.Errorf("> expects a number as the second argument, got %v", args[1])
	}

	return value.NewBool(a.Value > b.Value), nil
}

func applyBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("apply expects 2 arguments, got %v", len(args))
	}
	// the first argument should be an applyer, the second an enumerable
	applyer, ok := args[0].(value.Applyer)
	if !ok {
		return nil, fmt.Errorf("apply expects a function as the first argument, got %v", args[0])
	}

	var values []value.Value

	if !value.NilValue.Equal(args[1]) {
		enum, ok := args[1].(value.Enumerable)
		if !ok {
			return nil, fmt.Errorf("apply expects an enumerable as the second argument, got %v", args[1])
		}
		var err error
		values, err = value.EnumerateAll(context.Background(), enum)
		if err != nil {
			return nil, err
		}
	}

	return applyer.Apply(env, values)
}

func printlnBuiltin(env value.Environment, args []value.Value) (value.Value, error) {
	for i, arg := range args {
		if arg == nil {
			// TODO: this should be an error, nil is represented by
			// *value.Nil
			env.Stdout().Write([]byte("nil"))
		} else {
			fmt.Println("converting?", arg)
			switch arg := value.ConvertFromGo(arg).(type) {
			case *value.Str:
				env.Stdout().Write([]byte(arg.Value))
			case *value.Char:
				env.Stdout().Write([]byte(string(arg.Value)))
			default:
				env.Stdout().Write([]byte(arg.String()))
			}
		}
		if i < len(args)-1 {
			env.Stdout().Write([]byte(" "))
		}
	}
	env.Stdout().Write([]byte("\n"))
	return nil, nil
}
